//
//

package business_access

import (
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
)

const DonateAppId = "wxff244f6b82a094d2"

func InsertUnionId(oid, unionId string) error {
	// check cache first
	profile, err := cache_access.GetUserProfile(oid)
	if err != nil {
		log.Error("get oid: %v profile cache error: %v", oid, err)
	} else {
		if len(profile.Unin) != 0 {
			return nil
		}
	}
	// check db again
	exist, err := data_access.IsInUserTable(oid)
	if err != nil {
		log.Error("IsInUserTable error, err = %v, oid = %s", err, oid)
		return err
	}
	if !exist {
		db_proxy := data_access.DBProxy{
			DBHandler: data_access.DBHandler,
			Tx:        false,
		}
		nt_date := time.Now().Format("2006-01-02 15:04:05")
		user_info := data_access.UserField{
			FUserId:        oid,
			FUninId:        unionId,
			FAppid:         DonateAppId,
			FJoinEvent:     1,
			FBackendUpdate: 1,
			FStatus:        0,
			FSubTime:       nt_date,
			FCreateTime:    nt_date,
			FModifyTime:    nt_date,
		}
		err = data_access.InsertUserInfo(db_proxy, user_info)
		if err != nil {
			log.Error("InsertUserInfo error, err = %v", err)
			return err
		}
		// add cache
		err = cache_access.AddUserProfile(oid, unionId, 0, 0, 1, 1, util.GetLocalFormatTime())
		if err != nil {
			log.Error("set user: %v profile into cache err: %v", oid, err)
		}
	} else {
		oidFunds, err := GetTransactionV2RechargeSum([]string{oid})
		if err != nil {
			log.Error("GetTransactionV2RechargeSum user: %v error: %v", oid, err)
		} else {
			leaf := oidFunds[oid] / 10
			info, err := GetUserInfo(oid)
			if err != nil {
				log.Error("GetUserInfo user: %v info error: %v", oid, err)
			} else {
				// add cache
				err = cache_access.AddUserProfile(oid, unionId, int(leaf), info.FStatus, info.FJoinEvent, info.FBackendUpdate, info.FSubTime)
				if err != nil {
					log.Error("set user: %v profile into cache err: %v", oid, err)
				}
			}
		}
	}
	return nil
}

func GetUserProfile(oid string) (*cache_access.UserProfile, error) {
	userProfile, err := cache_access.GetUserProfile(oid)
	if err != nil {
		userProfile, err = RecoverUserProfile(oid)
		if err != nil {
			log.Error("RecoverUserProfile get user: %v profile from db error: %v", oid, err)
			return nil, err
		}
		log.Info("cache_access.GetUserProfile get user: %v profile from redis error: %v but recover from db: %v", oid, err, userProfile)
		return userProfile, nil
	}
	return userProfile, nil
}

func RecoverUserProfile(oid string) (*cache_access.UserProfile, error) {
	var result *cache_access.UserProfile
	// get basic info
	info, err := GetUserInfo(oid)
	if err != nil {
		log.Error("GetUserInfo user: %v info error: %v", oid, err)
		return nil, err
	}
	result = &cache_access.UserProfile{
		Unin:     info.FUninId,
		Donated:  info.FStatus,
		Joined:   info.FJoinEvent,
		AutoFlag: info.FBackendUpdate,
		AutoTime: info.FSubTime,
	}
	if len(result.Unin) == 0 {
		return result, nil
	}
	// get leaf
	oidFunds, err := GetTransactionV2RechargeSum([]string{oid})
	if err != nil {
		log.Error("GetTransactionV2RechargeSum user: %v error: %v", oid, err)
	} else {
		leaf := oidFunds[oid] / 10
		result.Leaf = int(leaf)
		// add cache
		err = cache_access.AddUserProfile(oid, info.FUninId, int(leaf), info.FStatus, info.FJoinEvent, info.FBackendUpdate, info.FSubTime)
		if err != nil {
			log.Error("set user: %v profile into cache err: %v", oid, err)
		}
	}
	return result, nil
}

func UpdateUserStatus(db_proxy data_access.DBProxy, oid string, status int) error {
	if err := cache_access.DeleteUserStatus(oid); err != nil {
		log.Error("DeleteUserStatus error, err = %v, oid = %s", err, oid)
		return err
	}

	if err := data_access.UpdateUserStatus(db_proxy, oid, status); err != nil {
		log.Error("UpdateUserStatus error, err = %v, oid = %s, status = %d", err, oid, status)
		return err
	}

	if err := cache_access.UpdateUserProfileDonate(oid, status); err != nil {
		log.Error("UpdateUserProfileDonate error, err = %v, oid = %s, status = %d", err, oid, status)
		return err
	}

	return nil
}

func GetUserInfo(oid string) (uf data_access.UserField, err error) {
	uf, err = data_access.GetUserInfo(oid)
	if err != nil {
		log.Error("GetUserInfo error, err = %v, oid = %s", err, oid)
		return uf, err
	}
	return uf, nil
}


func GetUninid(oid string) (string, error) {
	profile, err := GetUserProfile(oid)
	if err != nil {
		log.Error("cache_access.GetUserProfile can not get oid: %v profile", oid)
	} else {
		if len(profile.Unin) != 0 {
			return profile.Unin, nil
		}
	}
	id, err := data_access.GetUninid(oid)
	if err != nil {
		log.Error("GetUninid error, err = %v", err)
		return "", err
	}
	return id, nil
}

func GetUninids(oids []string) (map[string]string, error) {
	var result = make(map[string]string)
	for _, value := range oids {
		ret, err := GetUninid(value)
		if err != nil {
			log.Error("GetUninids oid: %v, error: %v", value, err)
		} else {
			result[value] = ret
		}
	}
	/*
		ret_map, err := data_access.GetUninids(oids)
		if err != nil {
			log.Error("GetUninids error, err = %v", err)
			return nil, err
		}
	*/
	return result, nil
}
