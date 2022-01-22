package activity

import (
	"fmt"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/common"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
	"time"
)

const (
	DEFAULT_SNAPSHOT_MIN = 30 // 默认快照间隔为30分钟
)

// 排名数据
type RankInfo struct {
	Steps int64 // 用户的活动步数
	Rank  int   // 排名
}

// 活动用户排名更新到db
func dumpActivityUserRankDB(info *ActivityInfo, users map[string]*RankInfo) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	status := 0

	for oid, rank := range users {
		sql := "INSERT INTO t_activity_member_history (f_activity_id, f_user_id, f_user_steps, " +
			" f_activity_rank, f_status, f_start_time, f_end_time, f_create_time, f_modify_time) " +
			" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?) " +
			" ON DUPLICATE KEY UPDATE f_modify_time=?, f_user_steps=?, f_activity_rank=?, f_start_time=?, f_end_time=? "

		args := make([]interface{}, 0)
		args = append(args, info.Aid, oid, rank.Steps, rank.Rank, status, info.StartTime, info.EndTime, now, now)
		args = append(args, now, rank.Steps, rank.Rank, info.StartTime, info.EndTime)

		if _, err := cdbClient.ExecSQL(sql, args); err != nil {
			logger.Sys("aid = %s, sql = %s, update DB record err = %v", info.Aid, sql, err)
			return err
		}
	}

	logger.Sys("aid = %s, activity member = %d, update db record successfully", info.Aid, len(users))
	return nil
}

// 活动小队排名更新到db
func dumpActivityTeamRankDB(info *ActivityInfo, teams map[string]*RankInfo) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	status := 0

	for teamId, rank := range teams {
		sql := "INSERT INTO t_team_history (f_activity_id, f_team_id, f_team_steps, " +
			" f_team_rank, f_status, f_start_time, f_end_time, f_create_time, f_modify_time) " +
			" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?) " +
			" ON DUPLICATE KEY UPDATE f_modify_time=?, f_team_steps=?, f_team_rank=?, f_start_time=?, f_end_time=? "
		args := make([]interface{}, 0)
		args = append(args, info.Aid, teamId, rank.Steps, rank.Rank, status, info.StartTime, info.EndTime, now, now)
		args = append(args, now, rank.Steps, rank.Rank, info.StartTime, info.EndTime)

		if _, err := cdbClient.ExecSQL(sql, args); err != nil {
			logger.Sys("aid = %s, sql = %s, update DB record err = %v", info.Aid, sql, err)
			return err
		}
	}

	logger.Sys("aid = %s, team number = %d, update db record successfully", info.Aid, len(teams))
	return nil
}

// 小队成员排名更新到db
func dumpTeamUserRankDB(info *ActivityInfo, teamId string, users map[string]*RankInfo) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	status := 0
	inTeam := 2

	for oid, rank := range users {
		sql := "INSERT INTO t_team_member_history (f_activity_id, f_team_id, f_user_id, f_user_steps, " +
			" f_user_rank, f_status, f_in_team, f_start_time, f_end_time, f_create_time, f_modify_time) " +
			" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
			" ON DUPLICATE KEY UPDATE f_modify_time=?, f_user_steps=?, f_user_rank=?, f_in_team=?, f_start_time=?, f_end_time=? "
		args := make([]interface{}, 0)
		args = append(args, info.Aid, teamId, oid, rank.Steps, rank.Rank, status, inTeam, info.StartTime, info.EndTime, now, now)
		args = append(args, now, rank.Steps, rank.Rank, inTeam, info.StartTime, info.EndTime)

		if _, err := cdbClient.ExecSQL(sql, args); err != nil {
			logger.Sys("aid = %s, sql = %s, update DB record err = %v", info.Aid, sql, err)
			return err
		}
	}

	logger.Sys("aid = %s, teamId = %s, team user = %d, update db record successfully", info.Aid, teamId, len(users))
	return nil
}

