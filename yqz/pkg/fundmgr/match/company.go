//
package match

import (
	"fmt"
	"strconv"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/gyredis"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
)

const (
	COMPANY_RANK_KEY_PREFIX = "yqz:fundmgr:company"
	RANK_LEADER             = "yqz:fundmgr:company:leader"
	EARLIEST_DATE           = "2021-06-01 00:00:00"
	TOTAL_CNT_FIELD         = "total_cnt"
	TOTAL_FUND_FIELD        = "total_fund"
)

var (
	cacheCli           *gyredis.RedisClient
	dbCli              dbclient.DBClient
	rankStopChan       chan struct{}
	updateRankInterval int = 3600
)

// RunUpdateRankWorker start update rank worker...
func RunUpdateRankWorker(interval int, cache *gyredis.RedisClient, db dbclient.DBClient) {
	cacheCli = cache
	dbCli = db
	updateRankInterval = interval
	rankStopChan = make(chan struct{})
	go func() {
		logger.Sys("UpdateRankWorker is start")
		if err := generateSnapshot(); err != nil {
			logger.Error("generateSnapshot error: %v", err)
		}
		// trigger in next hour
		now := util.GetLocalTime()
		timestamp := 3600 - int64(now.Second()) - int64((60 * now.Minute()))
		t := time.NewTimer(time.Duration(timestamp) * time.Second)
		for {
			select {
			case <-rankStopChan:
				logger.Sys("UpdateRankWorker is stop")
				t.Stop()
				return
			case <-t.C:
				t.Reset(time.Duration(interval) * time.Second)
				leader, err := checkLeader(RANK_LEADER, interval)
				if err != nil {
					logger.Error("checkLeader error: %v", err)
				}
				if leader {
					if err := generateSnapshot(); err != nil {
						logger.Error("generateSnapshot error: %v", err)
					}
				}
			}
		}
	}()
}

// CloseUpdateRankWorker stop update rank worker...
func CloseUpdateRankWorker() {
	close(rankStopChan)
}

// GetCompanyRanklistCache get rank list info
func GetCompanyRankListCache(offset, size int32) (int32, []*metadata.CompanyRank, error) {
	var result []*metadata.CompanyRank
	logger.Debug("request offset: %v size: %v", offset, size)
	rankKey := generateCompanyRankKey()
	// get total company count
	count, err := cacheCli.ZCard(rankKey).Result()
	if err != nil {
		if err == redis.Nil {
			// recover company rank
			ret, err := pushRecoverElement(rankKey)
			if err != nil || ret == 0 {
				logger.Error("pushRecoverElement error: %v", err)
			}
			return 0, nil, errors.NewOpsNeedRetryError(errors.WithMsg("rankKey:%v recover now", rankKey))
		} else {
			logger.Error("redis ZCard key: %v error: %v", rankKey, err)
			return -1, nil, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
		}
	}
	// company id list with rev order
	list, err := cacheCli.ZRevRangeWithScores(rankKey, int64(offset), int64(offset+size-1)).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return -1, nil, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	for index, item := range list {
		if item.Score == 0 {
			count -= 1
			continue
		}
		user := &metadata.CompanyRank{
			Id:   item.Member.(string),
			Fund: int64(item.Score),
			Rank: int32(index) + offset + 1,
		}
		result = append(result, user)
	}
	logger.Debug("get company rank: %v", result)
	return int32(count), result, nil
}

