package team

import (
	"fmt"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/activity"
	"strconv"
	"strings"
	"time"

	activitypb "git.code.oa.com/gongyi/yqz/api/activitymgr"
	meta "git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/common"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
)

const (
	PAGE_CNT_MAX = 30 // 每页最多查询30条记录
)

var (
	cdbClient dbclient.DBClient
)

type TeamOper struct {
	t *meta.Team
}

func InitComponent(db dbclient.DBClient) error {
	cdbClient = db
	return nil
}

// 获取活动小队中人数最少的小队
func QueryMinMemberTeam(aid string) (*meta.Team, error) {
	///
	teams := make([]*meta.Team, 0)
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_META_KEY_PREFIX, aid)

	results, err := cacheclient.RedisClient.HGetAll(key).Result()
	if err != nil && err != redis.Nil {
		logger.Error("key = %s, query redis err = %v", key, err)
		return nil, err
	}
	if err == redis.Nil {
		logger.Error("key = %s, redis is empty error", key)
		return nil, errors.NewActivityMgrInternalError(errors.WithMsg("query teams is empty error"))
	}

	for _, item := range results {
		if len(item) <= 0 {
			logger.Error("aid = %s, item is empty", aid)
			continue
		}

		team := &meta.Team{}
		if err = util.Json2Pb(item, team); err != nil {
			logger.Error("aid = %s, item = %s, json to pb err = %v", aid, item, err)
			continue
		}
		teams = append(teams, team)
	}
	if 0 == len(teams) {
		logger.Error("aid = %s, redis results is empty error", aid)
		return nil, errors.NewActivityMgrInternalError(errors.WithMsg("query teams is empty error"))
	}
	logger.Debug("aid = %s, count=%d of teams:%+v", aid, len(teams), teams)

	// 查询小队成员人数
	var minMemberNums int64 = 0
	var minMemberTeam *meta.Team
	for i, v := range teams {
		teamid := v.TeamId
		//
		key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamid)
		mNums, err := cacheclient.RedisClient.ZCard(key).Result()
		if err != nil && err != redis.Nil {
			logger.Error("aid = %s, key = %s query redis team stats err = %v", aid, key, err)
			return nil, err
		}

		if minMemberNums == 0 && i == 0 {
			minMemberNums = mNums
			minMemberTeam = teams[i]
		} else if mNums < minMemberNums {
			minMemberNums = mNums
			minMemberTeam = teams[i]
		}
		logger.Debug("aid = %s, teamid=%s, mNums=%d,  minMemberNums:%d, i=%d", aid, teamid, mNums, minMemberNums, i)
	}
	return minMemberTeam, nil
}


// 查询小队统计(小队成员数量)
func (o *TeamOper) queryTeamStats(aid, teamId string) (*activitypb.TeamStats, error) {
	stats := &activitypb.TeamStats{}

	// 查询小队成员人数
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)
	teamMembers, err := cacheclient.RedisClient.ZCard(key).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, team_id = %s, key = %s query redis team stats err = %v", aid, teamId, key, err)
		return stats, err
	}

	// 查询小队总步数
	teamKey := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)
	teamSteps, err := cacheclient.RedisClient.ZScore(teamKey, teamId).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, team_id = %s, key = %s query redis team steps err = %v", aid, teamId, key, err)
		return stats, err
	}

	// 查询小队排名
	teamRank, err := cacheclient.RedisClient.ZRevRank(teamKey, teamId).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, team_id = %s, key = %s query redis team rank err = %v", aid, teamId, key, err)
		return stats, err
	}

	stats.TeamMember = teamMembers
	stats.TeamSteps = int64(teamSteps)
	stats.TeamRank = teamRank + 1
	return stats, nil
}