// 查询活动用户的排名
func fetchActivityUserRank(info *ActivityInfo) (map[string]*RankInfo, error) {
	ranks := make(map[string]*RankInfo)

	// 查询活动参与的用户排名
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, info.Aid)
	users, err := cacheclient.RedisClient.ZRevRangeWithScores(key, 0, -1).Result()
	if err != nil && err != redis.Nil {
		logger.Sys("key = %s query redis user rank err = %v", key, err)
		return ranks, err
	}

	for i, item := range users {
		if item.Member == nil {
			continue
		}
		oid := item.Member.(string)
		steps := int64(item.Score)
		rank := &RankInfo{
			Steps: steps,
			Rank:  i + 1,
		}
		ranks[oid] = rank
		logger.Sys("aid = %s, oid = %s, steps = %d, rank = %d, activity user rank fetch successfully", info.Aid, oid, rank.Steps, rank.Rank)
	}

	logger.Sys("aid = %s, activity member = %d fetch successfully", info.Aid, len(ranks))
	return ranks, nil
}

// 查询活动小队的排名
func fetchActivityTeamRank(info *ActivityInfo) (map[string]*RankInfo, error) {
	ranks := make(map[string]*RankInfo)

	// 查询活动的小队排名
	teamKey := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, info.Aid)
	teamRanks, err := cacheclient.RedisClient.ZRevRangeWithScores(teamKey, 0, -1).Result()
	if err != nil && err != redis.Nil {
		logger.Sys("key = %s, query team rank redis err = %v", teamKey, err)
		return ranks, err
	}

	for i, item := range teamRanks {
		if item.Member == nil {
			continue
		}
		teamId := item.Member.(string)
		steps := int64(item.Score)
		rank := &RankInfo{
			Steps: steps,
			Rank:  i + 1,
		}
		ranks[teamId] = rank
		logger.Sys("aid = %s, teamId = %s, steps = %d, rank = %d, activity team rank fetch successfully", info.Aid, teamId, rank.Steps, rank.Rank)
	}

	logger.Sys("aid = %s, activity team number = %d fetch successfully", info.Aid, len(ranks))
	return ranks, nil
}

// 查询小队成员的排名
func fetchTeamUserRank(info *ActivityInfo, teamId string) (map[string]*RankInfo, error) {
	ranks := make(map[string]*RankInfo)

	// 查询小队成员排名
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)
	teamUsers, err := cacheclient.RedisClient.ZRevRangeWithScores(key, 0, -1).Result()
	if err != nil && err != redis.Nil {
		logger.Sys("key = %s, query team user rank redis err = %v", key, err)
		return ranks, err
	}

	for i, item := range teamUsers {
		if item.Member == nil {
			continue
		}
		oid := item.Member.(string)
		steps := int64(item.Score)
		rank := &RankInfo{
			Steps: steps,
			Rank:  i + 1,
		}
		ranks[oid] = rank
		logger.Sys("aid = %s, teamId = %s, oid = %s, steps = %d, rank = %d, team user rank fetch successfully",
			info.Aid, teamId, oid, rank.Steps, rank.Rank)
	}

	logger.Sys("aid = %s, team user number = %d fetch successfully", info.Aid, len(ranks))
	return ranks, nil
}

// 活动结束则更新snapshot标识, 活动结束就不需要继续快照了
func updateSnapshotDb(info *ActivityInfo) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	// 活动未结束, 直接跳过
	if now < info.EndTime {
		logger.Sys("aid = %s, end_time = %s, is not finish no need to snapshot", info.Aid, info.EndTime)
		return nil
	}

	// snapshot=1表示活动结束, 快照完成了, 该活动就不需要再次快照了
	snapshot := 1
	sql := "UPDATE t_activity_info SET f_snapshot = ? WHERE f_activity_id = ?"
	args := make([]interface{}, 0)
	args = append(args, snapshot, info.Aid)
	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Sys("aid = %d, sql = %s update db err = %v", info.Aid, sql, err)
		return err
	}

	logger.Sys("aid = %s, update snapshot db record successfully", info.Aid)
	return nil
}

