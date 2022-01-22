package business_access

import (
	"fmt"
	"strings"
	"time"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"github.com/go-redis/redis"
)

func GetBigMapMyGlobalSteps(pId, myOid string) (*data_access.GlobalStep, error) {
	myGlobalStep, err := data_access.QueryBigMapMyGlobalSteps(pId, myOid)
	if err != nil {
		log.Error("QueryBigMapMyGlobalSteps error:%s", err.Error())
	}
	return myGlobalStep, err
}

func GetBigMapStrangerGlobalSteps(pid, myOid string, mySteps int64, nearbyFriends []string) (*data_access.GlobalStepSlice, error) {
	globalStep, err := data_access.QueryBigMapStrangerGlobalSteps(pid, myOid, mySteps, nearbyFriends)
	if err != nil {
		log.Error("QueryBigMapGlobalSteps error:%s", err.Error())
	}
	return globalStep, err
}

func GetFriendList(oid string) (*[]string, error) {
	friends := make([]string, 0)
	key := cache_access.GetFriendListKey(oid)
	rCmd := cache_access.RedisClient.ZRange(key, 0, -1)
	values, err := rCmd.Result()
	for _, v := range values {
		friends = append(friends, v)
	}

	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %+v", err, rCmd)
		return nil, err
	}
	log.Info("openid get friend list => %+v", friends)

	log.Debug("cmd = %+v", rCmd)
	return &friends, nil
}

func GetFriendDistancePoints(pid, oid string) (*data_access.GlobalStepSlice, error) {
	type FriendPoint struct {
		Openid string `json:"oid"`
		Steps  int64  `json:"steps"`
	}
	type FriendPoints struct {
		Dps []FriendPoint `json:"friends"`
	}

	globalStepSlice := new(data_access.GlobalStepSlice)
	var friendPoints FriendPoints
	key := fmt.Sprintf("yqz_%s_%s_friends_nearby", pid, oid)
	rCmd := cache_access.RedisClient.Get(key)
	if rCmd.Err() != nil && rCmd.Err() != redis.Nil {
		log.Error("get redis exception:%s", rCmd.Err().Error())
		return nil, rCmd.Err()
	} else {
		bytes, _ := rCmd.Result()
		if len(bytes) == 0 {
			return globalStepSlice, nil
		}

		if err := json.Unmarshal([]byte(bytes), &friendPoints); err != nil {
			log.Error("json.Marshal fail:%s \n", bytes)
			return nil, err
		}
	}

	for _, v := range friendPoints.Dps {
		*globalStepSlice = append(*globalStepSlice, data_access.GlobalStep{
			UserId:     v.Openid,
			TotalSteps: v.Steps,
		})
	}

	return globalStepSlice, nil
}

