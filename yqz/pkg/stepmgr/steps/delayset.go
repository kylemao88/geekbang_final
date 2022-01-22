//
package steps

import (
	"sync"
	"time"

	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"google.golang.org/protobuf/proto"
)

const STEP_SET_QUEUE string = "yqz:stepmgr:{stepDB}"

func runSetWorker(cnt, interval int) error {
	// run recover worker
	wg := &sync.WaitGroup{}
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func(index int) {
			logger.Sys("SetWorker[%v] is start", index)
			wg.Done()
			t := time.NewTimer(time.Duration(interval) * time.Millisecond)
			for {
				select {
				case <-stopSetChan:
					logger.Sys("SetWorker[%v] is stop", index)
					t.Stop()
					return
				case <-t.C:
					user, err := popSetUser()
					if err == nil && len(user) != 0 {
						userStep := &metadata.UserSteps{}
						err = proto.Unmarshal([]byte(user), userStep)
						if err != nil {
							logger.Error("proto.Unmarshal error: %v", err)
						} else {
							var datas = make(map[string]*metadata.UserSteps)
							datas[userStep.Oid] = userStep
							err = SetUsersStepsToDB(datas, false)
							if err != nil {
								logger.Error("SetWorker[%v] - err: %v", index, err)
							} else {
								logger.Debug("SetWorker[%v] handle user: %v set step success", index, userStep.Oid)
							}
						}
					}
					// if err or no data, sleep
					if err != nil {
						logger.Error("SetWorker[%v] - err: %v", index, err)
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else if len(user) == 0 {
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else {
						t.Reset(0)
					}
				}
			}
		}(i)
	}
	wg.Wait()
	return nil
}

func closeSetWorker() {
	close(stopSetChan)
}

// pushRecoverUser 0: push failed 1: push success
func pushSetUser(user *metadata.UserSteps) (int64, error) {
	userString, err := proto.Marshal(user)
	if err != nil {
		logger.Error("proto.Marshal user: %v, error: %v", user, err)
		return -1, err
	}
	result, err := redisClient.Eval(SCRIPT_PUSH, []string{STEP_SET_QUEUE}, []string{string(userString)}).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push error, error = %v, key = %s", util.GetCallee(), err, STEP_SET_QUEUE))
	}
	return result.(int64), err
}

// popSetUser return userID, error
func popSetUser() (string, error) {
	result, err := redisClient.Eval(SCRIPT_POP, []string{STEP_SET_QUEUE}, []string{""}).Result()
	if err != nil {
		return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop error, error = %v, key = %s", util.GetCallee(), err, STEP_SET_QUEUE))
	}
	return result.(string), nil
}
