//
//

package cache_access

import (
	"fmt"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"github.com/go-redis/redis"
)

func weekStartEnd(t time.Time) (time.Time, time.Time) {
	offset := int(time.Monday - t.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd
}

func GetBigMapRankKey(nt time.Time) string {
	week, _ := weekStartEnd(nt)
	return fmt.Sprintf("yqz_gb_%s_week_rank", week.Format("2006-01-02"))
}

func AddUserToBigMapRank(oid, sd string, totalStep int64) error {
	key := fmt.Sprintf("yqz_gb_%s_week_rank", sd)
	blackKey := fmt.Sprintf("yqz_gb_black_list_%s", sd)
	z := redis.Z{
		Score:  float64(totalStep),
		Member: oid,
	}
	pl := RedisClient.Pipeline()
	pl.ZAdd(key, z)
	pl.Expire(key, 14*24*time.Hour)
	pl.HDel(blackKey, oid)
	cmd, err := pl.Exec()
	log.Info("cmd = %+v", cmd)
	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %v, key = %s", err, cmd, key)
		return err
	}
	return nil
}

func RmUserFromBigMapRankAndAddBlackList(oid, sd string, nt time.Time, sumStep int64) error {
	rankKey := fmt.Sprintf("yqz_gb_%s_week_rank", sd)
	blackKey := fmt.Sprintf("yqz_gb_black_list_%s", sd)
	pl := RedisClient.Pipeline()
	pl.ZRem(rankKey, oid)
	pl.HSet(blackKey, oid, sumStep)
	_, end := weekStartEnd(nt)
	endDate := end.Format("2006-01-02") + " 23:59:59"
	tl, _ := time.LoadLocation("Asia/Shanghai")
	endTime, _ := time.ParseInLocation("2006-01-02 15:04:05", endDate, tl)
	pl.ExpireAt(blackKey, endTime)
	cmd, err := pl.Exec()
	log.Info("cmd = %+v", cmd)
	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %v", err, cmd)
		return err
	}
	return nil
}

func GetUserBigMapRankAndTotalNum(oid string, nt time.Time) (int64, int64, error) {
	key := GetBigMapRankKey(nt)
	pl := RedisClient.Pipeline()
	pl.ZRevRank(key, oid)
	pl.ZCard(key)
	cmd, err := pl.Exec()
	log.Info("cmd = %+v", cmd)
	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %v", err, cmd)
		return 0, 0, err
	}

	rankCmd := cmd[0].(*redis.IntCmd)
	totalCmd := cmd[1].(*redis.IntCmd)
	rank, _ := rankCmd.Result()
	total, _ := totalCmd.Result()

	return rank, total, nil
}

func IsInBigMap(oid, week string) (bool, error) {
	key := fmt.Sprintf("yqz_bg_user_hash_map_%s", week)
	exist, err := RedisClient.HExists(key, oid).Result()
	if err != nil {
		log.Error("redis hash exist error, err = %v, key = %s, oid = %s", err, key, oid)
		return false, err
	}
	return exist, nil
}

func SetUserInBigMap(oid, week string) error {
	key := fmt.Sprintf("yqz_bg_user_hash_map_%s", week)
	pl := RedisClient.Pipeline()
	pl.HSet(key, oid, 1)
	pl.Expire(key, 14*24*time.Hour)
	cmd, err := pl.Exec()
	if err != nil {
		log.Error("redis pipeline error, err = %v, cmd = %v, key = %s", err, cmd, key)
		return err
	}
	return nil
}

// GetMyWeekRank ...
func GetMyWeekRank(oid, weekID string) (int64, int64, error) {
	key := fmt.Sprintf("yqz_gb_%s_week_rank", weekID)
	pl := RedisClient.Pipeline()
	//ctx :=	context.Background()
	_ = pl.ZRevRank(key, oid)
	_ = pl.ZScore(key, oid)
	cmdList, err := pl.Exec()
	if err != nil {
		if err != redis.Nil {
			log.Error("redis pipeline error, err = %v, cmd = %+v", err, cmdList)
			return -1, -1, err
		} else {
			log.Info("oid: %v, weekID: %v has no record", oid, weekID)
			return 0, 0, nil
		}
	}

	rankCmd := cmdList[0].(*redis.IntCmd)
	scoreCmd := cmdList[1].(*redis.FloatCmd)
	rank, _ := rankCmd.Result()
	score, _ := scoreCmd.Result()
	log.Debug("cmd = %+v, %+v", rankCmd, scoreCmd)
	return rank, int64(score), nil
}
