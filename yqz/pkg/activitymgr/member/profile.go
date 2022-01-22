package member

import (
	"fmt"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"github.com/go-redis/redis"
	"strconv"
)

const (
	YQZ_USER_PROFILE_KEY        = "YQZ_USER_PROFILE"
	YQZ_USER_PROFILE_HAS_CREATE = "PROFILE_HAS_CREATE" // 是否创建过活动
	YQZ_USER_PROFILE_PHONE_NUM  = "PROFILE_PHONE_NUM"  // 电话号码
)

func FormatUserProfileKey(oid string) string {
	return fmt.Sprintf("%v:%v", YQZ_USER_PROFILE_KEY, oid)
}

func SetProfileHasCreate(oid string, has bool) error {
	key := FormatUserProfileKey(oid)
	_, err := cacheclient.RedisClient.HSet(key, YQZ_USER_PROFILE_HAS_CREATE, has).Result()
	if err != nil {
		logger.Error("HSet err = %v", err)
		return err
	}
	return nil
}

func GetProfileHasCreate(oid string) (bool, error) {
	key := FormatUserProfileKey(oid)
	val, err := cacheclient.RedisClient.HGet(key, YQZ_USER_PROFILE_HAS_CREATE).Result()
	if err != nil {
		logger.Error("HGet err = %v", err)
		return false, err
	}
	if len(val) == 0 {
		return false, redis.Nil
	}
	return strconv.ParseBool(val)
}

func SetProfilePhoneNum(oid, phoneNum string) error {
	key := FormatUserProfileKey(oid)
	_, err := cacheclient.RedisClient.HSet(key, YQZ_USER_PROFILE_PHONE_NUM, phoneNum).Result()
	if err != nil {
		logger.Error("HSet err = %v", err)
		return err
	}
	return nil
}

