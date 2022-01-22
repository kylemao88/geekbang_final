//
package match

import (
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/coupons"
	"github.com/go-redis/redis"
)

const INT_MAX = int(^uint(0) >> 1)

const MATCH_RECORD_PREFIX = "matchID_"

const MATCH_MIN_TIME = "1970-01-16 00:00:00"
const MATCH_MIN_DATE = "1970-01-16"

// for redis prefix
const MATCH_USER_KEY_PREFIX = "yqz:fundmgr:match:user"

const USER_TOTAL_STEP = "f_total_match_step"
const USER_TOTAL_FUND = "f_total_match_fund"
const USER_TOTAL_TIME = "f_total_match_time"
const USER_COMBO = "f_match_combo"

const USER_ITEM = "f_match_item_id"
const USER_ORG = "f_match_org_id"
const USER_FUND = "f_match_fund"
const USER_STEP = "f_match_step"
const USER_OP_TYPE = "f_op_type"
const USER_DATE = "f_match_date"
const USER_MATCH_ID = "f_match_id"
const ACTIVITY_ID = "f_activity_id"

const ALL_FUNDS = "all_funds"
const ALL_TIMES = "all_times"
const ALL_DAYS = "all_days"
const ALL_MODIFY = "all_modify"

// lua script
const (
	UserMatchInfoCacheScript = `
redis.call("HMSET", KEYS[1], "f_match_date", ARGV[1], "f_match_combo", ARGV[2], "f_match_fund", ARGV[3], "f_match_step", ARGV[4], "f_match_item_id", ARGV[5], "f_op_type", ARGV[6], "f_match_id", ARGV[7])
local fund = redis.call("HINCRBY", KEYS[1], "f_total_match_fund", ARGV[3])
local step = redis.call("HINCRBY", KEYS[1], "f_total_match_step", ARGV[4])
local time = redis.call("HINCRBY", KEYS[1], "f_total_match_time", 1)
local result = {}
result[1] = fund
result[2] = step 
result[3] = time
return result
`
	UserSumMatchInfoCacheScript = `
local fund = redis.call("HINCRBY", KEYS[1], "all_funds", ARGV[1])
local time = redis.call("HINCRBY", KEYS[1], "all_times", 1)
local day = redis.call("HINCRBY", KEYS[1], "all_days", ARGV[2])
redis.call("HMSET", KEYS[1], "all_modify", ARGV[3])
local result = {}
result[1] = fund
result[2] = time 
result[3] = day
return result
`
	EXPIRE_TIME = "1440h"
)

// for monitor
var (
	SpeedSetCache    int64 = 0
	SpeedGetCache    int64 = 0
	SpeedSetDBRecord int64 = 0
	SpeedSetDBStatus int64 = 0
	SpeedGetDBRecord int64 = 0
	SpeedGetDBStatus int64 = 0
)

type UserMatchRecord struct {
	FMatchID     string `json:"f_match_id"`
	FUserID      string `json:"f_user_id"`
	FMatchItemID string `json:"f_match_item_id"`
	FMatchOrgID  string `json:"f_match_org_id"`
	FActivityID  string `json:"f_activity_id"`
	FMatchFund   int    `json:"f_match_fund"`
	FMatchStep   int    `json:"f_match_step"`
	FMatchCombo  int    `json:"f_match_combo"`
	FMatchDate   string `json:"f_match_date"`
	FDate        string `json:"f_date"`
	FOpType      int    `json:"f_op_type"`
	FStatus      int    `json:"f_status"`
	FCreateTime  string `json:"f_create_time"`
	FModifyTime  string `json:"f_modify_time"`
}

type OpType int32

const (
	OP_UNKNOW OpType = 0
	OP_NORMAL OpType = 1
	OP_BUFF   OpType = 2
)

var (
	NullMatchRecord = &metadata.MatchRecord{Date: MATCH_MIN_TIME}
	NullMatchStatus = &metadata.MatchStatus{Date: MATCH_MIN_TIME}
	cdbClient       dbclient.DBClient
	stopChan        chan struct{}
)

func InitComponent(db dbclient.DBClient, conf config.MatchConfig) error {
	cdbClient = db
	RunRecoverWorker(conf.RecoverCount, conf.RecoverIntervalMs, cacheclient.RedisClient)
	RunUpdateRankWorker(conf.UpdateRankInterval, cacheclient.RedisClient, db)
	coupons.RunCouponsWorker(conf.CouponsCount, conf.CouponsIntervalMs, cacheclient.RedisClient)
	return nil
}

func CloseComponent() {
	CloseRecoverWorker()
	CloseUpdateRankWorker()
}

// SetUserMatchInfoCache add combo times to cache
func SetUserMatchInfoCache(oid, activityID string, fund, step, combo, op int, item, lastMatchTime string, totalStep, totalFunds int64, totalTimes int, matchID string) error {
	atomic.AddInt64(&SpeedSetCache, 1)
	key := generateMatchKey(oid, activityID)
	pipe := cacheclient.RedisClient.Pipeline()
	// set value
	value := make(map[string]interface{})
	value[USER_DATE] = lastMatchTime
	value[USER_COMBO] = combo
	value[USER_FUND] = fund
	value[USER_STEP] = step
	value[USER_ITEM] = item
	value[USER_OP_TYPE] = op
	value[USER_TOTAL_FUND] = totalFunds
	value[USER_TOTAL_STEP] = totalStep
	value[USER_TOTAL_TIME] = totalTimes
	value[USER_MATCH_ID] = matchID
	pipe.HMSet(key, value)
	// set expire time
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	pipe.Expire(key, time.Second*remainSecond)
	// exec
	_, err := pipe.Exec()
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	logger.Debug("%v - key: %v, mathcID: %v, oid: %v, combo: %v, fund: %v, step: %v, t_fund: %v, t_step: %v, t_time: %v, last_time: %v",
		util.GetCallee(), key, matchID, oid, combo, fund, step, totalFunds, totalStep, totalTimes, lastMatchTime)
	return nil
}

