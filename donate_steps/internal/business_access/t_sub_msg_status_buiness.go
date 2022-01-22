package business_access

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"github.com/go-redis/redis"
)

func GetSubMsgStatus(oid string) ([]data_access.SubMsgStatus, error) {
	var err error
	ret, cacheErr := cache_access.GetUserSubMsgStatus(oid)
	if cacheErr != nil || len(ret) == 0 {
		log.Error("cache_access.GetUserSubMsgStatus oid: %v err: %v", oid, cacheErr)
		ret, err = data_access.BatchGetSubMsgStatus(oid)
		if err != nil {
			log.Error("data_access.BatchGetSubMsgStatus err = %v, oid = %v", err, oid)
			return nil, err
		}
		if cacheErr == redis.Nil {
			for _, value := range ret {
				err = cache_access.SetSubMsgStatus(value.UserId, value.MsgTempleId, value.Status)
				if err != nil {
					log.Error("cache_access.SetSubMsgStatus set oid: %v, msgID: %v, status: %v",
						value.UserId, value.MsgTempleId, value.Status)
				}
			}
		}
	}
	return ret, nil
}

func GetSubMsgStatusById(oid string, msgIds []string) ([]data_access.SubMsgStatus, error) {
	var finalRet []data_access.SubMsgStatus
	ret, cacheErr := cache_access.GetUserSpecSubMsgStatus(oid, msgIds)
	if cacheErr != nil || len(ret) == 0 {
		log.Error("cache_access.GetUserSpecSubMsgStatus oid: %v err: %v", oid, cacheErr)
		ret, err := data_access.BatchGetSubMsgStatus(oid)
		if err != nil {
			log.Error("data_access.BatchGetSubMsgStatus error, err = %v, oid = %v", err, oid)
			return nil, err
		}
		for _, value := range ret {
			if cacheErr == redis.Nil {
				err = cache_access.SetSubMsgStatus(value.UserId, value.MsgTempleId, value.Status)
				if err != nil {
					log.Error("cache_access.SetSubMsgStatus set oid: %v, msgID: %v, status: %v",
						value.UserId, value.MsgTempleId, value.Status)
				}
			}
			for _, msgID := range msgIds {
				if msgID == value.MsgTempleId {
					finalRet = append(finalRet, data_access.SubMsgStatus{
						UserId:      oid,
						MsgTempleId: msgID,
						Status:      value.Status,
					})
				}
			}
		}
	} else {
		finalRet = ret
	}

	return finalRet, nil
}

func GetSubMsgStatusByOidsId(oids []string, msgId string) ([]data_access.SubMsgStatus, error) {
	ret, err := data_access.BatchGetSubMsgStatusByOidsId(oids, msgId)
	if err != nil {
		log.Error("GetSubMsgStatusById error, err = %v, oids = %v, tid = %v", err, oids, msgId)
		return nil, err
	}

	return ret, nil
}

func BatchUpdateSubMsgStatus(dp data_access.DBProxy, oids []string, msgId string, status int) error {
	for _, oid := range oids {
		if err := cache_access.SetSubMsgStatus(oid, msgId, status); err != nil {
			log.Error("cache_access.SetSubMsgStatus oid: %v, msgID: %v value: %v", oid, msgId, status)
		}
	}
	err := data_access.BatchUpdateSubMsgStatus(dp, oids, msgId, status)
	if err != nil {
		log.Error("BatchUpdateSubMsgStatus error, err = %v, oids = %v", err, oids)
		return err
	}

	return nil
}

func UpdateSubMsgStatus(dp data_access.DBProxy, oid string, msgIds []string) error {
	for _, msgID := range msgIds {
		if err := cache_access.SetSubMsgStatus(oid, msgID, 0); err != nil {
			log.Error("cache_access.SetSubMsgStatus oid: %v, msgID: %v value: 0", oid, msgID)
		}
	}
	err := data_access.BatchSetSubMsgStatus(dp, oid, msgIds)
	if err != nil {
		log.Error("UpdateSubMsgStatus error, err = %v, oid = %v", err, oid)
		return err
	}

	return nil
}

func BatchSetSubOidMsgStatus(dp data_access.DBProxy, oids []string, msgId string, status int) error {
	for _, oid := range oids {
		if err := cache_access.SetSubMsgStatus(oid, msgId, status); err != nil {
			log.Error("cache_access.SetSubMsgStatus oid: %v, msgID: %v value: %v", oid, msgId, status)
		}
	}
	err := data_access.BatchSetSubOidMsgStatus(dp, oids, msgId, status)
	if err != nil {
		log.Error("BatchSetSubOidMsgStatus error, err = %v, oids = %v, msgId = %s, status = %d", err, oids, msgId, status)
		return err
	}

	return nil
}

func SubMsgStatus(dp data_access.DBProxy, oids []string, msgID string) error {
	if len(oids) == 0 {
		return nil
	}
	var moids = make(map[string]bool)
	for _, v := range oids {
		moids[v] = true
	}

	// 获取状态需要更新的用户
	smss, err := GetSubMsgStatusByOidsId(oids, msgID)
	if err != nil {
		log.Error("GetSubMsgStatusByOidsId error, err = %v, oids = %v, msgID = %v", err, oids, msgID)
		return err
	}
	var uoids []string
	for _, v := range smss {
		if v.Status == 1 {
			continue
		}
		uoids = append(uoids, v.UserId)
		delete(moids, v.UserId)
	}
	if len(uoids) > 0 {
		err = BatchUpdateSubMsgStatus(dp, oids, msgID, 1)
		if err != nil {
			log.Error("BatchUpdateSubOidMsgStatus error, err = %v, oids = %v, msgID = %v", err, oids, msgID)
		}
	}

	// 需要插入数据的用户
	var ioids []string
	for k, v := range moids {
		if v == false {
			continue
		}
		ioids = append(ioids, k)
	}
	if len(ioids) > 0 {
		err = BatchSetSubOidMsgStatus(dp, oids, msgID, 1)
		if err != nil {
			log.Error("BatchSetSubOidMsgStatus error, err = %v, oids = %v, msgID = %v", err, oids, msgID)
		}
	}

	return nil
}
