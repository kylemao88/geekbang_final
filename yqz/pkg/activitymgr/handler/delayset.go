//
package handler

import (
	"sync"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"google.golang.org/protobuf/proto"
)

const STEP_UPDATE_QUEUE string = "{yqz:activitymgr:updatestep}"
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

var redisClient *gyredis.RedisClient
var stopSetChan chan struct{}

// InitStepComponent ...
func InitStepComponent(redisCli *gyredis.RedisClient) error {
	logger.Sys("Try InitStepComponent...")
	stopSetChan = make(chan struct{})
	redisClient = redisCli
	err := runSetWorker(4, 100)
	if err != nil {
		return err
	}
	return err
}

// CloseStepComponent ...
func CloseStepComponent() {
	logger.Sys("Try CloseStepComponent...")
	closeSetWorker()
	time.Sleep(1 * time.Second)
}

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
							err = updateActivityRankStats(userStep.Oid, userStep)
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
	result, err := redisClient.Eval(SCRIPT_PUSH, []string{STEP_UPDATE_QUEUE}, []string{string(userString)}).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push error, error = %v, key = %s", util.GetCallee(), err, STEP_UPDATE_QUEUE))
	}
	return result.(int64), err
}

// popSetUser return userID, error
func popSetUser() (string, error) {
	result, err := redisClient.Eval(SCRIPT_POP, []string{STEP_UPDATE_QUEUE}, []string{""}).Result()
	if err != nil {
		return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop error, error = %v, key = %s", util.GetCallee(), err, STEP_UPDATE_QUEUE))
	}
	return result.(string), nil
}
