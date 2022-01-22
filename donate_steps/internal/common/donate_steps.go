//
//

package common

import (
	"strconv"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
)

const MAX_RED_FUND = 1000 * 100        // 个人在一个活动下面最多领取1000元
const MAX_ONE_DAY_STEPS = 30000        // 每天限制捐步三万
const MAX_BLACK_ONE_DAY_STEPS = 100000 // 一天超过6w5步，就判定为刷步

type DonateStepsRet struct {
	DeltaSteps          int64
	YesterdayDeltaSteps int64
	DeltaDonate         int64
	WeekStep            int64 // 当前周的步数
	Both                bool
}

type DonateEventInfo struct {
	Eid         string
	Oid         string
	Week        string
	WeekRouteId string // 周地图id
}

type DifferentStepsRet struct {
	DeltaSteps           int64
	WeekRefillDeltaSteps int64 // 本周需要回补的步数
	WeekTotalSteps       int
	YesterdayDeltaSteps  int64
	TodayDeltaSteps      int64
	BeforeSteps          map[string]int64
	YesTodaySteps        map[string]int64
}

// donateOptions opts...
type donateOptions struct {
	JoinBigMap bool
}

// DonateOption configures donate.
type DonateOption interface {
	apply(*donateOptions)
}

// funcDonateOption wraps a function that modifies donateOptions into an
// implementation of the DonateOption interface.
type funcDonateOption struct {
	f func(*donateOptions)
}

func (fdo *funcDonateOption) apply(do *donateOptions) {
	fdo.f(do)
}

func newFuncDonateOption(f func(*donateOptions)) *funcDonateOption {
	return &funcDonateOption{
		f: f,
	}
}

func WithJoinBigMap(flag bool) DonateOption {
	return newFuncDonateOption(func(o *donateOptions) {
		o.JoinBigMap = flag
	})
}

func DecodeWxSteps(appid, unin_id, edata, eiv string) (map[string]int64, error) {
	var wx_steps_array WxStepsArray
	skey, err := QueryWxSessionKey(appid, unin_id)
	if err != nil {
		log.Error("QueryWxSessionKey error, err = %v, appid = %s, uni_id = %s", err, appid, unin_id)
		return nil, err
	}
	wx_steps_array, err = DecodeWxData(appid, skey, edata, eiv)
	if err != nil {
		log.Error("appid: %v, uni_id: %v DecodeWxData error, err = %v", appid, unin_id, err)
		return nil, err
	}

	date_steps := make(map[string]int64)
	for _, it := range wx_steps_array.Stepinfolist {
		tm := time.Unix(it.Timestamp, 0)
		date := tm.Format("2006-01-02")
		date_steps[date] = it.Steps
	}

	return date_steps, nil
}

