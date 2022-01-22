//
//

package cache_access

import (
	"fmt"
	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-redis/redis"
	"time"
)

func SetPoiMsgSend(oid, week string, poiIndex int, diff int64) error {
	key := fmt.Sprintf("yqz_poi_msg_send_%s_%s_%d", oid, week, poiIndex)
	_, err := RedisClient.Set(key, 1, time.Duration(diff)*time.Second).Result()
	if err != nil {
		log.Error("redis Set error, err = %v", err)
		return err
	}
	return nil
}

func IsPoiMsgSend(oid, week string, poiIndex int) (bool, error) {
	key := fmt.Sprintf("yqz_poi_msg_send_%s_%s_%d", oid, week, poiIndex)
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
