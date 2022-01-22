//
package match

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"github.com/go-redis/redis"
)

const MATCH_ACTIVITY_KEY_PREFIX = "yqz:fundmgr:match:activity"
const ACTIVITY_RANK_EXPIRE_TIME = 3600 * 24 * 31
const USER_LIMIT = 500

const DEFAULT_OID = "oproJj9U6jaTraolteGNtsQkacHA"

// for monitor
var SpeedRecoverMatchRankTime int32 = 0
var SpeedRecoverMatchRankRecord int32 = 0
var GetMatchRankCache int32 = 0
var SetMatchRankCache int32 = 0

func GetActivityMatchRank(activityID string, offset, size int32) (int32, []*metadata.UserRank, error) {
	atomic.AddInt32(&GetMatchRankCache, 1)
	var result []*metadata.UserRank
	logger.Info("request activity: %v  offset: %v size: %v", activityID, offset, size)
	activityKey := generateActivityKey(activityID)
	// get total count
	count, err := cacheclient.RedisClient.ZCard(activityKey).Result()
	if err != nil {
		if err == redis.Nil {
			// recover activity rank
			ret, err := pushRecoverElement(activityID)
			if err != nil || ret == 0 {
				logger.Error("pushRecoverElement error: %v", activityID)
			}
			return 0, nil, errors.NewOpsNeedRetryError(errors.WithMsg("activity:%v recover now", activityID))
		} else {
			logger.Error("redis ZCard key: %v error: %v", activityKey, err)
			return -1, nil, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
		}
	}
	if count == 0 {
		// recover activity rank
		ret, err := pushRecoverElement(activityID)
		if err != nil || ret == 0 {
			logger.Error("pushRecoverElement element: %v error: %v", activityID, err)
		}
		return 0, nil, errors.NewOpsNeedRetryError(errors.WithMsg("activity:%v recover now", activityID))
	}
	// comment id list with rev order
	list, err := cacheclient.RedisClient.ZRevRangeWithScores(activityKey, int64(offset), int64(offset+size-1)).Result()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return -1, nil, errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	for index, item := range list {
		if item.Score == 0 {
			count -= 1
			continue
		}
		user := &metadata.UserRank{
			Oid:  item.Member.(string),
			Fund: int32(item.Score),
			Rank: int32(index) + offset + 1,
		}
		result = append(result, user)
	}
	logger.Info("get rank: %v", result)
	return int32(count), result, nil
}

