//
package common

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
)

const (
	UNIQUE_QUEUE string = "yqz:stepmgr:{delay_unique}"
	NORMAL_QUEUE string = "yqz:stepmgr:{delay}"

	SCRIPT_PUSH_UNIQ = `
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

	SCRIPT_POP_UNIQ = `
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

var (
	redisClient *gyredis.RedisClient
	stopChan    chan struct{}
	funcHandler = make(map[string](func(string) error))
)

type DelaySetElem struct {
	Header  string
	Element string
}

func (m *DelaySetElem) String() string {
	return fmt.Sprintf("{header: %v, element: %v}", m.Header, m.Element)
}

func RunDelaySetWorker(cnt, interval int, cli *gyredis.RedisClient) error {
	stopChan = make(chan struct{})
	redisClient = cli
	wg := &sync.WaitGroup{}
	for i := 0; i < cnt; i++ {
		wg.Add(1)
		go func(index int) {
			logger.Sys("DelaySetWorker[%v] is start", index)
			wg.Done()
			t := time.NewTimer(time.Duration(interval) * time.Millisecond)
			for {
				select {
				case <-stopChan:
					logger.Sys("DelaySetWorker[%v] is stop", index)
					t.Stop()
					return
				case <-t.C:
					element, err := PopElement()
					if err == nil && len(element) != 0 {
						elementHandler(element)
					}
					// if err or no data, sleep
					if err != nil {
						logger.Error("DelaySetWorker[%v] - err: %v", index, err)
						t.Reset(time.Duration(interval) * time.Millisecond)
					} else if len(element) == 0 {
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

func elementHandler(msg string) error {
	var element = &DelaySetElem{}
	err := json.Unmarshal([]byte(msg), element)
	if err != nil {
		return errors.NewJsonParseError(errors.WithMsg("json.UnMarshal msg: %+v, error: %v", msg, err))
	}
	key := element.Header
	handler, ok := funcHandler[key]
	if !ok {
		return errors.NewFuncRegisterError(errors.WithMsg("no key: %v handle func, please register it", key))
	}
	return handler(element.Element)
}

// CloseCouponsWorker close coupons workers...
func CloseDelaySetWorker() {
	close(stopChan)
}

// RegisterHandler for func
func RegisterHandler(key string, handler func(msg string) error) {
	funcHandler[key] = handler
}

// PushElement ...
func PushElement(element *DelaySetElem) (int64, error) {
	msg, err := json.Marshal(element)
	if err != nil {
		return 0, errors.NewJsonParseError(errors.WithMsg("json.Marshal element: %+v, error: %v", element, err))
	}
	_, err = redisClient.RPush(NORMAL_QUEUE, msg).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push unique queue error: %v, key: %s", util.GetCallee(), err, NORMAL_QUEUE))
	}
	return 1, err
}

// PopElement return element, error
func PopElement() (string, error) {
	result, err := redisClient.LPop(NORMAL_QUEUE).Result()
	if err != nil {
		if err != redis.Nil {
			return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop unique queue error: %v, key: %s", util.GetCallee(), err, NORMAL_QUEUE))
		}
		return "", nil
	}
	return result, nil
}

// PushUniqElement 0: push failed 1: push success
func PushUniqElement(element *DelaySetElem) (int64, error) {
	msg, err := json.Marshal(element)
	if err != nil {
		return 0, errors.NewJsonParseError(errors.WithMsg("json.Marshal element: %+v, error: %v", element, err))
	}
	result, err := redisClient.Eval(SCRIPT_PUSH_UNIQ, []string{UNIQUE_QUEUE}, []string{string(msg)}).Result()
	if err != nil {
		return 0, errors.NewRedisClientError(errors.WithMsg("%v - redis push unique queue error: %v, key: %s", util.GetCallee(), err, UNIQUE_QUEUE))
	}
	return result.(int64), err
}

// PopUniqElement return element, error
func PopUniqElement() (string, error) {
	result, err := redisClient.Eval(SCRIPT_POP_UNIQ, []string{UNIQUE_QUEUE}, []string{""}).Result()
	if err != nil {
		return "", errors.NewRedisClientError(errors.WithMsg("%v - redis pop unique queue error: %v, key: %s", util.GetCallee(), err, UNIQUE_QUEUE))
	}
	return result.(string), nil
}