func DonateAllSteps(oid, nick string, nt time.Time, auto bool, dateSteps map[string]int64, opts ...DonateOption) error {
	eids, err := business_access.GetEventIDByOid(oid)
	if err != nil {
		log.Error("GetEventIDByOid error, err = %v, oid = %s", err, oid)
		return err
	}
	/*
		tx, err := data_access.DBHandler.Begin()
		if err != nil {
			log.Error("begin db transaction error, err = %s", err.Error())
			return err
		}
		defer tx.Rollback()
		db_proxy := data_access.DBProxy{
			DBHandler: tx,
			Tx:        true,
		}
	*/
	db_proxy := data_access.DBProxy{
		DBHandler: data_access.DBHandler,
		Tx:        false,
	}
	for _, eid := range eids {
		event_info, err := business_access.GetEventInfoByEid(eid)
		if err != nil {
			log.Error("GetEventInfoByEid error, err = %v, event_id = %s", err, eid)
			continue
		}
		if event_info.FRouteId == 0 || event_info.FStatus != 2 {
			log.Info("event info = %+v, route id = 0 or status != 2", event_info)
			continue
		}
		if event_info.FStatus == 3 {
			// 处理活动已结束的脏数据
			if err := business_access.UpdateEventUserStatus(db_proxy, eid, oid, 1); err != nil {
				log.Error("UpdateEventUserStatus error, err = %v, eid = %s, oid = %s", err, eid, oid)
				continue
			}
		}
		err = DonateStepsV2(db_proxy, eid, oid, nt, dateSteps)
		if err != nil {
			log.Error("DonateStepsV2 error, err = %v", err)
			continue
		}
	}
	// add user to big map or update big map data if need
	sum, err := BigMapDonate(db_proxy, oid, auto, nt, dateSteps, opts...)
	if err != nil {
		log.Error("BigMapDonate error, err = %v, oid = %s", err, oid)
	}
	// update step info
	err = InitiativeUpdateSteps(db_proxy, oid, dateSteps, auto)
	if err != nil {
		log.Error("InitiativeUpdateSteps error, err = %v", err)
		return err
	}

	// 自动捐步会触发POI推送
	if auto {
		wm, err := GetWeekMap()
		if err != nil {
			log.Error("GetWeekMap error, err = %v", err)
			return err
		}
		startTime, endTime := WeekStartEnd(nt)
		rid := wm[startTime.Format("20060102")].MapId
		/*
			weekConfig, err := GetGlobalWeekLevel()
			if err != nil {
				log.Error("GetGlobalWeekLevel error, err = %v", err)
				return err
			}
		*/
		ridStr := strconv.Itoa(rid)
		// get from ckv first
		gwl, err := GetSpecifyMapInCKVByID(ridStr)
		if err != nil {
			log.Error("common.GetSpecifyMapInCKVByID rid: %v error: %v", ridStr, err)
			// get from mem second
			gwl, err = GetSpecifyMapInMem(rid)
			if err != nil {
				log.Error("common.GetSpecifyMapInMem rid: %v error: %v", rid, err)
				return err
			}
		}

		if err = SendWeekFinishChallengeMsg(oid, sum, nt, endTime, gwl.Distance); err != nil {
			log.Error("SendWeekFinishChallengeMsg error, err = %v, oid = %s", err, oid)
			return err
		}

		err = SendPOIViewPoint(oid, startTime.Format("2006-01-02"), gwl.Abbr, int(sum), gwl.Poi, nt, endTime)
		if err != nil {
			log.Error("SendPOIViewPoint error, err = %v, oid = %s", err, oid)
			return err
		}
	}
	/*
		err = tx.Commit()
		if err != nil {
			log.Error("tx commit error, err = %v", err)
			return err
		}
	*/
	return nil
}

func DonateStepsV2(db_proxy data_access.DBProxy, eid, oid string, nt time.Time, dateStepsMap map[string]int64) error {
	var ret DonateStepsRet
	start, end := WeekStartEnd(nt)
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")

	stepsRet, err := DiffStepsV2(eid, oid, startDate, endDate, nt, dateStepsMap)
	if err != nil {
		log.Error("DiffStepsV2 error, err = %v, eid = %s, oid = %s, start_date = %s, end_date = %s,"+
			"date_steps_map = %v", err, eid, oid, startDate, endDate, dateStepsMap)
		return err
	}
	log.Info("eid = %s, oid = %s, steps_ret = %+v", eid, oid, stepsRet)

	if stepsRet.WeekRefillDeltaSteps > 0 && len(stepsRet.BeforeSteps) != 0 {
		mapList, err := GetWeekMap()
		if err != nil {
			log.Error("common.GetWeekMap error: %v", err)
			return err
		}
		t, _ := WeekStartEnd(time.Now())
		rid := mapList[t.Format("20060102")].MapId
		donateEventInfo := DonateEventInfo{
			Eid:         eid,
			Oid:         oid,
			Week:        startDate,
			WeekRouteId: strconv.FormatInt(int64(rid), 10),
		}
		err = DonateStepWithRewardV2(db_proxy, &donateEventInfo, &stepsRet)
		if err != nil {
			log.Error("DonateStepWithRewardV2 error, err = %v, eid = %s, oid = %s, user_steps = %+v", err, eid, oid, stepsRet)
			return err
		}
	}

	// force reset user team step stat
	if err = cache_access.AddTeamStatToCache(eid, &cache_access.TeamUserStat{Oid: oid, Score: int(stepsRet.WeekTotalSteps)}); err != nil {
		log.Error("AddTeamStatToCache user: %v, team: %v stat error: %v", oid, eid, err)
	}

	ret.DeltaSteps = stepsRet.DeltaSteps
	ret.YesterdayDeltaSteps = stepsRet.YesterdayDeltaSteps
	if ret.DeltaSteps == stepsRet.TodayDeltaSteps {
		ret.Both = false
	} else {
		ret.Both = true
	}
	log.Debug("DonateStepsV2 update eid: %v, oid: %v, ret: %v", eid, oid, ret)
	return nil
}

