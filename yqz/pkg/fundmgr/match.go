package fundmgr

import (
	"database/sql"
	"fmt"
	logger "git.code.oa.com/gongyi/gongyi_base/sys/log"
	meta "git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
	"strconv"
)

const MATCH_RECORD_PREFIX = "matchID_"

const MATCH_MIN_TIME = "1970-01-16 00:00:00"

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

type UserMatchRecord struct {
	FMatchID     string `json:"f_match_id"`
	FUserID      string `json:"f_user_id"`
	FMatchItemID string `json:"f_match_item_id"`
	FMatchOrgID  string `json:"f_match_org_id"`
	FMatchFund   int    `json:"f_match_fund"`
	FMatchStep   int    `json:"f_match_step"`
	FMatchCombo  int    `json:"f_match_combo"`
	FMatchDate   string `json:"f_match_date"`
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

var NullMatchRecord = &meta.MatchRecord{
	Date: MATCH_MIN_TIME,
}

// SetUserMatchInfoCache add combo times to cache
func SetUserMatchInfoCache(oid string, fund, step, combo, op int, item, lastMatchTime string, totalStep, totalFunds int64, totalTimes int) error {
	key := fmt.Sprintf("%s_%s", MATCH_USER_KEY_PREFIX, oid)
	pipe := cacheclient.RedisClient.TxPipeline()
	/*
		timeNow := time.Now()
		timeNowFmt := timeNow.In(util.Loc).Format("2006-01-02")
		// if specified match time, mean we assign combo value
		if len(lastMatchTime) == 0 {
			incr := pipe.HIncrBy(key, USER_COMBO, int64(combo))
			value[USER_DATE] = timeNowFmt
			combo = int(incr.Val())
		} else {
			value[USER_DATE] = lastMatchTime
			value[USER_COMBO] = combo
		}
	*/
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
	pipe.HMSet(key, value)
	/*
		// set expire time 7 days
		afterTime, _ := time.ParseDuration("168h")
		time7Day := timeNow.Add(afterTime)
		remainSecond := time.Duration(time7Day.Unix()-timeNow.Unix()) * time.Second
		pipe.Expire(key, time.Second*remainSecond)
	*/
	_, err := pipe.Exec()
	if err != nil {
		return errors.NewRedisClientError(errors.WithMsg("%v - err: %v", util.GetCallee(), err))
	}
	logger.Debug("%v - key: %v, oid: %v, combo: %v, fund: %v, step: %v, t_fund: %v, t_step: %v, t_time: %v",
		util.GetCallee(), key, oid, combo, fund, step, totalFunds, totalStep, totalTimes)
	return nil
}



// GetUserMatchInfoCache get user match info from cache
func GetUserMatchInfoCache(oid string) (*meta.MatchRecord, error) {
	key := fmt.Sprintf("%s_%s", MATCH_USER_KEY_PREFIX, oid)
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
	result := &meta.MatchRecord{
		Combo: int32(combo),
		Steps: int32(step),
		Funds: int32(fund),
		Date:  val[USER_DATE],
		Item:  val[USER_ITEM],
		Op:    meta.MatchRecord_OPType(op),
	}
	logger.Debug("%v - get key: %v result: %v", util.GetCallee(), key, result)
	return result, nil
}

func clearTransaction(tx *sql.Tx) {
	//err := tx.Rollback()
	err := cdbClient.TxRollback(tx)
	if err != sql.ErrTxDone && err != nil {
		logger.Error("%v", err)
	}
}



// GetUserMatchLastRecordDB get user match info from db
func GetUserMatchLastRecordDB(oid string) (*meta.MatchRecord, error) {
	result, err := GetUserMatchRecordByOffsetDB(oid, 0, 1)
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

func GetUserMatchRecordByOffsetDB(oid string, offset, size int) ([]*meta.MatchRecord, error) {
	/*
		loc, _ := time.LoadLocation("Asia/Shanghai")
		timeStamp, _ := time.ParseInLocation("2006-01-02", week, loc)
		weekStart, weekEnd := util.WeekStartEnd(timeStamp)
	*/
	// cdb get info
	sql_str := "select f_match_id, f_match_fund, f_match_step, f_match_combo, f_match_item_id, f_match_org_id, f_op_type, f_match_date from t_user_match_record where f_user_id = ? order by f_match_date desc limit ?, ?"
	args := make([]interface{}, 0)
	args = append(args, oid, offset, size)
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
	var result []*meta.MatchRecord
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

		result = append(result, &meta.MatchRecord{
			Combo:        int32(combo),
			Funds:        int32(fund),
			Steps:        int32(step),
			Date:         value[USER_DATE],
			Item:         value[USER_ITEM],
			Organization: value[USER_ORG],
			Id:           value[USER_MATCH_ID],
			Op:           meta.MatchRecord_OPType(op),
		})
	}
	logger.Debug("%v - user: %v, get %v result: %v", util.GetCallee(), oid, len(result), result)
	return result, nil
}