// GetUserMatchInfoCache get user match info from cache
func GetUserMatchInfoCache(oid, activityID string) (*metadata.MatchRecord, error) {
	atomic.AddInt64(&SpeedGetCache, 1)
	key := generateMatchKey(oid, activityID)
	val, err := cacheclient.RedisClient.HGetAll(key).Result()
	// unExpect error
	if err != nil && err != redis.Nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - redis Get error, error = %v, key = %s", util.GetCallee(), err, key))
	}
	if err == redis.Nil || len(val) == 0 {
		logger.Info("%v - redis Get key = %s not exist", util.GetCallee(), key)
		return NullMatchRecord, errors.NewRedisNilError(errors.WithMsg("%v - redis Get key = %s not exist", util.GetCallee(), key))
	}
	// format result
	combo, err := strconv.Atoi(val[USER_COMBO])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_COMBO]))
	}
	step, err := strconv.Atoi(val[USER_STEP])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_STEP]))
	}
	fund, err := strconv.Atoi(val[USER_FUND])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_FUND]))
	}
	op, err := strconv.Atoi(val[USER_OP_TYPE])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_OP_TYPE]))
	}
	t_step, err := strconv.Atoi(val[USER_TOTAL_STEP])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_STEP]))
	}
	t_fund, err := strconv.Atoi(val[USER_TOTAL_FUND])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_FUND]))
	}
	t_times, err := strconv.Atoi(val[USER_TOTAL_TIME])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_TIME]))
	}
	result := &metadata.MatchRecord{
		Combo:      int32(combo),
		Steps:      int32(step),
		Funds:      int32(fund),
		Date:       val[USER_DATE],
		Item:       val[USER_ITEM],
		Op:         metadata.MatchRecord_OPType(op),
		ActivityId: activityID,
		TFunds:     int32(t_fund),
		TSteps:     int32(t_step),
		TTimes:     int32(t_times),
		Id:         val[USER_MATCH_ID],
	}
	logger.Debug("%v - get key: %v result: %v", util.GetCallee(), key, result)
	// reset expire time
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	cacheclient.RedisClient.Expire(key, time.Second*remainSecond)
	return result, nil
}

func clearTransaction(tx *sql.Tx) {
	err := cdbClient.TxRollback(tx)
	if err != sql.ErrTxDone && err != nil {
		logger.Error("%v", err)
	}
}

// AddUserMatchRecordDB set user match info into db
func AddUserMatchRecordDB(oid, activityID string, fund, step, combo, op int, item, org, matchDate string) (string, error) {
	atomic.AddInt64(&SpeedSetDBRecord, 1)
	// insert info to db
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	matchID, err := util.GetTransId(MATCH_RECORD_PREFIX)
	if err != nil {
		logger.Error("%v - get matchID error: %v", util.GetCallee(), err)
		return "", err
	}
	matchTime, err := time.ParseInLocation("2006-01-02 15:04:05", matchDate, util.Loc)
	if err != nil {
		logger.Error("%v - parse match time: %v err: %v", matchDate, err)
		return "", err
	}
	date := matchTime.In(util.Loc).Format("2006-01-02")
	// add record first, future we should change tdsql
	data := UserMatchRecord{
		FMatchID:     matchID,
		FUserID:      oid,
		FMatchItemID: item,
		FMatchOrgID:  org,
		FActivityID:  activityID,
		FMatchFund:   fund,
		FMatchStep:   step,
		FMatchCombo:  combo,
		FOpType:      op,
		FMatchDate:   matchDate,
		FDate:        date,
		FCreateTime:  nowTime,
		FModifyTime:  nowTime,
	}
	err = cdbClient.Insert("t_user_match_record", data)
	if err != nil {
		return "", errors.NewDBClientError(errors.WithMsg("%v - Insert user record error, err = %v", util.GetCallee(), err))
	}
	/*
		tx, err := cdbClient.TxBegin()
		if err != nil {
			clearTransaction(tx.(*sql.Tx))
			return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
		}
		err = cdbClient.TxInsert(tx, "t_user_match_record", data)
		if err != nil {
			clearTransaction(tx.(*sql.Tx))
			return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
		}
		err = cdbClient.TxCommit(tx)
		if err != nil {
			logger.Error("%v - commit error: %v", util.GetCallee(), err)
			clearTransaction(tx.(*sql.Tx))
			return errors.NewDBClientError(errors.WithMsg("%v - commit error, err = %v, err", util.GetCallee(), err))
		}
	*/
	logger.Debug("%v - user record: %v", util.GetCallee(), data)
	return matchID, nil
}

// UpdateUserMatchStatusDB set user match info into db
func UpdateUserMatchStatusDB(oid, activity string, fund, step, combo, op int, item, org, date string, t_fund, t_step, t_time int64) error {
	atomic.AddInt64(&SpeedSetDBStatus, 1)
	// insert info to db
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	/*
		tx, err := cdbClient.TxBegin()
		if err != nil {
			clearTransaction(tx.(*sql.Tx))
			return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
		}
	*/
	// update user state
	sqlStr := "INSERT INTO t_user_match_status (f_user_id, f_activity_id, f_total_match_fund, f_total_match_step, f_total_match_time, f_match_combo, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=?, f_total_match_step=?, f_total_match_time=?, f_match_combo=? ,f_modify_time=?"
	args := make([]interface{}, 0)
	args = append(args, oid, activity, fund, step, 1, 1, nowTime, nowTime,
		t_fund, t_step, t_time, combo, nowTime)
	//res, err := cdbClient.TxExecSQL(tx, sqlStr, args)
	res, err := cdbClient.ExecSQL(sqlStr, args)
	if err != nil {
		//clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	/*
		err = cdbClient.TxCommit(tx)
		if err != nil {
			logger.Error("%v - commit error: %v", util.GetCallee(), err)
			clearTransaction(tx.(*sql.Tx))
			return errors.NewDBClientError(errors.WithMsg("%v - commit error, err = %v, err", util.GetCallee(), err))
		}
	*/
	logger.Debug("%v - user: %v affect result count: %v", util.GetCallee(), oid, res)
	return nil
}

// GetUserMatchLastRecordDB get user match info from db
func GetUserMatchLastRecordDB(oid, activityID string) (*metadata.MatchRecord, error) {
	result, err := GetUserMatchRecordByOffsetDB(oid, activityID, 0, 1, false)
	if err != nil {
		logger.Error("%v", err)
		return nil, err
	}
	// no match record just return empty record
	if len(result) == 0 {
		logger.Info("%v - user: %v get no match record before", util.GetCallee(), oid)
		return NullMatchRecord, errors.NewDBNilError(errors.WithMsg("%v - user: %v get no match record before", util.GetCallee(), oid))
	}
	logger.Debug("%v - user: %v get match record: %v", util.GetCallee(), oid, result[0])
	return result[0], nil
}

// GetUserMatchRecordDB get user match info from db
//func GetUserMatchRecordDB(oid, startTime, endTime string) ([]*UserMatchInfo, error) {
func GetUserMatchRecordDB(oid, activityID, startTime, endTime string) ([]*metadata.MatchRecord, error) {
	atomic.AddInt64(&SpeedGetDBRecord, 1)
	weekStart, weekEnd := util.WeekStartEnd(time.Now())
	if len(startTime) == 0 {
		startTime = weekStart.In(util.Loc).Format("2006-01-02 15:04:05")
	}
	if len(endTime) == 0 {
		endTime = weekEnd.In(util.Loc).Format("2006-01-02 15:04:05")
	}
	// cdb get info
	sql_str := "select f_match_id, f_match_fund, f_match_step, f_match_combo, f_match_item_id, f_match_org_id, f_op_type, f_match_date from t_user_match_record where f_user_id = ? and f_match_date BETWEEN ? and ? and f_activity_id = ?"
	args := make([]interface{}, 0)
	args = append(args, oid, startTime, endTime, activityID)
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error, err = %v, err", util.GetCallee(), err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("%v - user: %v  range: %v->%v get no match record before", util.GetCallee(), oid, startTime, endTime)
		return nil, nil
	}

	// format result
	var result []*metadata.MatchRecord
	for _, value := range res {
		fund, err := strconv.Atoi(value[USER_FUND])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_FUND]))
		}

		step, err := strconv.Atoi(value[USER_STEP])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_STEP]))
		}

		combo, err := strconv.Atoi(value[USER_COMBO])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_COMBO]))
		}

		op, err := strconv.Atoi(value[USER_OP_TYPE])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_OP_TYPE]))
		}

		result = append(result, &metadata.MatchRecord{
			Combo:        int32(combo),
			Funds:        int32(fund),
			Steps:        int32(step),
			Date:         value[USER_DATE],
			Item:         value[USER_ITEM],
			Organization: value[USER_ORG],
			Id:           value[USER_MATCH_ID],
			ActivityId:   value[ACTIVITY_ID],
			Op:           metadata.MatchRecord_OPType(op),
		})
	}
	logger.Debug("%v - user: %v, result: %v", util.GetCallee(), oid, result)
	return result, nil
}

