//
//
package cache_access

import (
	"fmt"
	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-redis/redis"
	"time"
)

func SetSendMsg(oid, date string, diff int64) error {
	key := fmt.Sprintf("yqz_send_week_msg_%s_%s", oid, date)
	_, err := RedisClient.Set(key, 1, time.Duration(diff)*time.Second).Result()
	if err != nil {
		log.Error("redis Set error, err = %v", err)
		return err
	}
	return nil
}

func IsSendMsg(oid, date string) (bool, error) {
	key := fmt.Sprintf("yqz_send_week_msg_%s_%s", oid, date)
	_, err := RedisClient.Get(key).Result()
	if err == redis.Nil {
		log.Info("redis Get key = %s, not exist", key)
		return false, nil
	}

	if err != nil && err != redis.Nil {
		log.Error("redis client Get error, err = %v, key = %s", err, key)
		return false, err
	}

	return true, nil
}