// GetCompanyRankCache get rank list info
func GetCompanyRankCache(companyID string) (*metadata.CompanyRank, error) {
	rankKey := generateCompanyRankKey()
	// company id list with rev order
	score, err := cacheCli.ZScore(rankKey, companyID).Result()
	if err != nil {
		if err == redis.Nil {
			return &metadata.CompanyRank{
				Id:   companyID,
				Fund: int64(0),
			}, errors.NewRedisNilError()
		}
		logger.Error("redis ops error: %v", err)
		return nil, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	company := &metadata.CompanyRank{
		Id:   companyID,
		Fund: int64(score),
	}
	logger.Debug("get company info: %+v", company)
	return company, nil
}

// GetMatchStat get total match stat ...
func GetMatchStat() (int64, int64, error) {
	key := generateMatchStatKey()
	// company id list with rev order
	ret, err := cacheCli.HMGet(key, TOTAL_CNT_FIELD, TOTAL_FUND_FIELD).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		if err == redis.Nil {
			return 0, 0, errors.NewRedisNilError()
		}
		return 0, 0, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	cnt, err := strconv.ParseInt(ret[0].(string), 10, 64)
	if err != nil {
		log.Error("can not parse value: %v, error: %v", ret[0], err)
		return 0, 0, err
	}
	fund, err := strconv.ParseInt(ret[1].(string), 10, 64)
	if err != nil {
		log.Error("can not parse value: %v, error: %v", ret[1], err)
		return 0, 0, err
	}
	logger.Debug("GetMatchStat totalCnt: %v, totalFund: %v", cnt, fund)
	return cnt, fund, nil
}

// SetCompanyRankCache set company rank info
func SetCompanyRankCache(companyID string, totalMatchFund int64) error {
	rankKey := generateCompanyRankKey()
	z := redis.Z{
		Score:  float64(totalMatchFund),
		Member: companyID,
	}
	_, err := cacheCli.ZAdd(rankKey, z).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	logger.Debug("redis ZAdd key: %v member: %+v", rankKey, z)
	return nil
}

// SetMatchStatCache set company all match info
func SetMatchStatCache(totalMatchCnt, totalMatchFund int64) error {
	statKey := generateMatchStatKey()
	var value = make(map[string]interface{})
	value[TOTAL_FUND_FIELD] = totalMatchFund
	value[TOTAL_CNT_FIELD] = totalMatchCnt
	_, err := cacheCli.HMSet(statKey, value).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	logger.Debug("redis ZAdd key: %v member: %v", statKey, value)
	return nil
}

// IncrCompanyRankCache incr company rank cache
func IncrCompanyRankCache(companyID string, incrMatchFund int) error {
	rankKey := generateCompanyRankKey()
	_, err := cacheCli.ZIncrBy(rankKey, float64(incrMatchFund), companyID).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	logger.Debug("redis ZIncr key: %v, member: %v, fund: %v", rankKey, companyID, incrMatchFund)
	return nil
}

// UpdateCompanyMatchDB ...
func UpdateCompanyMatchDB(companyID string, totalMatchFund int64, totalCnt int64, date string) error {
	sqlStr := "INSERT INTO t_company_match_stat (f_company_id, f_match_fund, f_match_cnt, f_stat_date, f_create_time, f_modify_time) VALUES (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE f_match_fund=VALUES(f_match_fund), f_match_cnt=VALUES(f_match_cnt), f_modify_time=VALUES(f_modify_time)"
	nowTime := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	args := make([]interface{}, 0)
	args = append(args, companyID, totalMatchFund, totalCnt, date, nowTime, nowTime)
	res, err := dbCli.ExecSQL(sqlStr, args)
	if err != nil {
		return errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	logger.Debug("%v - update company: %v, fund: %v, date: %v, affect count: %v", util.GetCallee(), companyID, totalMatchFund, date, res)
	return nil
}

// GetPeriodCompanyMatchStatFromDB sum match period match stat...
func GetPeriodCompanyMatchStatFromDB(startDate, endDate string) ([]*metadata.CompanyRank, error) {
	var result = make([]*metadata.CompanyRank, 0)
	sqlStr := "select f_match_org_id,sum(f_match_fund) as sum, count(*) as cnt from t_user_match_record where f_create_time >= ? and f_create_time < ? group by f_match_org_id;"
	args := make([]interface{}, 0)
	args = append(args, startDate, endDate)
	res, err := dbCli.Query(sqlStr, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	if len(res) == 0 {
		logger.Info("get no match record")
		return result, nil
	}
	for _, value := range res {
		sum, err := strconv.ParseInt(value["sum"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["sum"], err)
			continue
		}
		cnt, err := strconv.ParseInt(value["cnt"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["cnt"], err)
			continue
		}
		result = append(result, &metadata.CompanyRank{
			Id:    value["f_match_org_id"],
			Fund:  sum,
			Times: cnt,
		})
	}
	return result, nil
}

// RecoverCompanyRank use for recover rank from db
func RecoverCompanyRank() error {
	leader, err := checkLeader(RANK_LEADER, updateRankInterval)
	if err != nil {
		logger.Error("checkLeader error: %v", err)
	}
	if !leader {
		return nil
	}
	// step1 get all company rank for check last day we record
	earliestDate, err := fetchCompanyLastDateFromDB()
	if err != nil {
		logger.Error("fetchCompanyLastDateFromDB error: %v", err)
		return err
	}
	nowDate := util.GetTodayBeginTime().Format("2006-01-02 15:04:05")
	res, err := sumCompanyStatByRecordFromDB(earliestDate, nowDate)
	if err != nil {
		logger.Error("sumCompanyStatByRecordFromDB error: %v", err)
		return err
	}
	// step2 recover stat to db
	for _, value := range res {
		if len(value.Id) == 0 {
			continue
		}
		err = UpdateCompanyMatchDB(value.Id, value.Fund, value.Times, value.Date)
		if err != nil {
			logger.Error("UpdateCompanyMatchDB company: %v, fund: %v, date: %v, error: %v", value.Id, value.Fund, value.Date, err)
			return err
		}
	}
	// step3 recover to cache
	dbRes, err := sumCompanyStatFromDB(nowDate)
	if err != nil {
		logger.Error("sumCompanyStatFromDB error: %v", err)
		return err
	}
	var totalCnt, totalFund int64
	for _, item := range dbRes {
		err = SetCompanyRankCache(item.Id, item.Fund)
		if err != nil {
			logger.Error("SetCompanyRankCache company: %v, fund: %v, error: %v", item.Id, item.Fund, err)
			return err
		}
		logger.Info("SetCompanyRankCache company: %v, fund: %v success", item.Id, item.Fund)
		totalCnt += item.Times
		totalFund += item.Fund
	}
	// set total stat
	err = SetMatchStatCache(totalCnt, totalFund)
	if err != nil {
		logger.Error("SetMatchStatCache error: %v", err)
		return err
	}
	// only all step finish we set snapshot date
	err = setCompanySnapshot(nowDate)
	if err != nil {
		log.Error("setCompanySnapshot error: %v", err)
		return err
	}
	return nil
}

// RemoveCompanyRank remove company rank info in redis
func RemoveCompanyRank(companyID string) error {
	key := generateCompanyRankKey()
	count, err := cacheCli.ZRem(key, companyID).Result()
	if err != nil {
		logger.Error("RemoveCompanyRank redis ZRem key: %v member: %v error: %v", key, companyID, err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	logger.Info("RemoveCompanyRank key: %v companyID: %v success, res: %v", key, companyID, count)
	return nil
}

func fetchCompanyLastDateFromDB() (string, error) {
	sqlStr := "select MAX(f_stat_date) as minDate from t_company_match_stat;"
	args := make([]interface{}, 0)
	res, err := dbCli.Query(sqlStr, args)
	if err != nil {
		return "", errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	// no match record just return EARLIEST_DATE`
	if len(res) == 0 {
		logger.Info("get no commany match stat")
		return EARLIEST_DATE, nil
	}
	var minDate = EARLIEST_DATE
	for _, value := range res {
		minDate = value["minDate"]
	}
	logger.Info("get company match stat min date: %v", minDate)
	return minDate, nil
}

func sumCompanyStatByRecordFromDB(earliestDate, endDate string) ([]*metadata.CompanyRank, error) {
	var result = make([]*metadata.CompanyRank, 0)
	var tempResult = make(map[string]*metadata.CompanyRank)
	sqlStr := "select f_match_org_id, f_date, sum(f_match_fund) as sum, count(*) as cnt from t_user_match_record where f_date >= ? and f_date < ? group by f_match_org_id,f_date ;"
	args := make([]interface{}, 0)
	args = append(args, earliestDate, endDate)
	res, err := dbCli.Query(sqlStr, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	if len(res) == 0 {
		logger.Info("get no match record")
		return result, nil
	}
	for _, value := range res {
		sum, err := strconv.ParseInt(value["sum"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["sum"], err)
			continue
		}
		cnt, err := strconv.ParseInt(value["cnt"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["cnt"], err)
			continue
		}
		realDate := value["f_date"][:10] + " 00:00:00"
		key := fmt.Sprintf("%s|%s", value["f_match_org_id"], realDate)
		item, ok := tempResult[key]
		if ok {
			item.Fund += sum
			item.Times += cnt
		} else {
			tempResult[key] = &metadata.CompanyRank{
				Id:    value["f_match_org_id"],
				Date:  realDate,
				Fund:  sum,
				Times: cnt,
			}
		}
		/*
			result = append(result, &metadata.CompanyRank{
				Id:    value["f_match_org_id"],
				Date:  value["f_date"],
				Fund:  sum,
				Times: cnt,
			})
		*/
	}
	// here we need some special logical ,because old t_user_match_record f_date is not real date
	for _, item := range tempResult {
		result = append(result, item)
	}

	return result, nil
}

func sumCompanyStatFromDB(endDate string) ([]*metadata.CompanyRank, error) {
	var result = make([]*metadata.CompanyRank, 0)
	sqlStr := "select f_company_id, sum(f_match_fund) as sum, sum(f_match_cnt) as cnt from t_company_match_stat where f_stat_date < ? group by f_company_id;"
	args := make([]interface{}, 0)
	args = append(args, endDate)
	res, err := dbCli.Query(sqlStr, args)
	if err != nil {
		return nil, errors.NewDBClientError(errors.WithMsg("%v - ExecSQL error, err = %v, err", util.GetCallee(), err))
	}
	if len(res) == 0 {
		logger.Info("get no company record")
		return result, nil
	}
	for _, value := range res {
		sum, err := strconv.ParseInt(value["sum"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["sum"], err)
			continue
		}
		cnt, err := strconv.ParseInt(value["cnt"], 10, 64)
		if err != nil {
			log.Error("can not parse value: %v, error: %v", value["cnt"], err)
			continue
		}
		result = append(result, &metadata.CompanyRank{
			Id:    value["f_company_id"],
			Fund:  sum,
			Times: cnt,
		})
	}
	return result, nil
}

func checkLeader(key string, interval int) (bool, error) {
	result, err := redisClient.SetNX(key, util.GetLocalIP(), time.Duration(interval*2)*time.Second).Result()
	if err != nil {
		logger.Error("RankWorker instance: %v set leader info error", util.GetLocalIP())
		return false, err
	}
	if result {
		logger.Info("instance: %v set leader info success", util.GetLocalIP())
		if _, err = redisClient.Expire(key, time.Duration(interval*2)*time.Second).Result(); err != nil {
			logger.Error("extend instance: %v rank leader error: %v", util.GetLocalIP(), err)
		}
		return true, nil
	}
	value, err := redisClient.Get(key).Result()
	if err != nil {
		logger.Error("instance: %v get leader info error", util.GetLocalIP())
		return false, err
	}
	// extend self leader
	if value == util.GetLocalIP() {
		if _, err = redisClient.Expire(key, time.Duration(interval*2)*time.Second).Result(); err != nil {
			logger.Error("extend instance: %v rank leader error: %v", util.GetLocalIP(), err)
		}
		return true, nil
	}
	return false, nil
}

func generateSnapshot() error {
	logger.Info("try generateSnapshot ...")
	nowDate := util.GetTodayBeginTime().Format("2006-01-02 15:04:05")
	lastDate, err := GetCompanySnapshot()
	if err != nil {
		logger.Error("can not get last snapshot time, error: %v", err)
		return err
	}
	if lastDate == nowDate {
		logger.Info("already make snapshot date: %v", nowDate)
		return nil
	}
	// get record
	res, err := sumCompanyStatByRecordFromDB(lastDate, nowDate)
	if err != nil {
		logger.Error("sumCompanyStatByRecordFromDB error: %v", err)
		return err
	}
	if len(res) == 0 {
		// if no record just update date
		err = setCompanySnapshot(nowDate)
		if err != nil {
			log.Error("setCompanySnapshot error: %v", err)
			return err
		}
		return nil
	}
	// set data into db
	for _, value := range res {
		if len(value.Id) == 0 {
			continue
		}
		err = UpdateCompanyMatchDB(value.Id, value.Fund, value.Times, value.Date)
		if err != nil {
			logger.Error("UpdateCompanyMatchDB company: %v, fund: %v, date: %v, error: %v", value.Id, value.Fund, value.Date, err)
			return err
		}
	}
	// get sum data set to cache
	dbRes, err := sumCompanyStatFromDB(nowDate)
	if err != nil {
		logger.Error("sumCompanyStatFromDB error: %v", err)
		return err
	}
	var totalCnt, totalFund int64
	for _, item := range dbRes {
		if err = SetCompanyRankCache(item.Id, item.Fund); err != nil {
			logger.Error("SetCompanyRankCache company: %v, fund: %v, error: %v", item.Id, item.Fund, err)
			return err
		}
		logger.Info("SetCompanyRankCache company: %v, fund: %v success", item.Id, item.Fund)
		totalCnt += item.Times
		totalFund += item.Fund
	}
	err = SetMatchStatCache(totalCnt, totalFund)
	if err != nil {
		logger.Error("SetMatchStatCache error: %v", err)
		return err
	}
	// only all step finish we set snapshot date
	err = setCompanySnapshot(nowDate)
	if err != nil {
		log.Error("setCompanySnapshot error: %v", err)
		return err
	}
	return nil
}

func setCompanySnapshot(date string) error {
	_, err := cacheCli.Set(generateCompanySnapshotKey(), date, 3600*24*365*time.Second).Result()
	if err != nil {
		logger.Error("setCompanySnapshot date: %v, error: %v", date, err)
		return errors.NewRedisClientError(errors.WithMsg("%s", err))
	}
	return nil
}

func GetCompanySnapshot() (string, error) {
	value, err := cacheCli.Get(generateCompanySnapshotKey()).Result()
	if err != nil && err != redis.Nil {
		logger.Error("GetCompanySnapshot error: %v", err)
		return "", errors.NewRedisClientError(errors.WithMsg("%s", err))
	}
	if err == redis.Nil {
		value = EARLIEST_DATE
	}
	return value, nil
}

func generateCompanyRankKey() string {
	return fmt.Sprintf("%s:%s", COMPANY_RANK_KEY_PREFIX, "rank")
}

func generateMatchStatKey() string {
	return fmt.Sprintf("%s:%s", COMPANY_RANK_KEY_PREFIX, "stat")
}

func generateCompanySnapshotKey() string {
	return fmt.Sprintf("%s:%s", COMPANY_RANK_KEY_PREFIX, "snapshot")
}