func BigMapDonate(db_proxy data_access.DBProxy, oid string, auto bool, nt time.Time, dateStepsMap map[string]int64, opts ...DonateOption) (int64, error) {
	sum := int64(0)
	start, end := WeekStartEnd(nt)
	startDate := start.Format("2006-01-02")
	endDate := end.Format("2006-01-02")
	if auto {
		// only auto trigger, it can accept
		var err error
		sum, err = business_access.GetSumStepsByDates(oid, startDate, endDate)
		if err != nil {
			log.Error("GetSumStepsByDates error, err = %v, oid = %s, sd = %s, ed = %s", err, oid, startDate, endDate)
			return 0, err
		}
	} else {
		for i := start.Unix(); i <= nt.Unix(); {
			tmpDate := time.Unix(i, 0).Format("2006-01-02")
			sum += dateStepsMap[tmpDate]
			i += 24 * 60 * 60
		}
	}
	if err := BigMapDonateStepV2(db_proxy, oid, startDate, endDate, nt, sum, dateStepsMap, opts...); err != nil {
		log.Error("BigMapDonateStepV2 error, err = %v, oid = %s, date_steps = %+v", err, oid, dateStepsMap)
		return 0, err
	}
	return sum, nil
}

func DiffStepsV2(eid, oid, wsd, wed string, nt time.Time, dateStepsMap map[string]int64) (DifferentStepsRet, error) {
	var ret DifferentStepsRet
	dateSteps, err := business_access.GetUserStatUserDateSteps(eid, oid, wsd, wed)
	if err != nil {
		log.Error("GetUserStatUserDateSteps error, err = %v, eid = %s, oid = %s, start_date = %s, end_date = %s",
			err, eid, oid, wsd, wed)
		return ret, err
	}
	var yDonateSteps, tDonateSteps, yDeltaSteps, tDeltaSteps, deltaSteps int64
	ntDate := nt.Format("2006-01-02")
	yesterdayDate := nt.AddDate(0, 0, -1).Format("2006-01-02")
	if _, ok := dateSteps[yesterdayDate]; ok {
		yDonateSteps = dateSteps[yesterdayDate]
	}
	if _, ok := dateSteps[ntDate]; ok {
		tDonateSteps = dateSteps[ntDate]
	}

	log.Info("eid = %s, oid = %s, y donate = %d, t donate = %d", eid, oid, yDonateSteps, tDonateSteps)
	log.Info("eid = %s, oid = %s, wx y steps = %d, wx t steps = %d", eid, oid, dateStepsMap[yesterdayDate], dateStepsMap[ntDate])

	// this logical is useless ??? why 30000 step a day
	ret.YesTodaySteps = make(map[string]int64, 2)
	tmpTodayStep := int64(0)
	if dateStepsMap[ntDate] > MAX_ONE_DAY_STEPS {
		tmpTodayStep = MAX_ONE_DAY_STEPS
		tDeltaSteps = MAX_ONE_DAY_STEPS - tDonateSteps
		ret.YesTodaySteps[ntDate] = MAX_ONE_DAY_STEPS
	} else {
		tmpTodayStep = dateStepsMap[ntDate]
		tDeltaSteps = dateStepsMap[ntDate] - tDonateSteps
		ret.YesTodaySteps[ntDate] = dateStepsMap[ntDate]
	}

	// 这个地方，如果第一次捐，那么昨天和今天的步数都算了奖励
	if wsd <= yesterdayDate {
		if dateStepsMap[yesterdayDate] > MAX_ONE_DAY_STEPS {
			deltaSteps = MAX_ONE_DAY_STEPS + tmpTodayStep - yDonateSteps - tDonateSteps
			yDeltaSteps = MAX_ONE_DAY_STEPS - yDonateSteps
			ret.YesTodaySteps[yesterdayDate] = MAX_ONE_DAY_STEPS
		} else {
			deltaSteps = dateStepsMap[yesterdayDate] + tmpTodayStep - yDonateSteps - tDonateSteps
			yDeltaSteps = dateStepsMap[yesterdayDate] - yDonateSteps
			ret.YesTodaySteps[yesterdayDate] = dateStepsMap[yesterdayDate]
		}
	} else {
		deltaSteps = tmpTodayStep - tDonateSteps
		yDeltaSteps = 0
		ret.YesTodaySteps[yesterdayDate] = 0
	}

	var weekRefillSteps int64
	var weekTotalSteps int
	// 默认回补从本周一开始的步数
	ret.BeforeSteps = make(map[string]int64)
	for k, iv := range dateStepsMap {
		if k >= wsd && k <= wed {
			v := iv
			if iv > MAX_ONE_DAY_STEPS {
				v = MAX_ONE_DAY_STEPS
			}
			if _, ok := dateSteps[k]; ok {
				delta := v - dateSteps[k]
				// 老活动没有3w步的限制，兼容老活动
				if delta < 0 {
					delta = 0
				} else if delta > 0 {
					// 有更新才记录下来
					ret.BeforeSteps[k] = delta
				}
				weekRefillSteps += delta
			} else {
				weekRefillSteps += v
				ret.BeforeSteps[k] = v
			}
		}
	}
	ret.DeltaSteps = deltaSteps
	ret.WeekRefillDeltaSteps = weekRefillSteps
	ret.TodayDeltaSteps = tDeltaSteps
	ret.YesterdayDeltaSteps = yDeltaSteps

	for date, step := range dateSteps {
		if date >= wsd && date <= wed {
			weekTotalSteps += int(step)
		}
	}
	ret.WeekTotalSteps = weekTotalSteps + int(weekRefillSteps)

	return ret, nil
}

