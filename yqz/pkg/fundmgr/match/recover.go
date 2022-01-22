//
package match

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
)

const FUNDMGR_RECOVER_QUEUE string = "yqz:{fundmgr:recover}"
const ACTIVITY_PREFIX string = "ca_"
const YQZ_MATCH_PREFIX string = "yqzMatch_"

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
var redisClient *gyredis.RedisClient

func RunRecoverWorker(cnt, interval int, cli *gyredis.RedisClient) {
	// run recover worker
	redisClient = cli
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
					element, err := popRecoverElement()
					if err == nil && len(element) != 0 {
						err = recoverHandle(index, element)
					}
					// if err or no data, sleep
					if err != nil {
						logger.Error("RecoverWorker[%v] - err: %v", index, err)
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else if len(element) == 0 {
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
}

func CloseRecoverWorker() {
	close(stopChan)
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

// recoverHandle logical ops write here
func recoverHandle(index int, element string) error {
	if strings.HasPrefix(element, ACTIVITY_PREFIX) {
		logger.Debug("RecoverWorker[%v] handle elements: %v recover activity match rank", index, element)
		err := RecoverActivityMatchRank(element)
		if err != nil {
			logger.Info("RecoverWorker[%v] - activity: %v error: %v", index, element, err)
			return err
		} else {
			logger.Info("RecoverWorker[%v] - recover activity: %v success", index, element)
		}
	} else if strings.HasPrefix(element, YQZ_MATCH_PREFIX) {
		logger.Debug("RecoverWorker[%v] handle elements: %v recover yqz match state", index, element)
		realID := element[len(YQZ_MATCH_PREFIX):]
		// TODO
		err := RecoverYQZMatchState(realID)
		if err != nil {
			logger.Info("RecoverWorker[%v] - recover match stat: %v error: %v", index, realID, err)
			return err
		} else {
			logger.Info("RecoverWorker[%v] - recover match stat: %v success", index, realID)
		}
	} else if strings.HasPrefix(element, COMPANY_RANK_KEY_PREFIX) {
		logger.Debug("RecoverWorker[%v] handle elements: %v recover company rank", index, element)
		err := RecoverCompanyRank()
		if err != nil {
			logger.Info("RecoverWorker[%v] - recover company rank error: %v", index, err)
			return err
		} else {
			logger.Info("RecoverWorker[%v] - recover company rank success", index)
		}
	} else {
		logger.Error("RecoverWorker[%v] handle suspicious elements(%v) recover comment", index, element)
	}
	return nil
}

// pushRecoverElement 0: push failed 1: push success
func pushRecoverElement(commentID string) (int64, error) {
	result, err := redisClient.Eval(SCRIPT_PUSH, []string{FUNDMGR_RECOVER_QUEUE}, []string{commentID}).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push error, error = %v, key = %s", util.GetCallee(), err, FUNDMGR_RECOVER_QUEUE))
	}
	return result.(int64), err
}

// popRecoverElement return element, error
func popRecoverElement() (string, error) {
	result, err := redisClient.Eval(SCRIPT_POP, []string{FUNDMGR_RECOVER_QUEUE}, []string{""}).Result()
	if err != nil {
		return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop error, error = %v, key = %s", util.GetCallee(), err, FUNDMGR_RECOVER_QUEUE))
	}
	return result.(string), nil
}

func getQueueSet(queue string) string {
	return queue + "_set"
}