func GetUserMatchRecordByOffsetDB(oid, activity string, offset, size int, all bool) ([]*metadata.MatchRecord, error) {
	var result []*metadata.MatchRecord
	if size == 0 {
		logger.Debugf("oid: %v ac: %v params size = 0, skip record search", oid, activity)
		return result, nil
	}
	atomic.AddInt64(&SpeedGetDBRecord, 1)
	// cdb get info
	var sql_str string
	args := make([]interface{}, 0)
	if all {
		sql_str = "select f_match_id, f_match_fund, f_match_step, f_match_combo, f_match_item_id, f_match_org_id, f_op_type, f_match_date, f_activity_id from t_user_match_record where f_user_id = ? order by f_match_date desc limit ?, ?"
		args = append(args, oid, offset, size)
	} else {
		sql_str = "select f_match_id, f_match_fund, f_match_step, f_match_combo, f_match_item_id, f_match_org_id, f_op_type, f_match_date, f_activity_id from t_user_match_record where f_user_id = ? and f_activity_id = ? order by f_match_date desc limit ?, ?"
		args = append(args, oid, activity, offset, size)
	}
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error, err = %v, err", util.GetCallee(), err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("%v - user: %v, offset: %v, size: %v get no match record before", util.GetCallee(), oid, offset, size)
		return nil, nil
	}

	// format result
	for _, value := range res {
		fund, err := strconv.Atoi(value[USER_FUND])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_FUND]))
		}

		step, err := strconv.Atoi(value[USER_STEP])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_STEP]))
		}

		combo, err := strconv.Atoi(value[USER_COMBO])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_COMBO]))
		}

		op, err := strconv.Atoi(value[USER_OP_TYPE])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value[USER_OP_TYPE]))
		}

		result = append(result, &metadata.MatchRecord{
			Combo:        int32(combo),
			Funds:        int32(fund),
			Steps:        int32(step),
			Date:         value[USER_DATE],
			Item:         value[USER_ITEM],
			Organization: value[USER_ORG],
			Id:           value[USER_MATCH_ID],
			ActivityId:   value[ACTIVITY_ID],
			Op:           metadata.MatchRecord_OPType(op),
		})
	}
	logger.Debug("%v - user: %v, get %v result: %v", util.GetCallee(), oid, len(result), result)
	return result, nil
}

// GetUserMatchStatusCache return user total match status
func GetUserMatchStatusCache(oid, activityID string) (*metadata.MatchStatus, error) {
	atomic.AddInt64(&SpeedGetCache, 1)
	key := generateMatchKey(oid, activityID)
	val, err := cacheclient.RedisClient.HGetAll(key).Result()
	// unExpect error
	if err != nil && err != redis.Nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - redis Get error, error = %v, key = %s", util.GetCallee(), err, key))
	}
	if err == redis.Nil || len(val) == 0 {
		logger.Info("%v - redis Get key = %s not exist", util.GetCallee(), key)
		return &metadata.MatchStatus{}, errors.NewRedisNilError(errors.WithMsg("%v - redis Get key = %s not exist", util.GetCallee(), key))
	}
	// format result
	times, err := strconv.Atoi(val[USER_TOTAL_TIME])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_TIME]))
	}
	step, err := strconv.ParseInt(val[USER_TOTAL_STEP], 10, 64)
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.ParseInt error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_STEP]))
	}
	fund, err := strconv.ParseInt(val[USER_TOTAL_FUND], 10, 64)
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - strconv.ParseInt error, error = %v, val = %s", util.GetCallee(), err, val[USER_TOTAL_FUND]))
	}
	result := &metadata.MatchStatus{
		Funds: fund,
		Steps: step,
		Times: int32(times),
	}
	logger.Debug("%v - get user: %v result: %v", oid, util.GetCallee(), result)
	// reset expire time
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	cacheclient.RedisClient.Expire(key, time.Second*remainSecond)
	return result, nil
}

