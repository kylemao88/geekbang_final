package business_access

import (
	"fmt"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"github.com/go-redis/redis"
)

const (
	YQZ_MSG_OPEN_KEY = "yqz:msg_open_stat"
	MONTH_SECOND     = 60 * 60 * 24 * 30
)

func GetMsgStat(id, oid string, msgType int) (data_access.MsgStat, error) {
	ms, err := data_access.GetMsgStat(id, oid, msgType)
	if err != nil {
		log.Error("GetMsgStat error, err = %v", err)
		return ms, err
	}
	return ms, nil
}

func GetMsgStatList(id string, oids []string, msgType int) ([]data_access.MsgStat, error) {
	mss, err := data_access.GetMsgStatList(id, oids, msgType)
	if err != nil {
		log.Error("GetMsgStatList error, err = %v", err)
		return mss, err
	}
	return mss, nil
}

func UpdateMsgStatStatus(db_proxy data_access.DBProxy, id, oid string, msgType, status int) error {
	if err := data_access.UpdateMsgStatStatus(db_proxy, id, oid, msgType, status); err != nil {
		log.Error("UpdateMsgStatStatus error, err = %v", err)
		return err
	}
	return nil
}

func InsertMsgStat(db_proxy data_access.DBProxy, ms data_access.MsgStat) error {
	if err := data_access.InsertMsgStat(db_proxy, ms); err != nil {
		log.Error("InsertMsgStat error, err = %v, msgStat = %v", err, ms)
		return err
	}
	return nil
}

func InsertMsgStatList(db_proxy data_access.DBProxy, mss []data_access.MsgStat) error {
	if err := data_access.InsertMsgStatList(db_proxy, mss); err != nil {
		log.Error("InsertMsgStatList error, err = %v, msgStatList = %v", err, mss)
		return err
	}
	return nil
}

func AddUserMsgStat(oid, week string) error {
	key := formatUserMsgStatKey(week)
	_, err := cache_access.RedisClient.ZAdd(key, redis.Z{Member: oid, Score: float64(util.GetLocalTime().Unix())}).Result()
	if err != nil {
		log.Error("redis client add member error: %v, key: %v, member: %v", err, key, oid)
		return err
	}
	// refresh expire time 1 month
	cache_access.RedisClient.Expire(key, time.Second*MONTH_SECOND).Result()
	log.Debug("add key: %v, member: %v", key, oid)
	return nil
}

func formatUserMsgStatKey(week string) string {
	return fmt.Sprintf("%v:%v", YQZ_MSG_OPEN_KEY, week)
}
