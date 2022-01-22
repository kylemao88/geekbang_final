//
package user

import (
	"fmt"
	"strconv"

	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"github.com/go-redis/redis"
)

const (
	// user profile cache, use hashmap
	YQZ_USER_PROFILE_KEY            = "YQZ_USER_PROFILE"
	YQZ_USER_PROFILE_YQZID          = "PROFILE_YQZID"
	YQZ_USER_PROFILE_SEND_LEAF      = "PROFILE_SEND_LEAF"
	YQZ_USER_PROFILE_DONATE_FLAG    = "PROFILE_DONATE_FLAG"
	YQZ_USER_PROFILE_TEAM_FLAG      = "PROFILE_TEAM_FLAG"
	YQZ_USER_PROFILE_AUTO_STEP_FLAG = "PROFILE_AUTO_STEP_FLAG"
	YQZ_USER_PROFILE_AUTO_STEP_TIME = "PROFILE_AUTO_STEP_TIME"
)

type UserProfile struct {
	Unin     string
	Leaf     int
	Donated  int
	Joined   int
	AutoFlag int
	AutoTime string
}

func FormatUserProfileKey(oid string) string {
	return fmt.Sprintf("%v:%v", YQZ_USER_PROFILE_KEY, oid)
}

func ExistUserProfile(redisCli *gyredis.RedisClient, oid string) (bool, error) {
	key := FormatUserProfileKey(oid)
	status, err := redisCli.Exists(key).Result()
	if err == redis.Nil {
		logger.Info("redis Get key = %s not exist", key)
		return false, nil
	}
	if err != nil && err != redis.Nil {
		logger.Error("redis HGet error, error = %v, key = %s", err, key)
		return false, err
	}
	if status > 0 {
		return true, nil
	}
	return false, nil
}

func AddUserProfile(redisCli *gyredis.RedisClient, oid, unin string, leaf, donated, joined int, autoFlag int, autoTime string) error {
	contextKey := FormatUserProfileKey(oid)
	value := make(map[string]interface{})
	value[YQZ_USER_PROFILE_YQZID] = unin
	value[YQZ_USER_PROFILE_SEND_LEAF] = leaf
	value[YQZ_USER_PROFILE_DONATE_FLAG] = donated
	value[YQZ_USER_PROFILE_TEAM_FLAG] = joined
	value[YQZ_USER_PROFILE_AUTO_STEP_FLAG] = autoFlag
	value[YQZ_USER_PROFILE_AUTO_STEP_TIME] = autoTime
	_, err := redisCli.HMSet(contextKey, value).Result()
	if err != nil {
		logger.Error("redis HMSet context key: %v value: %v, error: %v", contextKey, value, err)
		return err
	}
	logger.Debug("redis HMSet context key: %v value: %v success", contextKey, value)
	return nil
}

func GetUserProfile(redisCli *gyredis.RedisClient, oid string) (*UserProfile, error) {
	result := &UserProfile{}
	key := FormatUserProfileKey(oid)
	fields := make([]string, 0)
	fields = append(fields, YQZ_USER_PROFILE_YQZID, YQZ_USER_PROFILE_SEND_LEAF, YQZ_USER_PROFILE_DONATE_FLAG,
		YQZ_USER_PROFILE_TEAM_FLAG, YQZ_USER_PROFILE_AUTO_STEP_FLAG, YQZ_USER_PROFILE_AUTO_STEP_TIME)
	var finished bool
	redisResult, err := redisCli.HMGet(key, fields...).Result()
	if err != nil {
		if err != redis.Nil {
			logger.Error("redis HMGet key: %s, error: %v", key, err)
			return nil, err
		}
	} else if err == redis.Nil || len(redisResult) != len(fields) {
		logger.Error("redis HMGet key: %s, error:%v result=%v, len(field)=%v ", key, err, len(redisResult), len(fields))
		return nil, redis.Nil
	} else {
		logger.Debug("redis user profile: %v", redisResult)
	loop:
		for index, value := range redisResult {
			if value == nil {
				logger.Error("unExpect index: %v nil value", index)
				break
			}
			switch index {
			case 0:
				result.Unin = value.(string)
			case 1:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					logger.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Leaf = ret
			case 2:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					logger.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Donated = ret
			case 3:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					logger.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Joined = ret
			case 4:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					logger.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.AutoFlag = ret
			case 5:
				result.AutoTime = redisResult[5].(string)
				finished = true
			default:
				logger.Error("out of range: %v", index)
			}
		}
	}
	if !finished {
		logger.Error("redis HMGet key: %s, error: not complete field", key)
		return nil, fmt.Errorf("redis HMGet key: %s, error: not complete field", key)
	}
	return result, nil
}
