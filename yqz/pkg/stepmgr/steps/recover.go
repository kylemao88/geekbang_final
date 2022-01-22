//
package steps

import (
	"sync"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
)

const STEP_RECOVER_QUEUE string = "yqz:stepmgr:{recover}"
const STEP_MIN_DATE = "1970-01-16"

const (
	SCRIPT_PUSH = `
local q = KEYS[1]
local q_set = KEYS[1] .. "_set"
local v = redis.call("SADD", q_set, ARGV[1])
if v == 1
then
	return redis.call("RPUSH", q, ARGV[1]) and 1
else
	return 0
end
`

	SCRIPT_POP = `
local q = KEYS[1]
local q_set = KEYS[1] .. "_set"
local v = redis.call("LPOP", q)
if (v==nil or (type(v) == "boolean" and not v))
then
	return ""
else
	redis.call("SREM", q_set, v)
	return v
end
`
)

// for recover
var SpeedRecover int64 = 0

func runRecoverWorker(cnt, interval int) error {
	// run recover worker
	wg := &sync.WaitGroup{}
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func(index int) {
			logger.Sys("RecoverWorker[%v] is start", index)
			wg.Done()
			t := time.NewTimer(time.Duration(interval) * time.Millisecond)
			for {
				select {
				case <-stopChan:
					logger.Sys("RecoverWorker[%v] is stop", index)
					t.Stop()
					return
				case <-t.C:
					user, err := popRecoverUser()
					if err == nil && len(user) != 0 {
						logger.Debug("RecoverWorker[%v] handle user: %v recover step", index, user)
						var usersStep map[string]*metadata.UserSteps
						usersStep, err = getUsersMonthStepsFromDB([]string{user})
						if err != nil && !errors.IsStepMgrIncompleteError(err) {
							logger.Error("RecoverWorker[%v] - err: %v", index, err)
						} else {
							// only available in recover one user
							if errors.IsStepMgrIncompleteError(err) {
								usersStep = make(map[string]*metadata.UserSteps)
								steps := make(map[string]int32)
								steps[STEP_MIN_DATE] = 0
								usersStep[user] = &metadata.UserSteps{
									Steps: steps,
								}
							}
							err = SetUsersStepsToCache(usersStep, true)
							if err != nil {
								logger.Error("RecoverWorker[%v] - err: %v", index, err)
							}
						}
					}
					// if err or no data, sleep
					if err != nil {
						logger.Error("RecoverWorker[%v] - err: %v", index, err)
						//time.Sleep(time.Duration(interval) * time.Millisecond)
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else if len(user) == 0 {
						//logger.Debug("RecoverWorker[%v] - no record in queue", index)
						//time.Sleep(time.Duration(interval) * time.Millisecond)
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else {
						atomic.AddInt64(&SpeedRecover, 1) //加操作
						t.Reset(0)
					}
				}
			}
		}(i)
	}
	wg.Wait()
	return nil
}

func closeRecoverWorker() {
	close(stopChan)
}

// pushRecoverUser 0: push failed 1: push success
func pushRecoverUser(userID string) (int64, error) {
	result, err := redisClient.Eval(SCRIPT_PUSH, []string{STEP_RECOVER_QUEUE}, []string{userID}).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push error, error = %v, key = %s", util.GetCallee(), err, STEP_RECOVER_QUEUE))
	}
	return result.(int64), err
}

// popRecoverUser return userID, error
func popRecoverUser() (string, error) {
	result, err := redisClient.Eval(SCRIPT_POP, []string{STEP_RECOVER_QUEUE}, []string{""}).Result()
	if err != nil {
		return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop error, error = %v, key = %s", util.GetCallee(), err, STEP_RECOVER_QUEUE))
	}
	return result.(string), nil
}

// QueueLength get queue length
func QueueLength(queue string) (resp int64, err error) {
	result, err := redisClient.LLen(queue).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis get len error, error = %v, key = %s", util.GetCallee(), err, queue))
	}
	return result, err
}

// queueClear clear queue
func QueueClear(queue string) error {
	_, err := redisClient.Del(queue).Result()
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("%v - redis clear error, error = %v, key = %s", util.GetCallee(), err, queue))
	}
	_, err = redisClient.Del(getQueueSet(queue)).Result()
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("%v - redis clear error, error = %v, key = %s", util.GetCallee(), err, getQueueSet(queue)))
	}
	return nil
}

func getQueueSet(queue string) string {
	return queue + "_set"
}

// getUsersMonthStepsFromDB ...
func getUsersMonthStepsFromDB(userIDs []string) (map[string]*metadata.UserSteps, error) {
	// cdb get info
	currentDate := util.GetLocalFormatDate()
	deltaDate, err := util.GetLocalDeltaFormatDate("-720h")
	if err != nil {
		return nil, errors.NewInternalError(errors.WithMsg("delta format err: %v", err))
	}
	logger.Info("recover user: %v month steps from db", userIDs)
	return GetUsersStepsByDateRangeFromDB(userIDs, deltaDate, currentDate)
}