func DonateStepWithRewardV2(db_proxy data_access.DBProxy, einfo *DonateEventInfo, diffSteps *DifferentStepsRet) error {
	var exist bool
	teams, err := business_access.GetEventIDByOid(einfo.Oid)
	if err != nil {
		log.Error("business_access.GetEventIDByOid oid: %v error, err = %v", einfo.Oid, err)
		return err
	} else {
		for _, value := range teams {
			if value == einfo.Eid {
				exist = true
			}
		}
	}
	rank, err := business_access.GetTeamUserRank(einfo.Eid, einfo.Oid)
	if err != nil || rank < 1 {
		log.Info("user: %v not in team: %v rank", einfo.Oid, einfo.Eid)
		exist = false
	}

	nt_time := time.Now()
	nt_date := nt_time.Format("2006-01-02 15:04:05")
	if !exist {
		event_user_field := data_access.EventUserField{
			FEventId:            einfo.Eid,
			FUserId:             einfo.Oid,
			FUserNickname:       "", // 废弃
			FUserPic:            "", // 废弃
			FTotalStep:          diffSteps.WeekRefillDeltaSteps,
			FTotalFund:          0,
			FTotalRechargeTimes: 0,
			FTotalDonation:      0,
			FTotalLike:          0,
			FTotalInvited:       0,
			FStatus:             2,
			FPointStatus:        1, // 废弃
			FCurrentPointStep:   0, // 废弃
			FJoinTime:           nt_date,
			FCreateTime:         nt_date,
			FModifyTime:         nt_date,
		}
		if err = business_access.InsertEventUserData(db_proxy, event_user_field); err != nil {
			log.Error("InsertEventUserData error, err = %v, event_user_field = %v", err, event_user_field)
			return err
		}

		if err = business_access.IncrEventTotalMember(db_proxy, einfo.Eid); err != nil {
			log.Error("IncrEventTotalMember error, err = %v, eid = %s", err, einfo.Eid)
			return err
		}

		if err = business_access.UpdateEventUserStatus(db_proxy, einfo.Eid, einfo.Oid, 2); err != nil {
			log.Error("business_access.UpdateEventUserStatus error, err = %v, eid: %s, oid: %v", err, einfo.Eid, einfo.Oid)
			return err
		}
	}

	// update t_event_week_stat
	exist, err = business_access.IsInEventWeekStat(einfo.Eid, einfo.Week, einfo.Oid)
	if err != nil {
		log.Error("IsInEventRouteStat error, err = %v, eid = %s, week = %s, oid = %s", err, einfo.Eid, einfo.Week, einfo.Oid)
		return err
	}
	if !exist {
		var event_week_stat data_access.EventWeekStat
		event_week_stat.FEventId = einfo.Eid
		event_week_stat.FWeek = einfo.Week
		event_week_stat.FRouteId = einfo.WeekRouteId
		event_week_stat.FUserId = einfo.Oid
		event_week_stat.FStep = diffSteps.WeekRefillDeltaSteps
		event_week_stat.FInTeam = 2
		event_week_stat.FCreateTime = nt_date
		event_week_stat.FModifyTime = nt_date
		if err = business_access.InsertEventWeekStat(db_proxy, event_week_stat); err != nil {
			log.Error("InsertEventWeekStat error, err = %v", err)
			return err
		}
	} else {
		if err = business_access.AddEventWeekStatStepInTeamStatus(db_proxy, einfo.Eid, einfo.Week, einfo.Oid, diffSteps.WeekRefillDeltaSteps, 2); err != nil {
			log.Error("AddEventWeekStatStepInTeamStatus error, err = %v, eid = %s, week = %s, oid = %s, step = %d",
				err, einfo.Eid, einfo.Week, einfo.Oid, diffSteps.WeekRefillDeltaSteps)
			return err
		}
	}

	// update t_user_stat
	for k, v := range diffSteps.BeforeSteps {
		if err = UpdateDateStepsToUserStat(db_proxy, einfo.Eid, einfo.Oid, k, v); err != nil {
			log.Error("UpdateDateStepsToUserStat error, err = %v, eid = %s, oid = %s, date = %s, steps = %d",
				err, einfo.Eid, einfo.Oid, k, v)
			return err
		}
	}

	return nil
}