func weekStartEnd(t time.Time) (time.Time, time.Time) {
	offset := int(time.Monday - t.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd
}

func IsExistInGlobalStep(oid string, pid string) (bool, error) {
	var exist bool
	var err error
	start, _ := weekStartEnd(time.Now())
	if start.Format("2006-01-02") == pid {
		exist, err = cache_access.IsInBigMap(oid, pid)
		if err != nil {
			log.Error("cache_access.IsInBigMap error, err = %v, oid = %s, week = %s", err, oid, pid)
			exist, err = data_access.IsExistInGlobalStep(oid, pid)
			if err != nil {
				log.Error("IsExistInGlobalStep error, err = %v, oid = %s, pid = %s", err, oid, pid)
				return false, err
			}
		}
	} else {
		exist, err = data_access.IsExistInGlobalStep(oid, pid)
		if err != nil {
			log.Error("IsExistInGlobalStep error, err = %v, oid = %s, pid = %s", err, oid, pid)
			return false, err
		}
	}
	return exist, nil
}

// IsExistUserInGlobalStep check if user in global step ever
func IsExistUserInGlobalStep(oid string) (bool, error) {
	// add cache for query
	var exist bool
	profile, err := GetUserProfile(oid)
	if err != nil {
		exist, err = data_access.IsExistUserInGlobalStep(oid)
		if err != nil {
			log.Error("IsExistUserInGlobalStep error, err = %v, oid = %s", err, oid)
			return false, err
		}
	} else {
		if profile.Donated == 1 {
			exist = true
		}
	}
	return exist, nil
}

func GetGlobalStepUserJoinDate(oid string) ([]string, error) {
	dates, err := data_access.GetGlobalStepUserJoinDate(oid)
	if err != nil {
		log.Error("GetGlobalStepUserJoinDate error, err = %v, oid = %s", err, oid)
		return nil, err
	}
	return dates, nil
}

func InsertGlobalStep(db_proxy data_access.DBProxy, global_step data_access.GlobalStep) error {
	if err := data_access.InsertGlobalStep(db_proxy, global_step); err != nil {
		log.Error("data_access.InsertGlobalStep error, err = %v, gs = %+v", err, global_step)
		if !strings.Contains(err.Error(), "Duplicate entry") {
			return err
		}
	}
	if err := cache_access.SetUserInBigMap(global_step.UserId, global_step.Period); err != nil {
		log.Error("cache_access.SetUserInBigMap error, err = %v, oid = %s, week = %s", err, global_step.UserId, global_step.Period)
		return err
	}

	return nil
}

func GetCountGloableStepByPeriod(period string) (int64, error) {
	count, err := data_access.GetCountGloableStepByPeriod(period)
	if err != nil {
		log.Error("GetCountGloableStepByPeriod error:%s", err.Error())
		return 0, err
	}
	return count, err
}

func GetRankGloableStepByPeriod(step int, period string) (int64, error) {
	rank, err := data_access.GetRankGloableStepByPeriod(step, period)
	if err != nil {
		log.Error("GetRankGloableStepByPeriod error:%s", err.Error())
		return 0, err
	}
	return rank, err
}

func GetCountGlobalStep(pid string) (int64, error) {
	count, err := data_access.GetCountGlobalStep(pid)
	if err != nil {
		log.Error("GetCountGlobalStep error, err = %v, pid = %s", err, pid)
		return 0, err
	}
	return count, nil
}

func GetRandBigMapUserOid(pid string) ([]string, error) {
	oids, err := data_access.GetRandBigMapUserOid(pid)
	if err != nil {
		log.Error("GetRandBigMapUserOid error, err = %v, pid = %s", err, pid)
		return oids, err
	}
	return oids, nil
}

func GetGlobalStepList(oid string) ([]data_access.GlobalStep, error) {
	gss, err := data_access.GetGlobalStepList(oid)
	if err != nil {
		log.Error("GetGlobalStepList error, err = %v, oid = %v", err, oid)
		return nil, err
	}
	return gss, nil
}

func GetGlobalStepListByPeriodOids(period string, oids []string) ([]data_access.GlobalStep, error) {
	gss, err := data_access.GetGlobalStepListByPeriodOids(period, oids)
	if err != nil {
		log.Error("GetGlobalStepListByPeriodOids error, err = %v, period = %v, oids = %v", err, period, oids)
		return nil, err
	}
	return gss, nil
}

func GetBigMapDistanceZoneOid(pid string, st, et int64, limit int) ([]data_access.OidStep, error) {
	oid_step, err := data_access.GetBigMapDistanceZoneOid(pid, st, et, limit)
	if err != nil {
		log.Error("GetBigMapDistanceZoneOid error, err = %v, pid = %s, st = %d, et = %d, limit = %d", err, st, et, limit)
		return nil, err
	}
	return oid_step, nil
}

func GetBigMapDistanceZoneNum(pid string, st, et int64) (int, error) {
	num, err := data_access.GetBigMapDistanceZoneNum(pid, st, et)
	if err != nil {
		log.Error("GetBigMapDistanceZoneNum error, err = %v, pid = %s, st = %d, et = %d", err, st, et)
		return 0, err
	}
	return num, nil
}