// 查询小队成员排名(根据lastKey翻页, 避免排名实时变化太快?) 第一页page=0
func (o *TeamOper) queryUserRank(teamId string, page int64, pgCnt int64) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)

	results, err := cacheclient.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if err != nil && err != redis.Nil {
		logger.Error("team_id = %s, key = %s, page = %d, pgCnt = %d redis query team user rank err = %v", teamId, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询小队成员逆向排名(根据lastKey翻页, 避免排名实时变化太快?) 第一页page=0
func (o *TeamOper) queryUserDescRank(teamId string, page int64, pgCnt int64) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)

	results, err := cacheclient.RedisClient.ZRangeWithScores(key, start, stop).Result()
	if err != nil && err != redis.Nil {
		logger.Error("team_id = %s, key = %s, page = %d, pgCnt = %d redis query team desc user rank err = %v", teamId, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询小队的用户排名
// 返回值: bool: true表示用户参与小队, int: 用户在小队的排名
func (o *TeamOper) queryTeamUserRank(teamId, oid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)

	rank, err := cacheclient.RedisClient.ZRevRank(key, oid).Result()
	// 用户没有加入小队, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("tid = %s, oid = %s, key = %s, query redis team user rank err = %v", teamId, oid, key, err)
		return false, 0, err
	}

	return true, rank + 1, nil
}

// 查询小队的用户步数
// 返回值: bool: true表示用户参与小队, int: 用户在小队的步数
func (o *TeamOper) queryTeamUserStep(teamId, oid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)

	step, err := cacheclient.RedisClient.ZScore(key, oid).Result()

	// 用户没有加入活动, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("tid = %s, oid = %s, key = %s, query redis team user rank err = %v", teamId, oid, key, err)
		return false, 0, err
	}

	return true, int64(step), nil
}

// 更新小队排名及总公益金数据
func (o *TeamOper) modifyTeamFunds(aid, teamId, oid string, join bool) error {
	/// 获取用户在该活动下的公益金存量
	funds, err := activity.GetActUserFunds(aid, oid)
	if err != nil {
		return err
	}

	//
	var modifyFunds int64
	if join {
		modifyFunds = funds
	} else {
		modifyFunds = 0 - funds
	}

	// 初略更新小队的总公益金, 小队的总公益金在小队成员配捐获取公益金时实时更新
	rankKey := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_FUND_RANK_KEY_PREFIX, aid)
	if _, err := cacheclient.RedisClient.ZIncrBy(rankKey, float64(modifyFunds), teamId).Result(); err != nil {
		logger.Error("%v -- aid = %s, team_id = %s, oid = %s, key = %s, funds = %d, redis add team funds err = %v",
			util.GetCallee(), aid, teamId, oid, rankKey, modifyFunds, err)
		return err
	}
	logger.Debug("%v -- aid = %s, team_id = %s, oid = %s, key = %s, modifyFunds = %d, join = %v, add team funds successfully",
		util.GetCallee(), aid, teamId, oid, rankKey, modifyFunds, join)

	/// 更新小队总公益金和小队公益金数据更新时间   /// 这里是更新小队数据，用于小队详情页
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	team_key := fmt.Sprintf("%s:%s:%s", common.ACTIVITY_TEAM_MATCH_FUND_KEY_PREFIX, aid, teamId)
	//
	pipe := cacheclient.RedisClient.Pipeline()
	pipe.HSet(team_key, common.UPDATE_TIME_FIELD, now)
	teamFunds, _ := pipe.HIncrBy(team_key, common.TEAM_FUNDS_FIELD, modifyFunds).Result()
	// exec
	_, err = pipe.Exec()
	if err != nil {
		logger.Error("%v -- team_key = %s, redis pipe err = %v, oid:%s", util.GetCallee(), team_key, err, oid)
		return errors.NewRedisClientError(errors.WithMsg("redis pipeline team_key:%s, uid:%s, err:%+v ", team_key, oid, err))
	}
	logger.Info("%v -  update openid:%s succ, key:%s, teamFunds:%d ", util.GetCallee(), oid, team_key, teamFunds)

	return nil
}