// GetUserMatchStatusDB return user total match status
func GetUserMatchStatusDB(oid, activityID string) (*metadata.MatchStatus, error) {
	atomic.AddInt64(&SpeedGetDBStatus, 1)
	// cdb get info
	sql_str := "select f_total_match_fund, f_total_match_step, f_total_match_time from t_user_match_status where f_user_id = ? and f_activity_id = ?"
	args := make([]interface{}, 0)
	args = append(args, oid, activityID)
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error, err = %v, err", util.GetCallee(), err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("%v - user: %v, get no match status", util.GetCallee(), oid)
		return &metadata.MatchStatus{}, errors.NewDBNilError(errors.WithMsg("%v - user: %v, get no match status", util.GetCallee(), oid))
	}

	// format result
	var fund, step int64
	var time int
	for _, value := range res {
		fund, err = strconv.ParseInt(value["f_total_match_fund"], 10, 64)
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.ParseInt error, error = %v, val = %s", util.GetCallee(), err, value["f_total_match_time"]))
		}

		step, err = strconv.ParseInt(value["f_total_match_step"], 10, 64)
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.ParseInt error, error = %v, val = %s", util.GetCallee(), err, value["f_total_match_time"]))
		}

		time, err = strconv.Atoi(value["f_total_match_time"])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("%v - strconv.Atoi error, error = %v, val = %s", util.GetCallee(), err, value["f_total_match_time"]))
		}
	}
	result := &metadata.MatchStatus{
		Funds: fund,
		Steps: step,
		Times: int32(time),
	}
	logger.Debug("%v - user:%v get result: %v", util.GetCallee(), oid, result)
	return result, nil
}

func filterRules(antiRules *metadata.AntiBlackRules, step int32) (bool, int) {
	todayDate := time.Now().In(util.Loc).Format("2006-01-02")
	for _, value := range antiRules.AntiBlacks {
		start := todayDate + " " + value.StartTime
		end := todayDate + " " + value.EndTime
		result, err := util.CheckCurrentBetweenTime(start, end)
		if err != nil {
			logger.Error("%v - err: %v", util.GetCallee(), err)
			continue
		}
		if result && step > int32(value.Steps) {
			return true, int(value.Money)
		}
	}
	return false, 0
}

// GenerateMatchFund get current match fund, return matchFund, op, combo, lastMatchFmt
// if return fund=0, err=nil, mean user have already match today
func GenerateMatchFund(oid, activityID string, step int32, matchItem *metadata.MatchActivity, info *metadata.MatchRecord) (int, OpType, int, string, error) {
	var matchFund int
	var combo int
	var op OpType = OP_NORMAL
	// judge the record is today or not
	timeNowFmt := time.Now().In(util.Loc).Format("2006-01-02")
	beforeTime, _ := time.ParseDuration("-24h")
	timeYesterdayFmt := time.Now().Add(beforeTime).In(util.Loc).Format("2006-01-02")
	matchTime, err := time.ParseInLocation("2006-01-02 15:04:05", info.Date, util.Loc)
	if err != nil {
		logger.Error("parse match time: %v err: %v", info.Date, err)
		return -1, OP_UNKNOW, 0, "", err
	}
	lastMatchFmt := matchTime.In(util.Loc).Format("2006-01-02")
	//logger.Debug("now: %v , yesterday:%v,  last: %v", timeNowFmt, timeYesterdayFmt, lastMatchFmt)
	if lastMatchFmt == timeYesterdayFmt {
		combo = int(info.Combo)
		logger.Info("user: %v have matched yesterday: %v", oid, lastMatchFmt)
	} else if lastMatchFmt == timeNowFmt {
		combo = int(info.Combo)
		logger.Info("user: %v have matched today: %v", oid, lastMatchFmt)
		return int(info.Funds), OpType(info.Op), combo, lastMatchFmt, errors.NewFundMgrAlreadyMatchError()
	} else {
		combo = 0
		logger.Info("user: %v have no combo now, last match: %v", oid, lastMatchFmt)
	}
	// switch case rule
	itemRemainFund := matchItem.MatchStats.Remain
	var needFilter bool
	switch matchItem.MatchRule.RuleType {
	case metadata.MatchRule_REMAIN:
		// remain type
		if matchItem.MatchRule.RemainMatchRules == nil {
			return -1, OP_UNKNOW, 0, "", errors.NewFundMgrInternalError(errors.WithMsg("mode is [remain], but no rule"))
		}
		for _, rule := range matchItem.MatchRule.RemainMatchRules.RemainMatchRules {
			start := int(rule.IntervalStart)
			end := int(rule.IntervalEnd)
			if end == -1 {
				end = INT_MAX
			}
			if util.CheckBetweenExcludeMax(start, end, int(itemRemainFund)) {
				matchFund = util.GenerateBetweenNum(int(rule.MatchStart), int(rule.MatchEnd))
			}
			if matchFund > int(itemRemainFund) {
				matchFund = int(itemRemainFund)
			}
		}
	case metadata.MatchRule_BUFF_PERCENT:
		// percent type
		if matchItem.MatchRule.BuffPercentRule == nil {
			return -1, OP_UNKNOW, 0, "", errors.NewFundMgrInternalError(errors.WithMsg("mode is [percent], but no rule"))
		}
		buffMeta := matchItem.MatchRule.BuffPercentRule.BuffMeta
		percent := int(matchItem.MatchRule.BuffPercentRule.BuffPercent)
		var basicFund, basicWave, buffFund, buffWave = int(buffMeta.BasicFund), int(buffMeta.BasicWave), int(buffMeta.BuffFund), int(buffMeta.BuffWave)
		if basicFund < basicWave {
			logger.Info("%v - basic: %v smaller than wave: %v, just return basic fund", basicFund, basicWave)
			if basicFund > int(itemRemainFund) {
				return int(itemRemainFund), OP_NORMAL, combo, lastMatchFmt, nil
			}
			return basicFund, OP_NORMAL, combo, lastMatchFmt, nil
		}
		randNum := rand.Intn(100)
		if randNum < percent {
			matchFund = buffFund + (rand.Intn(buffWave*2+1) - buffWave)
			op = OP_BUFF
		} else {
			matchFund = basicFund + (rand.Intn(basicWave*2+1) - basicWave)
		}
		if matchFund > int(itemRemainFund) {
			matchFund = int(itemRemainFund)
		}
	case metadata.MatchRule_BUFF_COMBO:
		// combo type
		needFilter = true
		if matchItem.MatchRule.BuffComboRule == nil {
			return -1, OP_UNKNOW, 0, "", errors.NewFundMgrInternalError(errors.WithMsg("mode is [percent], but no rule"))
		}
		var buffMatch bool
		buffMeta := matchItem.MatchRule.BuffComboRule.BuffMeta
		buffThreshold := matchItem.MatchRule.BuffComboRule.BuffThreshold
		basicFund, basicWave, buffFund, buffWave, comboLimit := int(buffMeta.BasicFund), int(buffMeta.BasicWave), int(buffMeta.BuffFund), int(buffMeta.BuffWave), int(buffThreshold)
		if basicFund < basicWave {
			logger.Info("%v - basic: %v smaller than wave: %v, just return basic fund", basicFund, basicWave)
			if basicFund > int(itemRemainFund) {
				return int(itemRemainFund), OP_NORMAL, combo, lastMatchFmt, nil
			}
			return basicFund, OP_NORMAL, combo, lastMatchFmt, nil
		}
		if lastMatchFmt == timeYesterdayFmt {
			if comboLimit == 0 {
				logger.Info("%v - combo limit: %v user get buff everyday", util.GetCallee(), comboLimit)
				buffMatch = true
			} else if (combo+1 >= comboLimit) && ((combo+1)%comboLimit == 0) {
				buffMatch = true
			}
		}
		// check fund
		if buffMatch {
			matchFund = buffFund + (rand.Intn(buffWave*2+1) - buffWave)
			op = OP_BUFF
		} else {
			matchFund = basicFund + (rand.Intn(basicWave*2+1) - basicWave)
		}
		if matchFund > int(itemRemainFund) {
			matchFund = int(itemRemainFund)
		}
	case metadata.MatchRule_STEP_RELATIVE:
		// step relative type
		needFilter = true
		if matchItem.MatchRule.StepRelativeRule == nil {
			return -1, OP_UNKNOW, 0, "", errors.NewFundMgrInternalError(errors.WithMsg("mode is [step_relative], but no rule"))
		}
		if step >= matchItem.MatchRule.StepRelativeRule.MaxSteps || matchItem.MatchRule.StepRelativeRule.MaxSteps == matchItem.MatchRule.StepRelativeRule.MinSteps {
			matchFund = int(matchItem.MatchRule.StepRelativeRule.MaxMatch)
		} else {
			matchFund = int(matchItem.MatchRule.StepRelativeRule.MinMatch + (matchItem.MatchRule.StepRelativeRule.MaxMatch-matchItem.MatchRule.StepRelativeRule.MinMatch)*(step-matchItem.MatchRule.StepRelativeRule.MinSteps)/
				(matchItem.MatchRule.StepRelativeRule.MaxSteps-matchItem.MatchRule.StepRelativeRule.MinSteps))
		}
		if matchFund > int(itemRemainFund) {
			matchFund = int(itemRemainFund)
		}
	default:
		logger.Error("unknow match rule type: %v", matchItem.MatchRule.RuleType)
		return -1, OP_UNKNOW, 0, "", errors.NewFundMgrMatchGenerateError(errors.WithMsg("unknown match rule type: %v", matchItem.MatchRule.RuleType))
	}
	// add filter rule
	if needFilter {
		antiRules := matchItem.MatchInfo.FAntiBlack
		filterResult, filterFund := filterRules(antiRules, step)
		if filterResult {
			matchFund = filterFund
			logger.Debug("%v - user: %v step: %v is suspicious, set matchFund: %v", util.GetCallee(), oid, step, matchFund)
		}
	}
	logger.Debug("user: %v rule: %v get match fund: %v buff: %v combo: %v", oid, matchItem.MatchRule.RuleType, matchFund, op, combo)
	return matchFund, op, combo, lastMatchFmt, nil
}