// 查询每个活动的用户排名, 更新落地
func snapActivityUserRank(info *ActivityInfo) error {
	actUsers, err := fetchActivityUserRank(info)
	if err != nil {
		return err
	}

	if err = dumpActivityUserRankDB(info, actUsers); err != nil {
		return err
	}

	return nil
}

// 查询每个活动的小队排名, 更新落地
func snapActivityTeamRank(info *ActivityInfo) (map[string]*RankInfo, error) {
	teams, err := fetchActivityTeamRank(info)
	if err != nil {
		return teams, err
	}

	if err = dumpActivityTeamRankDB(info, teams); err != nil {
		return teams, err
	}

	return teams, nil
}

// 查询每个小队的成员排名, 更新落地
func snapTeamUserRank(info *ActivityInfo, ranks map[string]*RankInfo) error {
	for teamId := range ranks {
		teamUsers, err := fetchTeamUserRank(info, teamId)
		if err != nil {
			continue
		}
		if err = dumpTeamUserRankDB(info, teamId, teamUsers); err != nil {
			continue
		}
	}
	return nil
}

// 查询所有进行中的活动和刚结束不久的活动（刚结束1天左右的活动）
func querySnapshotActivity() (map[string]*ActivityInfo, error) {
	acts := make(map[string]*ActivityInfo)
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	yesterday := time.Now().In(util.Loc).AddDate(0, 0, -1).Format("2006-01-02 15:04:05")

	sql := "select f_activity_id, f_start_time, f_end_time from t_activity_info where f_start_time <= ? and f_end_time >= ? and f_snapshot = 0"
	args := make([]interface{}, 0)
	args = append(args, now, yesterday)

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Sys("sql = %s, QueryDB error, err = %v", sql, err)
		return acts, errors.NewDBClientError(errors.WithMsg("QueryDB error, err = %v", err))
	}

	for _, item := range res {
		info := &ActivityInfo{}
		info.Aid = item["f_activity_id"]
		info.StartTime = item["f_start_time"]
		info.EndTime = item["f_end_time"]
		info.StartDate = util.Datetime2Date(item["f_start_time"])
		info.EndDate = util.Datetime2Date(item["f_end_time"])
		acts[info.Aid] = info
		logger.Sys("aid = %s, start_date = %s, end_date = %s prepare to deal", info.Aid, info.StartDate, info.EndDate)
	}

	return acts, nil
}

// 定时将排名数据落地
func Snapshot() {
	snapshotMin := config.GetConfig().ActivityConf.SnapshotMin
	if snapshotMin <= 0 {
		snapshotMin = DEFAULT_SNAPSHOT_MIN
	}

	for {
		// 查询所有进行中的活动
		acts, err := querySnapshotActivity()
		if err != nil {
			logger.Sys("query activity err = %v", err)
			alarm.CallAlarmFunc("Snapshot query activity error")
		}

		for _, info := range acts {
			// 查询每个活动的用户排名, 更新落地
			if err = snapActivityUserRank(info); err != nil {
				alarm.CallAlarmFunc("snapActivityUserRank error")
				continue
			}

			// 查询每个活动的小队排名, 更新落地
			ranks, err := snapActivityTeamRank(info)
			if err != nil {
				alarm.CallAlarmFunc("snapActivityTeamRank error")
				continue
			}

			// 查询每个小队的成员排名, 更新落地
			if err = snapTeamUserRank(info, ranks); err != nil {
				alarm.CallAlarmFunc("snapTeamUserRank error")
				continue
			}

			// 如果活动结束了, 更新snapshot标志, 下次就不用再落地了
			if err = updateSnapshotDb(info); err != nil {
				alarm.CallAlarmFunc("updateSnapshotDb error")
				continue
			}

			// 避免更新db过于频繁, 每更新一个活动等待延时一小会
			time.Sleep(30 * time.Millisecond)
		}

		// 延时10分钟, 等待下一次更新db
		time.Sleep(time.Duration(snapshotMin) * time.Minute)
	}
}