// 初略调整小队的总步数, 用户加入或者退出小队时触发
func (o *TeamOper) modifyTeamSteps(aid, teamId, oid string, steps int64, join bool) error {
	var modifySteps int64

	if join {
		modifySteps = steps
	} else {
		modifySteps = 0 - steps
	}

	// 初略更新小队的总步数, 小队的总步数具体在定时任务每隔1分钟更新1次
	statsKey := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)
	if _, err := cacheclient.RedisClient.ZIncrBy(statsKey, float64(modifySteps), teamId).Result(); err != nil {
		logger.Error("aid = %s, team_id = %s, oid = %s, key = %s, steps = %d, redis add team steps err = %v",
			aid, teamId, oid, statsKey, steps, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, oid = %s, key = %s, steps = %d, join = %v, add team steps successfully",
		aid, teamId, oid, statsKey, modifySteps, join)
	return nil
}

// 更新小队的成员排名（加入小队）
func (o *TeamOper) updateUserRank(aid, teamId, oid string, steps int64) error {
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)
	z := redis.Z{
		Score:  float64(steps),
		Member: oid,
	}

	if _, err := cacheclient.RedisClient.ZAdd(key, z).Result(); err != nil {
		logger.Error("aid = %s, team_id = %s, oid = %s, key = %s, add team user rank err = %v", aid, teamId, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, oid = %s, key = %s, steps = %d, add team user rank successfully", aid, teamId, oid, key, steps)
	return nil
}

// 删除小队的成员排名（退出小队）
func (o *TeamOper) deleteUserRank(aid, teamId, oid string) error {
	key := fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamId)

	if _, err := cacheclient.RedisClient.ZRem(key, oid).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, team_id = %s, oid = %s, key = %s, remove team user rank err = %v", aid, teamId, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, oid = %s, key = %s, remove team user rank successfully", aid, teamId, oid, key)
	return nil
}

// 小队基础配置存储到redis
func (o *TeamOper) syncTeamRedis(team *meta.Team) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_META_KEY_PREFIX, team.ActivityId)
	field := fmt.Sprintf("%s:%s", common.TEAM_FIELD_INFO_PREFIX, team.TeamId)

	content, err := util.Pb2Json(team)
	if err != nil {
		logger.Error("aid = %s, teamId = %s serial pb to json err = %v", team.ActivityId, team.TeamId, err)
		return err
	}

	if _, err = cacheclient.RedisClient.HSet(key, field, content).Result(); err != nil {
		logger.Error("key = %s, content = %s, redis hset err = %v", key, content, err)
		return err
	}

	logger.Debug("key = %s, content = %s, sync to redis successfully", key, content)
	return nil
}

// 小队基础配置存储到DB
func (o *TeamOper) syncTeamDB(team *meta.Team) error {
	teamFlag := int(team.TeamFlag)

	args := make([]interface{}, 0)
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "INSERT INTO t_team (f_activity_id, f_team_id, f_team_type, f_team_desc, f_team_creator, f_team_leader, f_team_flag, f_status, f_create_time, f_modify_time) " +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
		" ON DUPLICATE KEY UPDATE f_modify_time=?, f_team_type=?, f_team_desc=?, f_team_creator=?, f_team_leader=?, f_team_flag=?, f_status=? "
	args = append(args, team.ActivityId, team.TeamId, team.TeamType, team.TeamDesc, team.TeamCreator, team.TeamLeader, teamFlag, team.Status, now, now)
	args = append(args, now, team.TeamType, team.TeamDesc, team.TeamCreator, team.TeamLeader, teamFlag, team.Status)
	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("sql = %s, update DB record err = %v", sql, err)
		return err
	}

	logger.Debug("aid = %s, teamId = %s update db team record successfully", team.ActivityId, team.TeamId)
	return nil
}

// 小队基础配置更新到DB
func (o *TeamOper) updateTeamDB(team *meta.Team) error {
	teamFlag := int(team.TeamFlag)

	args := make([]interface{}, 0)
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "update t_team set f_modify_time=?, f_team_type=?, f_team_desc=?, f_team_creator=?, f_team_leader=?, f_team_flag=?, f_status=? " +
		" where f_activity_id = ? and f_team_id = ?"
	args = append(args, now, team.TeamType, team.TeamDesc, team.TeamCreator, team.TeamLeader, teamFlag, team.Status)
	args = append(args, team.ActivityId, team.TeamId)

	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("sql = %s, update DB record err = %v", sql, err)
		return err
	}

	logger.Debug("aid = %s, teamId = %s update db team record successfully", team.ActivityId, team.TeamId)
	return nil
}