// SetUserSummaryMatchInfoCache ...
func SetUserSummaryMatchInfoCache(oid, date string, funds, days, times int) error {
	atomic.AddInt64(&SpeedSetCache, 1)
	key := generateSummaryMatchKey(oid)
	pipe := cacheclient.RedisClient.Pipeline()
	// set value
	value := make(map[string]interface{})
	value[ALL_DAYS] = days
	value[ALL_FUNDS] = funds
	value[ALL_TIMES] = times
	value[ALL_MODIFY] = date
	pipe.HMSet(key, value)
	/*
		// set expire time 7 days
		timeNow := time.Now()
		afterTime, _ := time.ParseDuration(EXPIRE_TIME)
		time7Day := timeNow.Add(afterTime)
		remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
		pipe.Expire(key, time.Second*remainSecond)
	*/
	// exec
	_, err := pipe.Exec()
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("err: %v", util.GetCallee(), err))
	}
	logger.Debug("key: %v, oid: %v, funds: %v, days: %v, date: %v",
		key, oid, funds, days, date)
	return nil
}

// GetUserSummaryMatchInfoCache get user match info from cache
func GetUserSummaryMatchInfoCache(oid string) (*metadata.MatchStatus, error) {
	atomic.AddInt64(&SpeedGetCache, 1)
	key := generateSummaryMatchKey(oid)
	val, err := cacheclient.RedisClient.HGetAll(key).Result()
	// unExpect error
	if err != nil && err != redis.Nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("%v - redis Get error, error = %v, key = %s", util.GetCallee(), err, key))
	}
	if err == redis.Nil || len(val) == 0 {
		logger.Info("%v - redis Get key = %s not exist", util.GetCallee(), key)
		return nil, errors.NewRedisNilError(errors.WithMsg("%v - redis Get key = %s not exist", util.GetCallee(), key))
	}
	// format result
	funds, err := strconv.Atoi(val[ALL_FUNDS])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("strconv.Atoi error, error = %v, val = %s", err, val[ALL_FUNDS]))
	}
	days, err := strconv.Atoi(val[ALL_DAYS])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("strconv.Atoi error, error = %v, val = %s", err, val[ALL_DAYS]))
	}
	times, err := strconv.Atoi(val[ALL_TIMES])
	if err != nil {
		return nil, errors.NewRedisClientError(errors.WithMsg("strconv.Atoi error, error = %v, val = %s", err, val[ALL_TIMES]))
	}
	result := &metadata.MatchStatus{
		Funds: int64(funds),
		Times: int32(times),
		Days:  int32(days),
		Date:  val[ALL_MODIFY],
	}
	logger.Debug("get key: %v result: %v", key, result)
	// reset expire time
	cacheclient.RedisClient.Persist(key)
	return result, nil
}