func UpdateActivityMatchRank(activityID, oid string, totalMatchFund int) error {
	atomic.AddInt32(&SetMatchRankCache, 1)
	pipe := cacheclient.RedisClient.Pipeline()
	activityKey := generateActivityKey(activityID)
	z := redis.Z{
		Score:  float64(totalMatchFund),
		Member: oid,
	}
	pipe.ZAdd(activityKey, z)
	logger.Debug("redis ZAdd key: %v value: %v", activityKey, z)
	// when user update rank we refresh rotten time
	pipe.Expire(activityKey, ACTIVITY_RANK_EXPIRE_TIME*time.Second)
	_, err := pipe.Exec()
	if err != nil {
		logger.Error("redis ops error: %v", err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	return nil
}

// RecoverActivityMatchRank use for new redis or recover old activity
func RecoverActivityMatchRank(activityID string) error {
	atomic.AddInt32(&SpeedRecoverMatchRankTime, 1)
	// step1 get all activity match record in user status
	ranks, err := fetchActivityMatchFromDB(activityID)
	if err != nil {
		logger.Error("fetchActivityMatchFromDB error: %v", err)
		return err
	}
	if len(ranks) == 0 {
		// we need add useless user in it avoid read db again
		err = UpdateActivityMatchRank(activityID, DEFAULT_OID, 0)
		if err != nil {
			logger.Error("add default item to activity: %v rank error: %v", activityID, err)
			return err
		}
	} else {
		// recover to redis
		for _, value := range ranks {
			err = UpdateActivityMatchRank(activityID, value.Oid, int(value.Fund))
			if err != nil {
				logger.Error("add user: %v to activity: %v rank error: %v", value.Oid, activityID, err)
				return err
			}
		}
	}
	logger.Info("recover activity: %v record count: %v success", activityID, len(ranks))
	return nil
}

// RecoverActivityMatchRank use for new redis or recover old activity
func RecoverActivityUserMatchRank(activityID, oid string) error {
	atomic.AddInt32(&SpeedRecoverMatchRankTime, 1)
	// step1 get all activity match record in user status
	rank, err := fetchActivityUserMatchFromDB(activityID, oid)
	if err != nil {
		logger.Error("fetchActivityUserMatchFromDB error: %v", err)
		return err
	}
	if rank == nil {
		logger.Info("user: %v has no activity: %v match record", oid, activityID)
		return nil
	} else {
		// recover to redis
		err = UpdateActivityMatchRank(activityID, rank.Oid, int(rank.Fund))
		if err != nil {
			logger.Error("add user: %v to activity: %v rank error: %v", rank.Oid, activityID, err)
			return err
		}
	}
	logger.Info("recover activity: %v oid: %v success", activityID, oid)
	return nil
}

// RemoveUserActivityMatchInfo remove user rank info in redis
func RemoveUserActivityMatchInfo(activityID, oid string) error {
	activityKey := generateActivityKey(activityID)
	count, err := cacheclient.RedisClient.ZRem(activityKey, oid).Result()
	if err != nil {
		logger.Error("RemoveUserActivityMatchInfo redis ZRem key: %v member: %v error: %v", activityKey, oid, err)
		return errors.NewRedisClientError(errors.WithMsg("redis ops err: %v", err))
	}
	logger.Info("RemoveUserActivityMatchInfo activity: %v oid: %v success, res: %v", activityID, oid, count)
	return nil
}

func fetchActivityMatchFromDB(activityID string) ([]*metadata.UserRank, error) {
	var result []*metadata.UserRank
	sqlStr := "select f_user_id, f_total_match_fund from t_user_match_status where f_activity_id = ? limit ?, ?"
	//sqlStr := "select f_user_id, f_total_match_fund from t_user_match_status where f_activity_id = ? order by f_total_match desc limit ?, ?"
	for i := 0; ; i++ {
		// collect batch user from db
		offset := i * USER_LIMIT
		args := []interface{}{activityID, offset, USER_LIMIT}
		res, err := cdbClient.Query(sqlStr, args)
		if err != nil {
			logger.Errorf("QueryDB error: %v", err)
			return nil, err
		}
		for index, item := range res {
			fund, _ := strconv.Atoi(item["f_total_match_fund"])
			user := &metadata.UserRank{
				Oid:  item["f_user_id"],
				Fund: int32(fund),
				Rank: int32(index),
			}
			result = append(result, user)
		}
		// if finish fetch all user from db
		if len(result) != USER_LIMIT {
			break
		}
	}
	// wait all thread finish util timeout or finish
	logger.Info("handle total user count: %v", len(result))
	atomic.AddInt32(&SpeedRecoverMatchRankRecord, int32(len(result)))
	return result, nil
}

func fetchActivityUserMatchFromDB(activityID, oid string) (*metadata.UserRank, error) {
	var result *metadata.UserRank = nil
	sqlStr := "select f_user_id, f_total_match_fund from t_user_match_status where f_activity_id = ? and f_user_id = ?"
	args := []interface{}{activityID, oid}
	res, err := cdbClient.Query(sqlStr, args)
	if err != nil {
		logger.Errorf("QueryDB error: %v", err)
		return nil, err
	}
	for index, item := range res {
		fund, _ := strconv.Atoi(item["f_total_match_fund"])
		result = &metadata.UserRank{
			Oid:  item["f_user_id"],
			Fund: int32(fund),
			Rank: int32(index),
		}
	}

	// wait all thread finish util timeout or finish
	atomic.AddInt32(&SpeedRecoverMatchRankRecord, 1)
	return result, nil
}

func generateActivityKey(activityID string) string {
	return fmt.Sprintf("%s:%s", MATCH_ACTIVITY_KEY_PREFIX, activityID)
}