// 从redis查询小队配置
func (o *TeamOper) queryTeamsRedis(aid string, teamIds []string) ([]*meta.Team, error) {
	teams := make([]*meta.Team, 0)
	fields := make([]string, 0)

	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_META_KEY_PREFIX, aid)
	for _, id := range teamIds {
		fields = append(fields, fmt.Sprintf("%s:%s", common.TEAM_FIELD_INFO_PREFIX, id))
	}

	results, err := cacheclient.RedisClient.HMGet(key, fields...).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, teamIds = %v query redis err = %v", aid, teamIds, err)
		return teams, err
	}
	if err == redis.Nil {
		logger.Error("aid = %s, teamIds = %v, redis is empty error", aid, teamIds)
		return teams, errors.NewActivityMgrInternalError(errors.WithMsg("query teams is empty error"))
	}

	// 解析小队配置
	// HMGet必须判断nil值, 否则取值可能会异常
	// 参考: https://github.com/go-redis/redis/blob/master/commands_test.go  (搜索'HMGet')
	for _, item := range results {
		if nil == item {
			continue
		}
		content := item.(string)
		if len(content) <= 0 {
			logger.Error("aid = %s, teamIds = %v content is empty", aid, teamIds)
			continue
		}

		team := &meta.Team{}
		if err = util.Json2Pb(content, team); err != nil {
			logger.Error("aid = %s, content = %s, json to pb err = %v", aid, content, err)
			continue
		}
		teams = append(teams, team)
	}
	if 0 == len(teams) {
		logger.Error("aid = %s, teamIds = %v redis results is empty error", aid, teamIds)
		return teams, errors.NewActivityMgrInternalError(errors.WithMsg("query teams is empty error"))
	}

	return teams, nil
}

// 从DB查询小队配置
func (o *TeamOper) queryTeamsDB(aid string, teamIds []string) ([]*meta.Team, error) {
	teams := make([]*meta.Team, 0)

	args := make([]interface{}, 0)
	sql := "select f_activity_id, f_team_id, f_team_type, f_team_desc, f_team_creator, f_team_leader, f_team_flag, f_status from t_team where f_activity_id = ? and f_team_id in ("
	args = append(args, aid)
	for _, id := range teamIds {
		sql += "?, "
		args = append(args, id)
	}
	sql = strings.TrimRight(sql, ", ")
	sql += ")"

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("aid = %s, sql = %s, QueryDB error, err = %v, err", aid, sql, err)
		return teams, errors.NewDBClientError(errors.WithMsg("aid = %s, QueryDB error, err = %v", aid, err))
	}
	if 0 == len(res) {
		logger.Error("aid = %s, sql = %s, DB record is not exist err", aid, sql)
		return teams, errors.NewInternalError(errors.WithMsg("aid = %s, sql = %s DB is not exist err", aid, sql))
	}

	for _, item := range res {
		team := &meta.Team{}
		teamFlag, _ := strconv.Atoi(item["f_team_flag"])
		teamType, _ := strconv.Atoi(item["f_team_type"])
		status, _ := strconv.Atoi(item["f_status"])
		team.ActivityId = item["f_activity_id"]
		team.TeamId = item["f_team_id"]
		team.TeamType = int32(teamType)
		team.TeamDesc = item["f_team_desc"]
		team.TeamCreator = item["f_team_creator"]
		team.TeamLeader = item["f_team_leader"]
		team.TeamFlag = meta.Team_TeamFlag(teamFlag)
		team.Status = int32(status)
		teams = append(teams, team)
	}

	return teams, nil
}

func (o *TeamOper) fetchTeams(aid string, teamIds []string) ([]*meta.Team, error) {
	if 0 == len(teamIds) {
		t := make([]*meta.Team, 0)
		logger.Error("aid = %s, team is empty error", aid)
		return t, errors.NewActivityMgrInternalError(errors.WithMsg("query teams is empty error"))
	}

	success := true

	// 先查询redis
	teams, err := o.queryTeamsRedis(aid, teamIds)
	if err != nil {
		success = false
	}

	if !success {
		// redis查询失败, 再尝试查询DB
		teams, err = o.queryTeamsDB(aid, teamIds)
		if err != nil {
			return teams, err
		}
		// 将活动信息回补到redis, 失败仅报错, 不返回错误
		for _, team := range teams {
			var _ = o.syncTeamRedis(team)
		}
	}

	return teams, nil
}