// RecoverUserSummaryMatchStatusDB return user total match status
func RecoverUserSummaryMatchStatusDB(oid string) (*metadata.MatchStatus, error) {
	atomic.AddInt64(&SpeedGetDBRecord, 1)
	// cdb get info
	sql_str := "select f_match_fund, f_match_date from t_user_match_record where f_user_id = ?"
	args := make([]interface{}, 0)
	args = append(args, oid)
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error, err = %v, err", util.GetCallee(), err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("user %v get no match record", oid)
		return &metadata.MatchStatus{Date: MATCH_MIN_DATE}, errors.NewDBNilError(errors.WithMsg("user %v get no match record", oid))
	}

	// format result
	var t_fund int64
	var t_times int
	var lastDate string = MATCH_MIN_DATE
	var statistic = make(map[string]int)
	for _, value := range res {
		fund, err := strconv.ParseInt(value["f_match_fund"], 10, 64)
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("strconv.ParseInt error, error = %v, val = %s", err, value["f_total_match_time"]))
		}
		matchDate, ok := value["f_match_date"]
		if !ok {
			return nil, errors.NewDBClientError(errors.WithMsg("no f_match_date field"))
		}
		matchTime, err := time.ParseInLocation("2006-01-02 15:04:05", matchDate, util.Loc)
		if err != nil {
			logger.Error("parse match time: %v err: %v", matchDate, err)
			return nil, errors.NewDBClientError(errors.WithMsg("ParseInLocation error = %v, val = %s", err, matchDate))
		}
		date := matchTime.In(util.Loc).Format("2006-01-02")
		t_fund += fund
		t_times++
		statistic[date]++
		if date > lastDate {
			lastDate = date
		}
	}
	result := &metadata.MatchStatus{
		Funds: t_fund,
		Times: int32(t_times),
		Days:  int32(len(statistic)),
		Date:  lastDate,
	}
	logger.Debug("%v - user:%v get result: %v", util.GetCallee(), oid, result)
	return result, nil
}