type BigMapDonateStepRet struct {
	DeltaStep int64
	Black     bool
}

func BigMapDonateStepV2(dbProxy data_access.DBProxy, oid, sd, ed string, nt time.Time, sum int64, dateSteps map[string]int64, opts ...DonateOption) error {
	options := &donateOptions{}
	for _, v := range opts {
		v.apply(options)
	}
	block := FilterBlackUser(sd, nt, dateSteps)
	if block {
		if err := cache_access.RmUserFromBigMapRankAndAddBlackList(oid, sd, nt, sum); err != nil {
			log.Error("RmUserFromBigMapRankAndAddBlackList error, err = %v, oid = %s, sd = %s", err, oid, sd)
			return err
		}
		log.Info("%v - oid: %v is in blacklist", util.GetCallee(), oid)
	}

	// check if in current yqz map
	exist, err := business_access.IsExistInGlobalStep(oid, sd)
	if err != nil {
		log.Error("IsExistInGlobalStep error, err = %v, oid = %s, week = %s", err, oid, sd)
		return err
	}
	if !exist {
		// check if joined ever
		joined, err := business_access.IsExistUserInGlobalStep(oid)
		if err != nil {
			log.Error("IsExistUserInGlobalStep error, err = %v, oid = %s", err, oid)
			return err
		}
		// check if need join to big map
		if !options.JoinBigMap && !joined {
			log.Info("%v - oid: %v do not need join bigmap", util.GetCallee(), oid)
			return nil
		}
		// add new record for current period yqz map
		if !exist {
			ntDate := nt.Format("2006-01-02 15:04:05")
			if err := business_access.InsertGlobalStep(dbProxy, data_access.GlobalStep{
				UserId:        oid,
				RouteId:       0,
				Period:        sd,
				TotalSteps:    0,
				DistancePoint: 0,
				JoinDate:      ntDate[0:10],
				Status:        0,
				CreateTime:    ntDate,
				ModifyTime:    ntDate,
			}); err != nil {
				log.Error("InsertGlobalStep error, err = %v, oid = %s", err, oid)
				return err
			}
			log.Info("%v - oid: %v join week: %v big map", util.GetCallee(), oid, sd)
		}
		// change big map status
		if !joined {
			if err := business_access.UpdateUserStatus(dbProxy, oid, 1); err != nil {
				log.Error("UpdateUserStatus error, err = %v, oid = %s", err, oid)
				return err
			}
			log.Info("%v - oid: %v join yqz activity", util.GetCallee(), oid)
		}
	}
	// check if not set in rank
	if !block {
		if err := cache_access.AddUserToBigMapRank(oid, sd, sum); err != nil {
			log.Error("AddUserToBigMapRank error, err = %v, oid = %s", err, oid)
			return err
		}
	}

	return nil
}