// 查询小队配置信息
func QueryTeams(aid string, teamIds []string) ([]*meta.Team, error) {
	oper := &TeamOper{}
	teams, err := oper.fetchTeams(aid, teamIds)
	if err != nil {
		return teams, err
	}
	// 查询小队队长的排名 和公益金信息
	for i, item := range teams {
		if len(item.TeamLeader) > 0 {
			_, rank, err := oper.queryTeamUserRank(item.TeamId, item.TeamLeader)
			if err != nil {
				return teams, err
			}
			teams[i].TeamLeaderRank = int32(rank)
		}
		teams[i].TeamFunds, _ = activity.GetActTeamFunds(aid, item.TeamId) //出错，公益金为0
	}

	return teams, nil
}

// 创建小队配置
func CreateTeam(team *meta.Team) (*meta.Team, error) {
	oper := &TeamOper{}
	t := &meta.Team{}

	// 1·生成唯一小队id
	teamId, err := util.GetTransId("team_")
	if err != nil {
		logger.Error("GetTransId error, err = %v, err", err)
		return t, err
	}

	// 2.初始化小队配置
	t.TeamId = teamId
	t.ActivityId = team.ActivityId
	t.TeamType = team.TeamType
	t.TeamDesc = team.TeamDesc
	t.TeamCreator = team.TeamCreator
	t.TeamLeader = team.TeamLeader
	t.TeamFlag = team.TeamFlag
	t.Status = 1 // 小队状态, 1 正常 2 解散(系统默认小队不能解散)

	// 3.同步小队配置到redis
	if err = oper.syncTeamRedis(t); err != nil {
		return t, err
	}

	// 4.同步小队配置到db
	if err = oper.syncTeamDB(t); err != nil {
		return t, err
	}

	content, _ := util.Pb2Json(t)
	logger.Debug("create team: %s successfully", content)
	return t, nil
}

// 查询小队统计(小队成员数量)
func QueryTeamStats(aid, teamId string) (*activitypb.TeamStats, error) {
	oper := &TeamOper{}
	return oper.queryTeamStats(aid, teamId)
}

// 更新用户在小队的排名
func UpdateUserRank(aid, teamId, oid string, steps int64) error {
	oper := &TeamOper{}
	return oper.updateUserRank(aid, teamId, oid, steps)
}

// 更新小队总公益金
func ModifyTeamFunds(aid, teamId, oid string, join bool) error {
	oper := &TeamOper{}
	return oper.modifyTeamFunds(aid, teamId, oid, join)
}

func ModifyTeamSteps(aid, teamId, oid string, steps int64, join bool) error {
	oper := &TeamOper{}
	return oper.modifyTeamSteps(aid, teamId, oid, steps, join)
}

// 删除用户在小队的排名
func DeleteUserRank(aid, teamId, oid string) error {
	oper := &TeamOper{}
	return oper.deleteUserRank(aid, teamId, oid)
}

// 查询小队用户排名
func QueryUserRank(teamId string, page, pgCnt int64) ([]redis.Z, error) {
	oper := &TeamOper{}
	return oper.queryUserRank(teamId, page, pgCnt)
}

// 查询小队用户排名
func QueryUserDescRank(teamId string, page, pgCnt int64) ([]redis.Z, error) {
	oper := &TeamOper{}
	return oper.queryUserDescRank(teamId, page, pgCnt)
}

// 查询小队的用户排名
// 返回值: bool: true表示用户参与小队, int: 用户在小队的排名
func QueryTeamUserRank(tid, oid string) (bool, int64, error) {
	oper := &TeamOper{}
	return oper.queryTeamUserRank(tid, oid)
}