// UpdateUserAllMatchInfoCache ...
// return: ac_total_fund, ac_total_step, ac_total_time, sum_total_fund, sum_total_day, sum_total_time
func UpdateUserAllMatchInfoCache(oid, activityID string, fund, step, combo, op int, item, lastMatchTime, todayDate string, days int, matchID string) (int64, int64, int64, int64, int64, int64, error) {
	atomic.AddInt64(&SpeedSetCache, 1)
	key := generateMatchKey(oid, activityID)
	comboStr := strconv.Itoa(combo)
	fundStr := strconv.Itoa(fund)
	stepStr := strconv.Itoa(step)
	opStr := strconv.Itoa(op)
	result, err := redisClient.Eval(UserMatchInfoCacheScript, []string{key}, []string{lastMatchTime, comboStr, fundStr, stepStr, item, opStr, matchID}).Result()
	if err != nil {
		logger.Error("redisClient.Eval error: %v", err)
		return 0, 0, 0, 0, 0, 0, errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	incrFund, ok := result.([]interface{})[0].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	incrStep, ok := result.([]interface{})[1].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	incrTime, ok := result.([]interface{})[2].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	/*
		// set activity value
		value := make(map[string]interface{})
		value[USER_DATE] = lastMatchTime
		value[USER_COMBO] = combo
		value[USER_FUND] = fund
		value[USER_STEP] = step
		value[USER_ITEM] = item
		value[USER_OP_TYPE] = op
		value[USER_MATCH_ID] = matchID
		incrFund := pipe.HIncrBy(key, USER_TOTAL_FUND, int64(fund))
		incrStep := pipe.HIncrBy(key, USER_TOTAL_STEP, int64(step))
		incrTime := pipe.HIncrBy(key, USER_TOTAL_TIME, 1)
		pipe.HMSet(key, value)
	*/

	// set summary value
	keySummary := generateSummaryMatchKey(oid)
	dayStr := strconv.Itoa(days)
	result, err = redisClient.Eval(UserSumMatchInfoCacheScript, []string{keySummary}, []string{fundStr, dayStr, todayDate}).Result()
	if err != nil {
		logger.Error("redisClient.Eval error: %v", err)
		return 0, 0, 0, 0, 0, 0, errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	incrSumFund, ok := result.([]interface{})[0].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	incrSumTime, ok := result.([]interface{})[1].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	incrSumDays, ok := result.([]interface{})[2].(int64)
	if !ok {
		return 0, 0, 0, 0, 0, 0, errors.NewFundMgrInternalError(errors.WithMsg("%v - err redis: %v", util.GetCallee(), result))
	}
	/*
		incrSumFund := pipe.HIncrBy(keySummary, ALL_FUNDS, int64(fund))
		incrSumTime := pipe.HIncrBy(keySummary, ALL_TIMES, 1)
		incrSumDays := pipe.HIncrBy(keySummary, ALL_DAYS, int64(days))
		if days != 0 {
			value := make(map[string]interface{})
			value[ALL_MODIFY] = todayDate
			pipe.HMSet(keySummary, value)
		}
	*/
	// set expire time 7 days
	pipe := cacheclient.RedisClient.Pipeline()
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	pipe.Expire(key, time.Second*remainSecond)
	//pipe.Expire(keySummary, time.Second*remainSecond)
	cacheclient.RedisClient.Persist(keySummary)
	_, err = pipe.Exec()
	if err != nil {
		return 0, 0, 0, 0, 0, 0, errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	logger.Debug("key: %v, combo: %v, fund: %v, step: %v, t_fund: %v, t_step: %v, t_times: %v, last_time: %v",
		//key, combo, fund, step, incrFund.Val(), incrStep.Val(), incrTime.Val(), lastMatchTime)
		key, combo, fund, step, incrFund, incrStep, incrTime, lastMatchTime)
	logger.Debug("keySum: %v, days: %v date: %v t_fund: %v, t_day: %v, t_time: %v",
		//keySummary, days, todayDate, incrSumFund.Val(), incrSumDays.Val(), incrSumTime.Val())
		keySummary, days, todayDate, incrSumFund, incrSumDays, incrSumTime)
	//return incrFund, incrStep, incrTime, incrSumFund.Val(), incrSumDays.Val(), incrSumTime.Val(), nil
	return incrFund, incrStep, incrTime, incrSumFund, incrSumDays, incrSumTime, nil
}

// UpdateUserAllMatchStatusDB set user match info into db
func UpdateUserAllMatchStatusDB(oid, activity string, combo, op int, item, org, date string, t_fund, t_step, t_time int64, todayDate string, sum_fund, sum_day, sum_time int) error {
	/*
		// update user summary state
		err := UpdateUserSummaryMatchStatusDB(oid, todayDate, int64(sum_fund), int64(sum_day), int64(sum_time))
		if err != nil {
			logger.Error("UpdateUserSummaryMatchStatusDB error: %v, oid: %v", err, oid)
			return err
		}
	*/
	// update user activity state
	err := UpdateUserActivityMatchStatusDB(oid, activity, combo, t_fund, t_step, t_time)
	if err != nil {
		logger.Error("UpdateUserActivityMatchStatusDB error: %v, oid: %v", err, oid)
		return err
	}
	return nil
}

// UpdateUserAllMatchStatusDB set user match info into db
func UpdateUserActivityMatchStatusDB(oid, activity string, combo int, totalFund, totalStep, totalTime int64) error {
	atomic.AddInt64(&SpeedSetDBStatus, 1)
	// update user activity state
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sqlStr := "INSERT INTO t_user_match_status (f_user_id, f_activity_id, f_total_match_fund, f_total_match_step, f_total_match_time, f_match_combo, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=VALUES(f_total_match_fund), f_total_match_step=VALUES(f_total_match_step), f_total_match_time=VALUES(f_total_match_time), f_match_combo=VALUES(f_match_combo) ,f_modify_time=VALUES(f_modify_time)"
	args := make([]interface{}, 0)
	args = append(args, oid, activity, totalFund, totalStep, totalTime, combo, nowTime, nowTime)
	resAc, err := cdbClient.ExecSQL(sqlStr, args)
	if err != nil {
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	logger.Debug("%v - user: %v affect  count: %v", util.GetCallee(), oid, resAc)
	return nil
}

func generateMatchKey(oid, activityID string) string {
	if len(activityID) == 0 {
		return fmt.Sprintf("%s_%s", MATCH_USER_KEY_PREFIX, oid)
	}
	return fmt.Sprintf("%s_%s:%s", MATCH_USER_KEY_PREFIX, oid, activityID)
}

func generateSummaryMatchKey(oid string) string {
	return fmt.Sprintf("%s:%s", MATCH_USER_KEY_PREFIX, oid)
}

/*
// UpdateUserMatchInfoCache return total_fund, total_step, total_time
func UpdateUserMatchInfoCache(oid, activityID string, fund, step, combo, op int, item, lastMatchTime, matchID string) (int64, int64, int64, error) {
	atomic.AddInt64(&SpeedSetCache, 1)
	key := generateMatchKey(oid, activityID)
	pipe := cacheclient.RedisClient.Pipeline()
	// set activity value
	value := make(map[string]interface{})
	value[USER_DATE] = lastMatchTime
	value[USER_COMBO] = combo
	value[USER_FUND] = fund
	value[USER_STEP] = step
	value[USER_ITEM] = item
	value[USER_OP_TYPE] = op
	incrFund := pipe.HIncrBy(key, USER_TOTAL_FUND, int64(fund))
	incrStep := pipe.HIncrBy(key, USER_TOTAL_STEP, int64(step))
	incrTime := pipe.HIncrBy(key, USER_TOTAL_TIME, 1)
	value[USER_MATCH_ID] = matchID
	pipe.HMSet(key, value)
	// set expire time 7 days
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	pipe.Expire(key, time.Second*remainSecond)
	// exec
	_, err := pipe.Exec()
	if err != nil {
		return 0, 0, 0, errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	logger.Debug("%v - key: %v, matchID: %v, oid: %v, combo: %v, fund: %v, step: %v, t_fund: %v, t_step: %v, t_times: %v, last_time: %v",
		util.GetCallee(), key, matchID, oid, combo, fund, step, incrFund.Val(), incrStep.Val(), incrTime.Val(), lastMatchTime)
	return incrFund.Val(), incrStep.Val(), incrTime.Val(), nil
}
*/

/*
// AddUserMatchRecordAndStatusDB set user match info into db
func AddUserMatchRecordAndStatusDB(oid, activity string, fund, step, combo, op int, item, org, date string, t_fund, t_step, t_time int64) error {
	// insert info to db
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	matchID, err := util.GetTransId(MATCH_RECORD_PREFIX)
	if err != nil {
		logger.Error("%v - get matchID error: %v", util.GetCallee(), err)
		return err
	}

	// add record first, future we should change tdsql
	data := UserMatchRecord{
		FMatchID:     matchID,
		FUserID:      oid,
		FMatchItemID: item,
		FMatchOrgID:  org,
		FMatchFund:   fund,
		FMatchStep:   step,
		FMatchCombo:  combo,
		FOpType:      op,
		FMatchDate:   date,
		FCreateTime:  nowTime,
		FModifyTime:  nowTime,
	}
	tx, err := cdbClient.TxBegin()
	if err != nil {
		clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
	}
	err = cdbClient.TxInsert(tx, "t_user_match_record", data)
	if err != nil {
		clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
	}

	// update user state
	sqlStr := "INSERT INTO t_user_match_status (f_user_id, f_activity_id, f_total_match_fund, f_total_match_step, f_total_match_time, f_match_combo, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=?, f_total_match_step=?, f_total_match_time=?, f_match_combo=? ,f_modify_time=?"
	args := make([]interface{}, 0)
	args = append(args, oid, activity, fund, step, 1, 1, nowTime, nowTime,
		t_fund, t_step, t_time, combo, nowTime)
	res, err := cdbClient.TxExecSQL(tx, sqlStr, args)
	if err != nil {
		clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	err = cdbClient.TxCommit(tx)
	if err != nil {
		logger.Error("%v - commit error: %v", util.GetCallee(), err)
		clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - commit error, err = %v, err", util.GetCallee(), err))
	}
	logger.Debug("%v - user: %v affect result count: %v", util.GetCallee(), oid, res)
	return nil
}
*/

/*
// UpdateUserSummaryMatchInfoCache return total_fund, total_day, total_time
func UpdateUserSummaryMatchInfoCache(oid, date string, funds, days int) (int64, int64, int64, error) {
	atomic.AddInt64(&SpeedSetCache, 1)
	key := generateSummaryMatchKey(oid)
	pipe := cacheclient.RedisClient.Pipeline()
	// set value
	incrFund := pipe.HIncrBy(key, ALL_FUNDS, int64(funds))
	incrTime := pipe.HIncrBy(key, ALL_TIMES, 1)
	incrDays := pipe.HIncrBy(key, ALL_DAYS, int64(days))
	if days != 0 {
		value := make(map[string]interface{})
		value[ALL_MODIFY] = date
		pipe.HMSet(key, value)
	}
	// set expire time 7 days
	timeNow := time.Now()
	afterTime, _ := time.ParseDuration(EXPIRE_TIME)
	time7Day := timeNow.Add(afterTime)
	remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
	pipe.Expire(key, time.Second*remainSecond)
	// exec
	_, err := pipe.Exec()
	if err != nil {
		return -1, -1, -1, errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	logger.Debug("key: %v, funds: %v, days: %v, date: %v, t_funds: %v, f_times: %v, t_days: %v",
		key, funds, days, date, incrFund.Val(), incrTime.Val(), incrDays.Val())
	return incrFund.Val(), incrTime.Val(), incrDays.Val(), nil
}
*/

/*
// UpdateUserSummaryMatchStatusDB set user match info into db
func UpdateUserSummaryMatchStatusDB(oid string, todayDate string, sumFund, sumDay, sumTime int64) error {
	atomic.AddInt64(&SpeedSetDBStatus, 1)
	// insert info to db
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sqlStr := "INSERT INTO t_user_match_summary (f_user_id,  f_total_match_fund, f_total_match_time, f_total_match_day, f_match_date, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=VALUES(f_total_match_fund),  f_total_match_time=VALUES(f_total_match_time), f_total_match_day=VALUES(f_total_match_day), f_match_date=VALUES(f_match_date) ,f_modify_time=VALUES(f_modify_time)"
	args := make([]interface{}, 0)
	args = append(args, oid, sumFund, sumTime, sumDay, todayDate, nowTime, nowTime)
	res, err := cdbClient.ExecSQL(sqlStr, args)
	if err != nil {
		//clearTransaction(tx.(*sql.Tx))
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	logger.Debug("%v - user: %v affect result count: %v", util.GetCallee(), oid, res)
	return nil
}
*/

/*
// GetUserSummaryMatchStatusDB return user total match status
func GetUserSummaryMatchStatusDB(oid string) (*metadata.MatchStatus, error) {
	atomic.AddInt64(&SpeedGetDBStatus, 1)
	// cdb get info
	sql_str := "select f_total_match_fund, f_total_match_time, f_total_match_day, f_match_date from t_user_match_summary where f_user_id = ?"
	args := make([]interface{}, 0)
	args = append(args, oid)
	res, err := cdbClient.Query(sql_str, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("QueryDB error, err = %v, err", err))
	}
	// no match record just return empty record
	if len(res) == 0 {
		logger.Info("user: %v, get no match summary", oid)
		return &metadata.MatchStatus{}, errors.NewDBNilError(errors.WithMsg("user: %v, get no match summary", oid))
	}
	// format result
	var fund int64
	var matchTimes, day int
	var date string
	for _, value := range res {
		fund, err = strconv.ParseInt(value["f_total_match_fund"], 10, 64)
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("strconv.ParseInt error, error = %v, val = %s", err, value["f_total_match_time"]))
		}

		matchTimes, err = strconv.Atoi(value["f_total_match_time"])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("strconv.ParseInt error, error = %v, val = %s", err, value["f_total_match_time"]))
		}

		day, err = strconv.Atoi(value["f_total_match_day"])
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("strconv.Atoi error, error = %v, val = %s", err, value["f_total_match_time"]))
		}

		mDate, ok := value["f_match_date"]
		if !ok {
			return nil, errors.NewDBClientError(errors.WithMsg("strconv.Atoi error, error = %v, val = %s", err, value["f_match_date"]))
		}
		matchDate, err := time.ParseInLocation("2006-01-02 15:04:05", mDate, util.Loc)
		if err != nil {
			return nil, errors.NewDBClientError(errors.WithMsg("ParseInLocation error = %v, val = %s", err, mDate))
		}
		date = matchDate.In(util.Loc).Format("2006-01-02")
	}
	result := &metadata.MatchStatus{
		Funds: fund,
		Times: int32(matchTimes),
		Days:  int32(day),
		Date:  date,
	}
	logger.Debug("%v - user:%v get result: %v", util.GetCallee(), oid, result)
	return result, nil
}
*/

/*
// UpdateUserAllMatchStatusDBTX set user match info into db
func UpdateUserAllMatchStatusDBTX(oid, activity string, combo, op int, item, org, date string, t_fund, t_step, t_time int64, todayDate string, sum_fund, sum_day, sum_time int) error {
	atomic.AddInt64(&SpeedSetDBStatus, 1)
	// insert info to db
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	tx, err := cdbClient.TxBegin()
	if err != nil {
		if tx != nil {
			clearTransaction(tx.(*sql.Tx))
		}
		return errors.NewDBClientError(errors.WithMsg("%v - TxInsert user record error, err = %v", util.GetCallee(), err))
	}
	// update user activity state
	sqlStr := "INSERT INTO t_user_match_status (f_user_id, f_activity_id, f_total_match_fund, f_total_match_step, f_total_match_time, f_match_combo, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=VALUES(f_total_match_fund), f_total_match_step=VALUES(f_total_match_step), f_total_match_time=VALUES(f_total_match_time), f_match_combo=VALUES(f_match_combo) ,f_modify_time=VALUES(f_modify_time)"
	args := make([]interface{}, 0)
	args = append(args, oid, activity, t_fund, t_step, t_time, combo, nowTime, nowTime)
	resAc, err := cdbClient.TxExecSQL(tx, sqlStr, args)
	if err != nil {
		if tx != nil {
			clearTransaction(tx.(*sql.Tx))
		}
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	// update user summary state
	sqlStr = "INSERT INTO t_user_match_summary (f_user_id,  f_total_match_fund, f_total_match_time, f_total_match_day, f_match_date, f_create_time, f_modify_time)" +
		" VALUES(?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_total_match_fund=VALUES(f_total_match_fund),  f_total_match_time=VALUES(f_total_match_time), f_total_match_day=VALUES(f_total_match_day), f_match_date=VALUES(f_match_date) ,f_modify_time=VALUES(f_modify_time)"
	args = make([]interface{}, 0)
	args = append(args, oid, sum_fund, sum_time, sum_day, todayDate, nowTime, nowTime)
	resSum, err := cdbClient.TxExecSQL(tx, sqlStr, args)
	if err != nil {
		if tx != nil {
			clearTransaction(tx.(*sql.Tx))
		}
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	err = cdbClient.TxCommit(tx)
	if err != nil {
		logger.Error("%v - commit error: %v", util.GetCallee(), err)
		if tx != nil {
			clearTransaction(tx.(*sql.Tx))
		}
		return errors.NewDBClientError(errors.WithMsg("%v - commit error, err = %v, err", util.GetCallee(), err))
	}
	logger.Debug("%v - user: %v affect ac result count: %v, summary result count: %v", util.GetCallee(), oid, resAc, resSum)
	return nil
}
*/