// 以周为单位，只要用户某一天的步数达到6.5万步，立刻放入黑名单
func FilterBlackUser(sd string, nt time.Time, dateStep map[string]int64) bool {
	start, _ := time.Parse("2006-01-02", sd)
	for i := start.Unix(); i <= nt.Unix(); {
		tmpDate := time.Unix(i, 0).Format("2006-01-02")
		if _, ok := dateStep[tmpDate]; ok {
			if dateStep[tmpDate] >= MAX_BLACK_ONE_DAY_STEPS {
				return true
			}
		}
		i += 24 * 60 * 60
	}
	return false
}

func UpdateDateStepsToUserStat(db_proxy data_access.DBProxy, eid, oid, date string, delta_steps int64) error {
	exist, err := data_access.IsUserDonateByEidUidDate(eid, oid, date[0:10])
	if err != nil {
		log.Error("IsUserDonateByEidUidDate error, err = %v, eid = %s, oid = %s, date = %s", err, eid, oid, date)
		return err
	}

	if !exist {
		user_stat := data_access.UserStatField{
			FEventId:    eid,
			FUserId:     oid,
			FStep:       delta_steps,
			FFund:       0,
			FDonation:   0,
			FLike:       0,
			FInvited:    0,
			FDate:       date[0:10],
			FCreateTime: date,
			FModifyTime: date,
		}
		if err = business_access.InsertUserStat(db_proxy, user_stat); err != nil {
			log.Error("InsertUserStat error, err = %v", err)
			return err
		}
	} else {
		if err = business_access.AddUserStatStep(db_proxy, eid, oid, date[0:10], delta_steps); err != nil {
			log.Error("AddUserStatStepDonation error, err = %v", err)
			return err
		}
	}
	return nil
}

func CalculateEmptyEidDonateDeltaStep(oid string, input_date_step map[string]int64) (int64, error) {
	nt := time.Now()
	sd := nt.Format("2006-01") + "-01"
	nt_date := nt.Format("2006-01-02")
	date_steps, err := business_access.GetStepsByDates(oid, sd, nt_date)
	if err != nil {
		log.Error("GetStepsByDates error, err = %v, oid = %s, sd = %s, ed = %s", err, oid, sd, nt_date)
		return 0, err
	}
	date_steps_map := make(map[string]int64)
	for _, it := range date_steps {
		date_steps_map[it.FDate] = it.FStep
	}
	delta_step := int64(0)
	for k, vl := range input_date_step {
		if _, ok := date_steps_map[k]; ok {
			delta_step += vl - date_steps_map[k]
		} else if k >= sd && k <= nt_date {
			delta_step += vl
		}
	}
	return delta_step, nil
}