func queryMemberListDB(aid, teamId string, offset, count int) ([]*meta.TeamMember, error) {
	list := make([]*meta.TeamMember, 0)

	sql := "select f_activity_id,f_team_id,f_user_id,f_in_team,f_status,f_create_time from t_team_member where f_activity_id = ? and f_team_id = ? and f_in_team = ? order by f_create_time limit ?,?"
	args := make([]interface{}, 0)
	args = append(args, aid, teamId, 2, offset, count)

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("%v - QueryDB error, err = %v, err", util.GetCallee(), err)
		return list, errors.NewDBClientError(errors.WithMsg("%v - QueryDB error, err = %v, err", util.GetCallee(), err))
	}

	for _, v := range res {
		inTeam, _ := strconv.Atoi(v["f_in_team"])
		status, _ := strconv.Atoi(v["f_status"])
		list = append(list, &meta.TeamMember{
			ActivityId: v["f_activity_id"],
			TeamId:     v["f_team_id"],
			UserId:     v["f_user_id"],
			InTeam:     int32(inTeam),
			Status:     int32(status),
		})
	}

	return list, nil
}

// 队长选举
func ElectTeamLeader(oid string, join bool, team *meta.Team) error {
	updateTeam := false // true表示要更新小队配置

	if join { // 用户加入
		// 队长已经存在, 直接返回
		if len(team.TeamLeader) > 0 {
			return nil
		}
		// 重新设置队长, 将用户设置为队长（第一位加入小队的成员作为队长）
		team.TeamLeader = oid
		updateTeam = true
	} else { // 用户退出
		// 退出的成员不是队长, 直接返回
		if oid != team.TeamLeader {
			return nil
		}
		// 重新设置队长, 按照加入时间顺序选取新的队长
		members, err := queryMemberListDB(team.ActivityId, team.TeamId, 0, 1)
		if err != nil {
			return err
		}
		if len(members) > 0 { // 还有人在小队
			team.TeamLeader = members[0].UserId
			updateTeam = true
		} else { // 没有人在小队了, 队长为空
			team.TeamLeader = ""
			updateTeam = true
		}
	}

	if updateTeam {
		logger.Debug("oid = %s, aid = %s, team_id = %s, team_leader = %s, join = %v, need to sync", oid, team.ActivityId, team.TeamId, team.TeamLeader, join)
		// 同步更新redis和DB配置
		oper := &TeamOper{}
		if err := oper.syncTeamRedis(team); err != nil {
			return err
		}
		if err := oper.syncTeamDB(team); err != nil {
			return err
		}
	}

	return nil
}

func DeleteTeam(activityID, teamID string) error {
	state, err := QueryTeamStats(activityID, teamID)
	if err != nil {
		logger.Error("QueryTeamStats error: %v", err)
		return err
	}
	if state.TeamMember != 0 {
		logger.Error("aid: %s team: %v, member_cnt: %v", activityID, teamID, state.TeamMember)
		return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid: %s team: %v, member_cnt: %v", activityID, teamID, state.TeamMember))
	}
	// set db status
	sql := "update t_team set f_status = 2 where f_activity_id = ? and f_team_id = ?"
	args := make([]interface{}, 0)
	args = append(args, activityID, teamID)
	_, err = cdbClient.ExecSQL(sql, args)
	if err != nil {
		logger.Error("ExecSQL: %v, error: %v", sql, err)
		return errors.NewDBClientError(errors.WithMsg("ExecSQL: %v, error: %v", sql, err))
	} else {
		logger.Debug("update db f_activity_id: %v, f_activity_id: %v status = 2", activityID, teamID)
	}
	// delete redis
	err = deleteTeamRelativeRedis(activityID, teamID)
	if err != nil {
		return err
	}
	return nil
}

