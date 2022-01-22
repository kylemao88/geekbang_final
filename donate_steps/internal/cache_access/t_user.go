//
package cache_access

import (
	"fmt"
	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-redis/redis"
	"strconv"
)

const (
	userStatus = "yqz:user:status:"
)

func DeleteUserStatus(oid string) error {
	key := fmt.Sprintf("%s_%s", userStatus, oid)
	_, err := RedisClient.Expire(key, 0).Result()
	if err != nil {
		log.Error("redis expire error, err = %v, key = %s", err, key)
		return err
	}
	return nil
}

func AddUserProfile(oid, unin string, leaf, donated, joined int, autoFlag int, autoTime string) error {
	contextKey := FormatUserProfileKey(oid)
	value := make(map[string]interface{})
	value[YQZ_USER_PROFILE_YQZID] = unin
	value[YQZ_USER_PROFILE_SEND_LEAF] = leaf
	value[YQZ_USER_PROFILE_DONATE_FLAG] = donated
	value[YQZ_USER_PROFILE_TEAM_FLAG] = joined
	value[YQZ_USER_PROFILE_AUTO_STEP_FLAG] = autoFlag
	value[YQZ_USER_PROFILE_AUTO_STEP_TIME] = autoTime
	_, err := RedisClient.HMSet(contextKey, value).Result()
	if err != nil {
		log.Error("redis HMSet context key: %v value: %v, error: %v", contextKey, value, err)
		return err
	}
	log.Debug("redis HMSet context key: %v value: %v success", contextKey, value)
	return nil
}

func GetUserProfile(oid string) (*UserProfile, error) {
	result := &UserProfile{}
	key := FormatUserProfileKey(oid)
	fields := make([]string, 0)
	fields = append(fields, YQZ_USER_PROFILE_YQZID, YQZ_USER_PROFILE_SEND_LEAF, YQZ_USER_PROFILE_DONATE_FLAG,
		YQZ_USER_PROFILE_TEAM_FLAG, YQZ_USER_PROFILE_AUTO_STEP_FLAG, YQZ_USER_PROFILE_AUTO_STEP_TIME)
	var finished bool
	redisResult, err := RedisClient.HMGet(key, fields...).Result()
	if err != nil {
		if err != redis.Nil {
			log.Error("redis HMGet key: %s, error: %v", key, err)
			return nil, err
		}
	} else if err == redis.Nil || len(redisResult) != len(fields) {
		log.Error("redis HMGet key: %s, error:%v result=%v, len(field)=%v ", key, err, len(redisResult), len(fields))
		return nil, redis.Nil
	} else {
		log.Debug("redis user: %v, profile: %v", oid, redisResult)
	loop:
		for index, value := range redisResult {
			if value == nil {
				log.Error("unExpect index: %v nil value", index)
				break
			}
			switch index {
			case 0:
				result.Unin = value.(string)
			case 1:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					log.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Leaf = ret
			case 2:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					log.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Donated = ret
			case 3:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					log.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.Joined = ret
			case 4:
				ret, err := strconv.Atoi(value.(string))
				if err != nil {
					log.Error("strconv.Atoi error, index: %v, error = %v, val = %s", index, err, value)
					break loop
				}
				result.AutoFlag = ret
			case 5:
				result.AutoTime = redisResult[5].(string)
				finished = true
			default:
				log.Error("out of range: %v", index)
			}
		}
	}
	if !finished {
		log.Error("redis HMGet key: %s, error: not complete field", key)
		return nil, fmt.Errorf("redis HMGet key: %s, error: not complete field", key)
	}
	return result, nil
}

func UpdateUserProfileDonate(oid string, donated int) error {
	contextKey := FormatUserProfileKey(oid)
	_, err := RedisClient.HSet(contextKey, YQZ_USER_PROFILE_DONATE_FLAG, int64(donated)).Result()
	if err != nil {
		log.Error("redis HMSet context key: %v value: %v, error: %v", contextKey, donated, err)
		return err
	}
	log.Debug("redis HMSet context key: %v value: %v", contextKey, donated)
	return nil
}