func CalculateWeekDonateDeltaStep(oid, sd, ed string, inputDateStep map[string]int64) (int64, error) {
	dateSteps, err := business_access.GetStepsByDates(oid, sd, ed)
	if err != nil {
		log.Error("GetStepsByDates error, err = %v, oid = %s, sd = %s, ed = %s", err, oid, sd, ed)
		return 0, err
	}

	dateStepsMap := make(map[string]int64)
	for _, it := range dateSteps {
		dateStepsMap[it.FDate] = it.FStep
	}

	deltaStep := int64(0)
	for k, vl := range inputDateStep {
		if _, ok := dateStepsMap[k]; ok {
			deltaStep += vl - dateStepsMap[k]
		} else if k >= sd && k <= ed {
			deltaStep += vl
		}
	}
	return deltaStep, nil
}

func SendWeekFinishChallengeMsg(oid string, sum int64, nt, endTime time.Time, mapDistance int) error {
	// 计算是否完成本周里程
	ntDate := nt.Format("2006-01-02")
	send, err := cache_access.IsSendMsg(oid, ntDate)
	if err != nil {
		log.Error("IsSendMsg error, err = %v, oid = %s", err, oid)
		return err
	}
	if !send {
		stepDistance := int(float32(sum) * StepToMI)
		rank, num, err := cache_access.GetUserBigMapRankAndTotalNum(oid, nt)
		if err != nil {
			log.Error("GetUserBigMapRankAndTotalNum error, err = %v", err)
			return err
		}

		if stepDistance >= mapDistance {
			msg, err := GenerateWeekChallenge(&WeekChallenge{
				Oid:  oid,
				Step: sum,
				Rank: rank + 1,
				Num:  num,
				Time: nt.Format("2006-01-02 15:04:05"),
			})
			if err != nil {
				log.Error("GenerateWeekChallenge error, err = %v, oid = %s", err, oid)
				return err
			}
			if err := KafkaSendMsg(msg); err != nil {
				log.Error("KafkaSendMsg error, err = %v", err)
				return err
			}

			diff := endTime.Unix() - nt.Unix()
			if err := cache_access.SetSendMsg(oid, ntDate, diff); err != nil {
				log.Error("SetSendMsg error, err = %v, oid = %s", err, oid)
				// 重试一次
				_ = cache_access.SetSendMsg(oid, ntDate, diff)
			}
		}
	}
	return nil
}

func SendPOIViewPoint(oid, week, routeName string, weekStep int, poiList []PoiInfo, nt, endTime time.Time) error {
	size := len(poiList)
	for i := size - 1; i >= 0; i-- {
		if weekStep > poiList[i].Steps {
			send, err := cache_access.IsPoiMsgSend(oid, week, i)
			if err != nil {
				log.Error("IsPoiMsgSend error, err = %v, oid = %s", err, oid)
				return err
			}
			if !send {
				msg, err := GeneratePoiArriveData(&PoiArriveInfo{
					Oid:       oid,
					Week:      week,
					PoiName:   poiList[i].Name,
					RouteName: routeName,
					ArriveNum: i + 1,
				})
				if err != nil {
					log.Error("GeneratePoiArriveData error, err = %v, oid = %s", err, oid)
					return err
				}
				if err := KafkaSendMsg(msg); err != nil {
					log.Error("KafkaSendMsg error, err = %v", err)
					return err
				}
				if err := cache_access.SetPoiMsgSend(oid, week, i, endTime.Unix()-nt.Unix()); err != nil {
					log.Error("SetPoiMsgSend error, err = %v, oid = %s", err, oid)
					// 重试一次
					_ = cache_access.SetPoiMsgSend(oid, week, i, endTime.Unix()-nt.Unix())
				}
				break
			} else {
				break
			}
		}
	}
	return nil
}