func deleteTeamRelativeRedis(activityID, teamID string) error {
	// 删除小队基础信息
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_META_KEY_PREFIX, activityID)
	fields := make([]string, 0)
	fields = append(fields, fmt.Sprintf("%s:%s", common.TEAM_FIELD_INFO_PREFIX, teamID))
	results, err := cacheclient.RedisClient.HDel(key, fields...).Result()
	if err != nil {
		logger.Error("aid = %s, teamdIds = %v del redis err = %v", activityID, teamID, err)
		return errors.NewRedisClientError(errors.WithMsg("aid = %s, teamdIds = %v del redis err = %v", activityID, teamID, err))
	} else {
		logger.Debug("del redis key: %v, field = %v, count: %v", key, fields, results)
	}

	// 删除小队用户排名
	key = fmt.Sprintf("%s:%s", common.TEAM_USER_RANK_KEY_PREFIX, teamID)
	// can not use del command, just use expire
	delResult, err := cacheclient.RedisClient.Expire(key, 0*time.Second).Result()
	if err != nil {
		logger.Error("del redis key: %v error: %v", key, err)
		return errors.NewRedisClientError(errors.WithMsg("del redis key: %v error: %v", key, err))
	} else {
		logger.Debug("del redis key: %v, count: %v", key, delResult)
	}

	// 删除活动的小队步数排名
	key = fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, activityID)
	if _, err = cacheclient.RedisClient.ZRem(key, teamID).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, teamId = %s, key = %s, remove team rank err = %v", activityID, teamID, key, err)
		return err
	}
	logger.Debug("aid = %s, team_id = %s, key = %s, remove team rank successfully", activityID, teamID, key)

	// 删除活动的小队公益金排名
	key = fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_FUND_RANK_KEY_PREFIX, activityID)
	if _, err = cacheclient.RedisClient.ZRem(key, teamID).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, teamId = %s, key = %s, remove team fund rank err = %v", activityID, teamID, key, err)
		return err
	}
	logger.Debug("aid = %s, team_id = %s, key = %s, remove team rank successfully", activityID, teamID, key)

	return nil
}

// GetActivityTeamList ...
func GetActivityTeamList(offset, limit int32, aid string, flag int32) (int32, []*meta.Team, error) {
	var total int32
	teams := make([]*meta.Team, 0)
	if flag != -1 {
		sql := "select count(*) from t_team where f_activity_id = ? and f_team_flag = ?"
		args := make([]interface{}, 0)
		args = append(args, aid, flag)
		res, err := cdbClient.Query(sql, args)
		if err != nil {
			return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
		}
		if len(res) == 0 {
			logger.Info("DB t_team f_activity_id: %v, type: %v has no record", aid, flag)
			return 0, []*meta.Team{}, nil
		}
		count, _ := strconv.Atoi(res[0]["count(*)"])
		total = int32(count)
		if total == 0 {
			logger.Info("DB t_team f_activity_id: %v, type: %v has no record", aid, flag)
			return 0, []*meta.Team{}, nil
		}

		sql = "select f_activity_id, f_team_id, f_team_type, f_team_desc, f_team_creator, f_team_leader, f_status, f_team_flag from t_team where f_activity_id = ? and f_team_flag = ? order by f_create_time limit ?, ?"
		args = make([]interface{}, 0)
		args = append(args, aid, flag, offset, limit)
		res, err = cdbClient.Query(sql, args)
		if err != nil {
			logger.Error("QueryDB error: %v", err)
			return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", util.GetCallee(), err))
		}
		if len(res) == 0 {
			return -1, nil, errors.NewDBNilError(errors.WithMsg("DB t_team has no activity: %v team", aid))
		}
		for _, item := range res {
			team := &meta.Team{}
			teamType, _ := strconv.Atoi(item["f_team_type"])
			teamFlag, _ := strconv.Atoi(item["f_team_flag"])
			status, _ := strconv.Atoi(item["f_status"])
			team.ActivityId = item["f_activity_id"]
			team.TeamId = item["f_team_id"]
			team.TeamType = int32(teamType)
			team.TeamDesc = item["f_team_desc"]
			team.TeamCreator = item["f_team_creator"]
			team.TeamLeader = item["f_team_leader"]
			team.Status = int32(status)
			team.TeamFlag = meta.Team_TeamFlag(teamFlag)
			teams = append(teams, team)
		}
	} else {
		sql := "select count(*) from t_team where f_activity_id = ?"
		args := make([]interface{}, 0)
		args = append(args, aid)
		res, err := cdbClient.Query(sql, args)
		if err != nil {
			return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
		}
		if len(res) == 0 {
			logger.Info("DB t_team f_activity_id: %v has no record", aid)
			return 0, []*meta.Team{}, nil
		}
		count, _ := strconv.Atoi(res[0]["count(*)"])
		total = int32(count)
		if total == 0 {
			logger.Info("DB t_team f_activity_id: %v, type: %v has no record", aid, flag)
			return 0, []*meta.Team{}, nil
		}

		sql = "select f_activity_id, f_team_id, f_team_type, f_team_desc, f_team_creator, f_team_leader, f_status, f_team_flag from t_team where f_activity_id = ? order by f_create_time limit ?, ?"
		args = make([]interface{}, 0)
		args = append(args, aid, offset, limit)
		res, err = cdbClient.Query(sql, args)
		if err != nil {
			logger.Error("QueryDB error: %v", err)
			return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", util.GetCallee(), err))
		}
		if len(res) == 0 {
			return -1, nil, errors.NewDBNilError(errors.WithMsg("DB t_team has no activity: %v team", aid))
		}
		for _, item := range res {
			team := &meta.Team{}
			teamType, _ := strconv.Atoi(item["f_team_type"])
			teamFlag, _ := strconv.Atoi(item["f_team_flag"])
			status, _ := strconv.Atoi(item["f_status"])
			team.ActivityId = item["f_activity_id"]
			team.TeamId = item["f_team_id"]
			team.TeamType = int32(teamType)
			team.TeamDesc = item["f_team_desc"]
			team.TeamCreator = item["f_team_creator"]
			team.TeamLeader = item["f_team_leader"]
			team.TeamFlag = meta.Team_TeamFlag(teamFlag)
			team.Status = int32(status)
			teams = append(teams, team)
		}
	}
	return total, teams, nil
}

