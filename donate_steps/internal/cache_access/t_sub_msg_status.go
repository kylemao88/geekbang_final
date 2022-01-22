//
//

package cache_access

import (
	"strconv"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"github.com/go-redis/redis"
)

func GetUserSubMsgStatus(oid string) ([]data_access.SubMsgStatus, error) {
	var result []data_access.SubMsgStatus
	key := FormatUserSubKey(oid)
	redisResult, err := RedisClient.HGetAll(key).Result()
	if err != nil {
		if err != redis.Nil {
			log.Error("redis HMGet key: %s, error: %v", key, err)
			return result, err
		}
	} else if err == redis.Nil || len(redisResult) == 0 {
		log.Error("redis HMGet key: %s, error:%v ", key, err)
		return result, redis.Nil
	} else {
		log.Debug("redis user msg: %v", redisResult)
		for key, value := range redisResult {
			status, err := strconv.Atoi(value)
			if err != nil {
				log.Error("Atoi msgID: %v status: %v error: %v", key, value, err)
				continue
			}
			result = append(result, data_access.SubMsgStatus{
				MsgTempleId: key,
				Status:      status,
			})
		}
	}
	return result, nil
}

func GetUserSpecSubMsgStatus(oid string, msgIDs []string) ([]data_access.SubMsgStatus, error) {
	var result []data_access.SubMsgStatus
	key := FormatUserSubKey(oid)
	fields := make([]string, 0)
	fields = append(fields, msgIDs...)
	redisResult, err := RedisClient.HMGet(key, fields...).Result()
	if err != nil {
		if err != redis.Nil {
			log.Error("redis HMGet key: %s, error: %v", key, err)
			return result, err
		}
	} else if err == redis.Nil || len(redisResult) == 0 {
		log.Error("redis HMGet key: %s, error:%v result=%v, len(field)=%v ", key, err, len(redisResult), len(fields))
		return result, redis.Nil
	} else {
		log.Debug("redis user profile: %v", redisResult)
		for index, value := range redisResult {
			if value == nil {
				log.Info("msgID: %v nil value", msgIDs[index])
				continue
			}
			status, _ := strconv.Atoi(value.(string))
			result = append(result, data_access.SubMsgStatus{
				UserId:      oid,
				MsgTempleId: msgIDs[index],
				Status:      status,
			})
		}
	}
	return result, nil
}

func SetSubMsgStatus(oid, msgID string, status int) error {
	key := FormatUserSubKey(oid)
	_, err := RedisClient.HSet(key, msgID, status).Result()
	if err != nil {
		log.Error("redis HMSet key: %v field: %v value: %v, error: %v",
			key, msgID, status, err)
		return err
	}
	return nil
}