// UpdateActivityTeam only change name now...
func UpdateActivityTeam(activityID, teamID, teamName string) error {
	// change db
	args := make([]interface{}, 0)
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "update t_team set f_modify_time=?, f_team_desc=?" +
		" where f_activity_id = ? and f_team_id = ?"
	args = append(args, now, teamName, activityID, teamID)
	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("sql = %s, update DB record err = %v", sql, err)
		return err
	}
	logger.Debug("aid = %s, teamId = %s update db team name: %v successfully", activityID, teamID, teamName)

	// change redis
	oper := &TeamOper{}
	result, err := oper.queryTeamsRedis(activityID, []string{teamID})
	if err != nil {
		logger.Error("get activity: %v team: %v redis record err = %v", activityID, teamID, err)
		return err
	}
	if len(result) == 0 {
		logger.Error("get activity: %v team: %v redis no record error", activityID, teamID)
		return errors.NewActivityMgrInternalError(errors.WithMsg("get activity: %v team: %v redis no record error", activityID, teamID))
	}
	team := result[0]
	team.TeamDesc = teamName
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_META_KEY_PREFIX, activityID)
	field := fmt.Sprintf("%s:%s", common.TEAM_FIELD_INFO_PREFIX, teamID)
	content, err := util.Pb2Json(team)
	if err != nil {
		logger.Error("aid = %s, teamId = %s serial pb to json err = %v", team.ActivityId, team.TeamId, err)
		return err
	}
	if _, err = cacheclient.RedisClient.HSet(key, field, content).Result(); err != nil {
		logger.Error("key = %s, content = %s, redis hset err = %v", key, content, err)
		return err
	}
	logger.Debug("key = %s, content = %s, sync to redis successfully", key, content)

	logger.Info("aid = %s, teamId = %s change name: %v successfully", activityID, teamID, teamName)
	return nil
}

func ModifyTeam(teamMeta *meta.Team) error {
	oper := &TeamOper{}
	err := oper.updateTeamDB(teamMeta)
	if err != nil {
		logger.Error("updateTeamDB error: %v", err)
		return err
	}
	err = oper.syncTeamRedis(teamMeta)
	if err != nil {
		logger.Error("syncTeamRedis error: %v", err)
		return err
	}
	return nil
}
