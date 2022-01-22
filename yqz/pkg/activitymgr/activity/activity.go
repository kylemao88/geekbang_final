package activity

import (
	"fmt"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/common/coupons"
	"strconv"
	"strings"
	"time"

	activitypb "git.code.oa.com/gongyi/yqz/api/activitymgr"
	meta "git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/common"
	steps2 "git.code.oa.com/gongyi/yqz/pkg/activitymgr/steps"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/proxyclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	fundClient "git.code.oa.com/gongyi/yqz/pkg/fundmgr/client"
	stepClient "git.code.oa.com/gongyi/yqz/pkg/stepmgr/client"
	"github.com/go-redis/redis"
)

const (
	PAGE_CNT_MAX       = 30 // 每页最多查询30条记录
	BGPIC_STATUS_DRAFT = 0  // 图片未审核
	BGPIC_STATUS_OK    = 1  // 图片审核通过
	BGPIC_STATUS_ERR   = 2  // 图片审核失败
)

var (
	cdbClient dbclient.DBClient
)

type ActivityOper struct {
	ca *meta.CustomizedActivity
}

// 从redis查询定制活动的基础配置
func (o *ActivityOper) queryActivityRedis(aid string) (*meta.Activity, error) {
	a := &meta.Activity{}

	key := fmt.Sprintf("%s:%s", common.CUSTOMIZED_ACTIVITY_META_KEY_PREFIX, aid)

	// ----------------------------- 活动配置json格式 -----------------------------
	result, err := cacheclient.RedisClient.HGet(key, common.ACTIVITY_FIELD_INFO).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s fetch activity from redis err = %v", aid, err)
		return a, errors.NewRedisClientError(errors.WithMsg("key = %s, redis hget err = %v", key, err))
	}
	// redis没有活动配置
	if len(result) <= 0 || err == redis.Nil {
		logger.Error("key = %s redis activity meta is empty error", key)
		return a, errors.NewInternalError(errors.WithMsg("key = %s redis is not exist err", key))
	}

	logger.Debug("activity result = %s", result)
	if err = util.Json2Pb(result, a); err != nil {
		logger.Error("content = %s, json to pb err = %v", result, err)
		return a, err
	}

	//content, _ := util.Pb2Json(a)
	//logger.Debug("activity content = %s", content)

	return a, nil
}

// 从db查询定制活动的基础配置
func (o *ActivityOper) queryActivityDB(aid string) (*meta.Activity, error) {
	a := &meta.Activity{}

	sql := "select f_activity_id, f_match_event_id, f_activity_creator, f_company_id, f_type, f_desc, f_slogan, f_route_id, " +
		"f_bgpic, f_bgpic_status, f_start_time, f_end_time, f_team_mode, f_team_off, f_rule," +
		"f_team_member_limit, f_show_sponsor, f_color, f_match_off, f_status, f_white_type, f_cover, f_relation_desc, f_forward_pic, f_create_time, f_modify_time from t_activity_info where f_activity_id = ? limit 1"
	args := make([]interface{}, 0)
	args = append(args, aid)

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("aid = %s, QueryDB error, err = %v", aid, err)
		return a, errors.NewDBClientError(errors.WithMsg("aid = %s, QueryDB error, err = %v", aid, err))
	}
	if 0 == len(res) {
		logger.Error("aid = %s, sql = %s DB is not exist err", aid, sql)
		return a, errors.NewInternalError(errors.WithMsg("aid = %s, sql = %s DB is not exist err", aid, sql))
	}
	for _, item := range res {
		activityType, _ := strconv.Atoi(item["f_type"])
		bgpicStatus, _ := strconv.Atoi(item["f_bgpic_status"])
		teamMode, _ := strconv.Atoi(item["f_team_mode"])
		teamOff, _ := strconv.Atoi(item["f_team_off"])
		teamMemberLimit, _ := strconv.Atoi(item["f_team_member_limit"])
		showSponsor, _ := strconv.Atoi(item["f_show_sponsor"])
		matchOff, _ := strconv.Atoi(item["f_match_off"])
		status, _ := strconv.Atoi(item["f_status"])
		whiteType, _ := strconv.Atoi(item["f_white_type"])
		//
		a.ActivityId = item["f_activity_id"]
		a.MatchId = item["f_match_event_id"]
		a.ActivityCreator = item["f_activity_creator"]
		a.CompanyId = item["f_company_id"]
		a.Type = meta.Activity_CreateType(activityType)
		a.Desc = item["f_desc"]
		a.Slogan = item["f_slogan"]
		a.OrgName = item["f_org_name"]
		a.OrgHead = item["f_org_head"]
		a.RouteId = item["f_route_id"]
		a.Bgpic = item["f_bgpic"]
		a.BgpicStatus = int32(bgpicStatus)
		a.StartTime = item["f_start_time"]
		a.EndTime = item["f_end_time"]
		a.TeamMode = int32(teamMode)
		a.TeamOff = int32(teamOff)
		a.TeamMemberLimit = int32(teamMemberLimit)
		a.ShowSponsor = int32(showSponsor)
		a.Color = item["f_color"]
		a.MatchOff = int32(matchOff)
		a.Status = meta.Activity_Status(status)
		a.Rule = item["f_rule"]
		a.WhiteType = meta.Activity_WhiteType(whiteType)
		a.CreateTime = item["f_create_time"]
		a.Cover = item["f_cover"]
		a.RelationDesc = item["f_relation_desc"]
		a.ForwardPic = item["f_forward_pic"]
		return a, nil
	}

	return a, nil
}

func (o *ActivityOper) fetchActivity(aid string) (*meta.Activity, error) {
	var err error
	success := true
	a := &meta.Activity{}

	// 先查询redis
	a, err = o.queryActivityRedis(aid)
	if err != nil {
		success = false
	}

	if !success {
		// redis查询失败, 再尝试查询DB
		a, err = o.queryActivityDB(aid)
		if err != nil {
			return a, err
		}
		// 将活动信息回补到redis, 失败仅报错, 不返回错误
		var _ = o.syncActivityRedis(a)
	}

	return a, nil
}

// 定制活动基础配置存储到redis
func (o *ActivityOper) syncActivityRedis(activity *meta.Activity) error {
	key := fmt.Sprintf("%s:%s", common.CUSTOMIZED_ACTIVITY_META_KEY_PREFIX, activity.ActivityId)

	// ----------------------------- json 格式 -----------------------------
	// 配置内容为json格式
	// 优点: 配置为json, 支持嵌套struct, 支持复杂的结构体, 注意手动修改redis的json值要注意中文包含\转移符
	// 不足: 修改某个字段需要更新整个json
	content, err := util.Pb2Json(activity)
	if err != nil {
		logger.Error("aid = %s serial pb to json err = %v", activity.ActivityId, err)
		return err
	}

	logger.Debug("syncActivityRedis content = %+v", content)

	if _, err = cacheclient.RedisClient.HSet(key, common.ACTIVITY_FIELD_INFO, content).Result(); err != nil {
		logger.Error("aid = %s, key = %s, content = %s, redis hset err = %v", activity.ActivityId, key, content, err)
		return err
	}

	logger.Debug("aid = %s, key = %s, content = %s, sync to redis successfully", activity.ActivityId, key, content)
	return nil
}

// 定制活动基础配置存储到DB
func (o *ActivityOper) syncActivityDB(activity *meta.Activity) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "INSERT INTO t_activity_info (f_activity_id, f_match_event_id, f_activity_creator, f_company_id, f_type, f_desc, f_slogan, " +
		" f_org_name, f_org_head, f_route_id, f_bgpic, f_bgpic_status, f_start_time, f_end_time, f_team_mode, f_team_off, " +
		" f_team_member_limit, f_show_sponsor, f_color, f_supp_coupons, f_match_off, f_status, f_rule, f_white_type, f_cover, f_relation_desc, f_forward_pic, f_create_time, f_modify_time) " +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
		" ON DUPLICATE KEY UPDATE f_modify_time=?, f_activity_creator=?, f_company_id=?, f_type=?, f_desc=?, f_slogan=?, f_org_name=?, " +
		" f_org_head=?, f_route_id=?, f_bgpic=?, f_bgpic_status=?, f_start_time=?, f_end_time=?, f_team_mode=?, f_team_off=?, " +
		" f_team_member_limit=?, f_show_sponsor=?, f_color=?, f_supp_coupons=?, f_match_off=?, f_status=?, f_rule=?, f_white_type=?, f_cover=?, f_relation_desc=?, f_forward_pic=?"

	args := make([]interface{}, 0)
	args = append(args, activity.ActivityId, activity.MatchId, activity.ActivityCreator, activity.CompanyId, activity.Type, activity.Desc, activity.Slogan,
		activity.OrgName, activity.OrgHead, activity.RouteId, activity.Bgpic, activity.BgpicStatus, activity.StartTime, activity.EndTime,
		activity.TeamMode, activity.TeamOff, activity.TeamMemberLimit, activity.ShowSponsor, activity.Color, activity.SuppCoupons, activity.MatchOff, activity.Status,
		activity.Rule, activity.WhiteType, activity.Cover, activity.RelationDesc, activity.ForwardPic, now, now)
	args = append(args, now, activity.ActivityCreator, activity.CompanyId, activity.Type, activity.Desc, activity.Slogan, activity.OrgName, activity.OrgHead,
		activity.RouteId, activity.Bgpic, activity.BgpicStatus, activity.StartTime, activity.EndTime, activity.TeamMode, activity.TeamOff,
		activity.TeamMemberLimit, activity.ShowSponsor, activity.Color, activity.SuppCoupons, activity.MatchOff, activity.Status, activity.Rule, activity.WhiteType, activity.Cover, activity.RelationDesc, activity.ForwardPic)

	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("aid = %s, sql = %s, update DB record err = %v", activity.ActivityId, sql, err)
		return err
	}

	logger.Debug("aid = %s, status = %v, update activity DB record successfully", activity.ActivityId, activity.Status)
	return nil
}

// 查询活动统计(参入活动人数、小队数量、参与小队人数)
func (o *ActivityOper) queryActivityStats(aid string) (*activitypb.ActivityStats, error) {
	stats := &activitypb.ActivityStats{}

	// TODO: 参与小队人数, 需求暂时不需要该字段
	var teamMember int64

	// 参与活动人数
	rankKey := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)
	activityMember, err := cacheclient.RedisClient.ZCard(rankKey).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, key = %s, query redis activity member err = %v", aid, rankKey, err)
		return stats, err
	}

	// 小队数量
	teamKey := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)
	teamCnt, err := cacheclient.RedisClient.ZCard(teamKey).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, key = %s, query redis teams num err = %v", aid, teamKey, err)
		return stats, err
	}

	// 活动数据最近更新时间
	stepsKey := fmt.Sprintf("%s:%s", common.ACTIVITY_STEPS_KEY_PREFIX, aid)
	updateTime, err := cacheclient.RedisClient.HGet(stepsKey, common.UPDATE_TIME_FIELD).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, key = %s, query redis activity update time err = %v", aid, stepsKey, err)
		return stats, err
	}

	// 活动的总步数
	activitySteps, err := cacheclient.RedisClient.HGet(stepsKey, common.ACTIVITY_STEPS_FIELD).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, key = %s, query redis activity steps err = %v", aid, stepsKey, err)
		return stats, err
	}

	stats.ActivityMember = activityMember
	stats.TeamMember = teamMember
	stats.TeamCnt = teamCnt
	stats.UpdateTime = updateTime
	stats.ActivitySteps, _ = strconv.ParseInt(activitySteps, 10, 64)

	return stats, nil
}

// 查询活动的用户公益金排名
// 返回值: bool: true表示用户参与活动, int: 用户在活动的排名
func (o *ActivityOper) queryActivityUserFundRank(aid, oid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_FUND_RANK_KEY_PREFIX, aid)

	rank, err := cacheclient.RedisClient.ZRevRank(key, oid).Result()

	// 用户没有加入活动, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, query redis activity user fund rank err = %v", aid, oid, key, err)
		return false, 0, err
	}

	return true, rank + 1, nil
}

// 查询活动的小队公益金排名
// 返回值: bool: true表示活动存在此小队, int: 小队在活动的排名
func (o *ActivityOper) queryActivityTeamFundRank(aid, teamid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_FUND_RANK_KEY_PREFIX, aid)

	rank, err := cacheclient.RedisClient.ZRevRank(key, teamid).Result()

	// 用户没有加入活动, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("aid = %s, teamid = %s, key = %s, query redis activity team fund rank err = %v", aid, teamid, key, err)
		return false, 0, err
	}

	return true, rank + 1, nil
}

// 查询活动的用户步数排名
// 返回值: bool: true表示用户参与活动, int: 用户在活动的排名
func (o *ActivityOper) queryActivityUserRank(aid, oid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)

	rank, err := cacheclient.RedisClient.ZRevRank(key, oid).Result()

	// 用户没有加入活动, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, query redis activity user rank err = %v", aid, oid, key, err)
		return false, 0, err
	}

	return true, rank + 1, nil
}

// 查询活动的小队步数排名
// 返回值: bool: true表示活动存在该小队, int: 小队在活动的排名
func (o *ActivityOper) queryActivityTeamStepRank(aid, teamid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)

	rank, err := cacheclient.RedisClient.ZRevRank(key, teamid).Result()

	// 活动不存在此小队, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("aid = %s, teamid = %s, key = %s, query redis activity user rank err = %v", aid, teamid, key, err)
		return false, 0, err
	}

	return true, rank + 1, nil
}

// 查询活动的用户步数
// 返回值: bool: true表示用户参与活动, int: 用户在活动的步数
func (o *ActivityOper) queryActivityUserStep(aid, oid string) (bool, int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)

	step, err := cacheclient.RedisClient.ZScore(key, oid).Result()

	// 用户没有加入活动, 直接返回
	if redis.Nil == err {
		return false, 0, nil
	}
	if err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, query redis activity user rank err = %v", aid, oid, key, err)
		return false, 0, err
	}

	return true, int64(step), nil
}

// 从db查询用户创建活动的数量（状态未支付除外）
func (o *ActivityOper) queryUserAllCreateSizeDB(oid string) (int64, error) {
	sql := "select count(*) from t_activity_info where f_activity_creator = ? and f_status != 0 and f_status != 4"
	args := make([]interface{}, 0)
	args = append(args, oid)

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("oid = %s, QueryDB error, err = %v", oid, err)
		return 0, errors.NewDBClientError(errors.WithMsg("oid = %s, QueryDB error, err = %v", oid, err))
	}

	var size int64
	for _, item := range res {
		size, _ = strconv.ParseInt(item["count(*)"], 0, 64)
		return size, nil
	}

	return size, nil
}

func (o *ActivityOper) queryUserCreateSize(oid string) (int64, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_CREATE_KEY_PREFIX, oid)

	size, err := cacheclient.RedisClient.ZCard(key).Result()
	if err != nil && err != redis.Nil {
		logger.Error("oid = %s, key = %s, redis query user create err = %v", oid, key, err)
		return 0, err
	}

	return size, nil
}

// 查询用户创建的活动列表, 第一页page=0
func (o *ActivityOper) queryUserCreate(oid string, page int64, pgCnt int64) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page + 1) * pgCnt
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_CREATE_KEY_PREFIX, oid)

	results, err := cacheclient.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if redis.Nil == err {
		results = make([]redis.Z, 0)
		return results, nil
	}
	if err != nil && err != redis.Nil {
		results = make([]redis.Z, 0)
		logger.Error("oid = %s, key = %s, page = %d, pgCnt = %d, redis query user create err = %v", oid, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询个人公益金排名(根据lastKey翻页, 避免排名实时变化太快?) 第一页page=0
func (o *ActivityOper) queryUserFundRank(aid string, page int64, pgCnt int64) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_FUND_RANK_KEY_PREFIX, aid)

	results, err := cacheclient.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if redis.Nil == err {
		results = make([]redis.Z, 0)
		return results, nil
	}
	if err != nil && err != redis.Nil {
		results = make([]redis.Z, 0)
		logger.Error("aid = %s, key = %s, page = %d, pgCnt = %d redis query user rank err = %v", aid, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询个人赛道排名(根据lastKey翻页, 避免排名实时变化太快?) 第一页page=0
func (o *ActivityOper) queryUserRank(aid string, page int64, pgCnt int64, rankType string) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	// rankType=="step"  //default
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)
	if rankType == common.RAND_TYPE_FUND { // rankType=="fund"
		key = fmt.Sprintf("%s:%s", common.ACTIVITY_USER_FUND_RANK_KEY_PREFIX, aid)
	}

	results, err := cacheclient.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if redis.Nil == err {
		results = make([]redis.Z, 0)
		return results, nil
	}
	if err != nil && err != redis.Nil {
		results = make([]redis.Z, 0)
		logger.Error("aid = %s, key = %s, page = %d, pgCnt = %d redis query user rank err = %v", aid, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询个人赛道逆向排名(根据lastKey翻页, 避免排名实时变化太快?) 第一页page=0
func (o *ActivityOper) queryUserDescRank(aid string, page int64, pgCnt int64) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt > PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)

	results, err := cacheclient.RedisClient.ZRangeWithScores(key, start, stop).Result()
	if redis.Nil == err {
		results = make([]redis.Z, 0)
		return results, nil
	}
	if err != nil && err != redis.Nil {
		results = make([]redis.Z, 0)
		logger.Error("aid = %s, key = %s, page = %d, pgCnt = %d redis query user desc rank err = %v", aid, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

// 查询小队赛道排名(翻页查询), 第一页page=0
func (o *ActivityOper) queryTeamRank(aid string, page int64, pgCnt int64, rankType string) ([]redis.Z, error) {
	if page <= 0 {
		page = 0
	}
	if pgCnt <= 0 || pgCnt >= PAGE_CNT_MAX {
		pgCnt = PAGE_CNT_MAX
	}

	start := page * pgCnt
	stop := (page+1)*pgCnt - 1
	// rankType == "step"
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)
	if rankType == common.RAND_TYPE_FUND {
		key = fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_FUND_RANK_KEY_PREFIX, aid)
	}

	results, err := cacheclient.RedisClient.ZRevRangeWithScores(key, start, stop).Result()
	if redis.Nil == err {
		results = make([]redis.Z, 0)
		return results, nil
	}
	if err != nil && err != redis.Nil {
		results = make([]redis.Z, 0)
		logger.Error("aid = %s, key = %s, page = %d, pgCnt = %d redis query team rank err = %v", aid, key, page, pgCnt, err)
		return results, err
	}

	return results, nil
}

func (o *ActivityOper) initTeamFundRank(aid, teamId string, teamFunds int64) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_FUND_RANK_KEY_PREFIX, aid)

	// 更新小队的总公益金和排名
	if _, err := cacheclient.RedisClient.ZIncrBy(key, float64(teamFunds), teamId).Result(); err != nil {
		logger.Error("aid = %s, teamId = %s, key = %s, steps = %d update team fund rank err = %v", aid, teamId, key, teamFunds, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, key = %s, team_steps = %d, update team fund rank successfully", aid, teamId, key, teamFunds)
	return nil
}

// 更新活动的小队排行榜
func (o *ActivityOper) updateTeamRank(aid, teamId string, teamSteps int64) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)

	// 更新小队的总步数和排名
	if _, err := cacheclient.RedisClient.ZIncrBy(key, float64(teamSteps), teamId).Result(); err != nil {
		logger.Error("aid = %s, teamId = %s, key = %s, steps = %d update team rank err = %v", aid, teamId, key, teamSteps, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, key = %s, team_steps = %d, update team rank successfully", aid, teamId, key, teamSteps)
	return nil
}

// 更新用户创建的活动列表
func (o *ActivityOper) updateUserCreate(aid, oid string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_CREATE_KEY_PREFIX, oid)

	_, err := cacheclient.RedisClient.ZRank(key, aid).Result()
	if err != nil && err != redis.Nil {
		logger.Error("aid = %s, oid = %s, key = %s, redis query user create err = %v", aid, oid, key, err)
		return err
	}

	// 用户之前没有创建活动
	if err == redis.Nil {
		unix := time.Now().Unix()
		z := redis.Z{
			Score:  float64(unix),
			Member: aid,
		}

		if _, err := cacheclient.RedisClient.ZAdd(key, z).Result(); err != nil {
			logger.Error("aid = %s, oid = %s, key = %s, redis add user create err = %v", aid, oid, key, err)
			return err
		}

		logger.Debug("aid = %s, oid = %s, key = %s, add user create activity successfully", aid, oid, key)
	}

	return nil
}

// 删除用户创建的活动（活动过期或者活动停止需要删除）
func (o *ActivityOper) removeUserCreate(aid, oid string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_CREATE_KEY_PREFIX, oid)

	if _, err := cacheclient.RedisClient.ZRem(key, aid).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, oid = %s, key = %s, redis remove user create err = %v", aid, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, remove user create activity successfully", aid, oid, key)
	return nil
}

// 调整活动总公益金, 用户加入/退出活动时触发
func (o *ActivityOper) modifyActivityFunds(aid, oid string, funds int64) error {
	//
	if funds == 0 {
		return nil
	}
	// 初略更新活动的总公益金, 然后在用户每次配捐获取公益金时刷新公益金数据
	statsKey := fmt.Sprintf("%s:%s", common.ACTIVITY_FUND_KEY_PREFIX, aid)
	if _, err := cacheclient.RedisClient.HIncrBy(statsKey, common.ACT_FUNDS_FIELD, funds).Result(); err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, update redis activity funds err = %v", aid, oid, statsKey, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, funds = %d, add activity funds successfully", aid, oid, statsKey, funds)
	return nil
}

// 初略调整活动总步数, 用户加入活动时触发
func (o *ActivityOper) modifyActivitySteps(aid, oid string, steps int64) error {
	// 初略更新活动的总步数, 活动总步数具体在定时任务每隔1分钟更新1次
	statsKey := fmt.Sprintf("%s:%s", common.ACTIVITY_STEPS_KEY_PREFIX, aid)
	if _, err := cacheclient.RedisClient.HIncrBy(statsKey, common.ACTIVITY_STEPS_FIELD, steps).Result(); err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, update redis activity steps err = %v", aid, oid, statsKey, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, steps = %d, add activity steps successfully", aid, oid, statsKey, steps)
	return nil
}

// 更新活动的个人公益金排行榜
func (o *ActivityOper) updateUserFundsRank(aid, oid string, funds int64) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_FUND_RANK_KEY_PREFIX, aid)
	z := redis.Z{
		Score:  float64(funds),
		Member: oid,
	}

	if _, err := cacheclient.RedisClient.ZAdd(key, z).Result(); err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, update activity user fund rank err = %v", aid, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, funds = %d, update activity user fund rank successfully", aid, oid, key, funds)
	return nil
}

// 更新活动的个人排行榜
func (o *ActivityOper) updateUserRank(aid, oid string, steps int64) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)
	z := redis.Z{
		Score:  float64(steps),
		Member: oid,
	}

	if _, err := cacheclient.RedisClient.ZAdd(key, z).Result(); err != nil {
		logger.Error("aid = %s, oid = %s, key = %s, update activity user rank err = %v", aid, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, steps = %d, update activity user rank successfully", aid, oid, key, steps)
	return nil
}

// 删除活动的成员公益金排名（退出活动）
func (o *ActivityOper) deleteUserFundRank(aid, oid string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_FUND_RANK_KEY_PREFIX, aid)

	if _, err := cacheclient.RedisClient.ZRem(key, oid).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, oid = %s, key = %s, remove activity user fund rank err = %v", aid, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, remove activity user fund rank successfully", aid, oid, key)
	return nil
}

// 删除活动的成员步数排名（退出活动）
func (o *ActivityOper) deleteUserRank(aid, oid string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, aid)

	if _, err := cacheclient.RedisClient.ZRem(key, oid).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, oid = %s, key = %s, remove activity user rank err = %v", aid, oid, key, err)
		return err
	}

	logger.Debug("aid = %s, oid = %s, key = %s, remove activity user rank successfully", aid, oid, key)
	return nil
}

func (o *ActivityOper) makeActivity(activity *meta.Activity) error {
	// 1.生成唯一活动id
	aid, err := util.GetTransId("ca_")
	if err != nil {
		logger.Error("GetTransId error, err = %v, err", err)
		return err
	}

	// 2.初始化活动配置
	activity.ActivityId = aid
	activity.BgpicStatus = BGPIC_STATUS_DRAFT // 名称和图片接入UGC审核平台

	// 3.根据发起类型区分活动初始状态, 手机端发起, 支付成功才变成未开始状态
	if meta.Activity_MOBILE == activity.Type {
		activity.Status = meta.Activity_DRAFT
	} else {
		activity.Status = meta.Activity_READY
	}

	return nil
}

// 获取活动状态
func (o *ActivityOper) fillActivityStatus(activity *meta.Activity) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	// 未审核状态, 直接返回
	if meta.Activity_DRAFT == activity.Status {
		return nil
	}

	// 结束或暂停状态, 直接返回
	if meta.Activity_EXPIRED == activity.Status ||
		meta.Activity_STOP == activity.Status ||
		meta.Activity_SUSPEND == activity.Status {
		return nil
	}

	// 判断活动是否进行中
	if activity.StartTime > now { // 活动未开始
		activity.Status = meta.Activity_READY
	} else if activity.EndTime < now { // 活动已结束
		activity.Status = meta.Activity_EXPIRED
	} else { // 活动进行中
		activity.Status = meta.Activity_RUNNING
	}

	return nil
}

func InitComponent(db dbclient.DBClient, proxyConf *proxyclient.ProxyConfig, fundSvrConf, stepSvrConf *proxyclient.ServiceConfig) error {
	cdbClient = db

	if proxyConf != nil && fundSvrConf != nil {
		if err := fundClient.InitClient(proxyConf, fundSvrConf); err != nil {
			return err
		}
	}

	if proxyConf != nil && stepSvrConf != nil {
		if err := stepClient.InitClient(proxyConf, stepSvrConf); err != nil {
			return err
		}
	}

	return nil
}

// 创建定制活动
func CreateActivity(activity *meta.Activity, matchInfo *meta.MatchInfo, matchRule *meta.MatchRule) (*meta.CustomizedActivity, error) {
	oper := &ActivityOper{}
	ca := &meta.CustomizedActivity{}

	// 1.创建定制活动的基础配置
	if err := oper.makeActivity(activity); err != nil {
		logger.Error("create activity err = %v", err)
		return ca, err
	}

	aid := activity.ActivityId

	// 2.创建定制活动的配捐功能(RPC)
	matchResp, err := fundClient.CreateMatchEvent(matchInfo, matchRule)
	if err != nil || 0 != matchResp.Header.Code {
		logger.Error("aid = %s create match event err = %v", aid, err)
		return ca, errors.NewInternalError(errors.WithMsg("aid = %s create match event err = %v", aid, err))
	}
	ca.Match = matchResp.MatchEvent
	ca.Activity = activity
	ca.Activity.MatchId = ca.Match.MatchInfo.FEventId

	// 3.同步活动配置到redis
	if err = oper.syncActivityRedis(ca.Activity); err != nil {
		return ca, err
	}

	// 4.同步活动配置到db
	if err = oper.syncActivityDB(ca.Activity); err != nil {
		return ca, err
	}

	content, _ := util.Pb2Json(ca)
	logger.Debug("create customized activity: %s successfully", content)
	return ca, nil
}

// 查询定制活动配置
func QueryActivity(aid string) (*meta.CustomizedActivity, error) {
	oper := &ActivityOper{}
	ca := &meta.CustomizedActivity{}

	// 1.查询定制基础配置
	a, err := oper.fetchActivity(aid)
	if err != nil {
		return ca, err
	}

	// 2.获取活动状态
	if err = oper.fillActivityStatus(a); err != nil {
		return ca, err
	}

	// 3.查询活动的配捐功能(RPC)
	matchResp, err := fundClient.GetMatchEvent(a.MatchId)
	if err != nil || 0 != matchResp.Header.Code {
		logger.Error("aid = %s, eid = %s GetMatchEvent err = %v", aid, a.MatchId, err)
		return ca, errors.NewInternalError(errors.WithMsg("aid = %s, eid = %s get match event err = %v", aid, a.MatchId, err))
	}

	ca.Activity = a
	ca.Match = matchResp.MatchEvent

	return ca, nil
}

// QueryActivityMeta for get act meta info
func QueryActivityMeta(aid string) (*meta.CustomizedActivity, error) {
	oper := &ActivityOper{}
	ca := &meta.CustomizedActivity{}

	now := time.Now()
	// 1.查询定制基础配置
	a, err := oper.fetchActivity(aid)
	if err != nil {
		return ca, err
	}
	logger.Info("fetchActivity consume time: %v", time.Since(now))

	now = time.Now()
	// 2.获取活动状态
	if err = oper.fillActivityStatus(a); err != nil {
		return ca, err
	}
	logger.Info("fillActivityStatus consume time: %v", time.Since(now))

	ca.Activity = a
	return ca, nil
}

// 更新定制活动状态
func UpdateActivityStatus(aid string, status meta.Activity_Status) error {
	logger.Debug("aid = %s, status = %d, prepare to update", aid, status)

	oper := &ActivityOper{}

	// 查询定制基础配置
	a, err := oper.fetchActivity(aid)
	if err != nil {
		logger.Error("aid = %s, update status = %d, fetch activity err = %v", aid, status, err)
		return err
	}

	// TODO: 活动结束后不能再更改活动状态了? 那续费呢?
	a.Status = status

	// 同步更新redis和DB
	if err = oper.syncActivityRedis(a); err != nil {
		logger.Error("aid = %s, update status = %d, sync to redis err = %v", aid, status, err)
		return err
	}
	if err = oper.syncActivityDB(a); err != nil {
		logger.Error("aid = %s, update status = %d, sync to DB err = %v", aid, status, err)
		return err
	}

	logger.Debug("aid = %s, eid = %s, type = %d, update status = %d successfully", aid, a.MatchId, a.Type, status)
	return nil
}

// 查询活动统计
func QueryActivityStats(aid string) (*activitypb.ActivityStats, error) {
	oper := &ActivityOper{}
	stats, err := oper.queryActivityStats(aid)
	if err != nil {
		return stats, err
	}
	return stats, nil
}

// 查询活动的小队排名
func QueryTeamRank(aid string, page int64, size int64, rankType string) ([]redis.Z, error) {
	oper := &ActivityOper{}
	zs, err := oper.queryTeamRank(aid, page, size, rankType)
	if err != nil {
		return zs, err
	}
	return zs, nil
}


// 查询活动的用户排名
func QueryUserRank(aid string, page int64, size int64, rankType string) ([]redis.Z, error) {
	oper := &ActivityOper{}
	zs, err := oper.queryUserRank(aid, page, size, rankType)
	if err != nil {
		return zs, err
	}
	return zs, nil
}

// 查询活动的用户逆向排名
func QueryUserDescRank(aid string, page int64, size int64) ([]redis.Z, error) {
	oper := &ActivityOper{}
	zs, err := oper.queryUserDescRank(aid, page, size)
	if err != nil {
		return zs, err
	}
	return zs, nil
}

// 查询活动的用户公益金排名
// 返回值: bool: true表示用户参与活动, int: 用户在活动的排名
func QueryActivityUserFundRank(aid, oid string) (bool, int64, error) {
	oper := &ActivityOper{}
	return oper.queryActivityUserFundRank(aid, oid)
}

// 查询活动的用户步数排名
// 返回值: bool: true表示用户参与活动, int: 用户在活动的排名
func QueryActivityUserRank(aid, oid string) (bool, int64, error) {
	oper := &ActivityOper{}
	return oper.queryActivityUserRank(aid, oid)
}

// 查询活动的小队公益金排名
// 返回值: bool: true表示活动存在此小队, int: 小队在活动的排名
func QueryActivityTeamFundRank(aid, teamid string) (bool, int64, error) {
	oper := &ActivityOper{}
	return oper.queryActivityTeamFundRank(aid, teamid)
}

// 查询活动的用户步数
// 返回值: bool: true表示用户参与活动, int: 用户在活动的步数
func QueryActivityUserStep(aid, oid string) (bool, int64, error) {
	oper := &ActivityOper{}
	return oper.queryActivityUserStep(aid, oid)
}

// 从db查询用户创建活动的数量（状态未支付除外）
func QueryUserAllCreateSizeDB(oid string) (int64, error) {
	oper := &ActivityOper{}
	size, err := oper.queryUserAllCreateSizeDB(oid)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// 查询用户创建的活动列表
func QueryUserCreate(oid string, page int64, size int64) ([]redis.Z, error) {
	oper := &ActivityOper{}
	zs, err := oper.queryUserCreate(oid, page, size)
	if err != nil {
		return zs, err
	}
	return zs, nil
}

func QueryUserCreateSize(oid string) (int64, error) {
	oper := &ActivityOper{}
	size, err := oper.queryUserCreateSize(oid)
	if err != nil {
		return 0, err
	}
	return size, nil
}


func QueryActivityUserStats(aid, oid string) (*activitypb.ActivityUserStats, error) {
	oper := &ActivityOper{}
	stats := &activitypb.ActivityUserStats{}

	// 查询活动基础配置
	a, err := oper.fetchActivity(aid)
	if err != nil {
		return stats, err
	}

	// 查询用户在活动的排名
	join, rank, err := oper.queryActivityUserRank(aid, oid)
	if err != nil {
		return stats, err
	}

	// 查询用户的今日步数和活动总步数
	steps, err := steps2.QueryUserSteps(aid, oid)
	if err != nil {
		return stats, err
	}

	isCreator := false
	if a.ActivityCreator == oid {
		isCreator = true
	}

	// 查询用户的活动的配捐统计
	match, err := fundClient.GetUserMatchRecordByOffset(oid, aid, 0, 0)
	if err != nil {
		return stats, err
	}

	stats.Join = join
	stats.Rank = rank
	stats.IsCreator = isCreator
	stats.TotalSteps = steps.ActivitySteps
	stats.TodaySteps = steps.TodaySteps
	stats.MatchMoney = int64(match.TotalFunds)
	stats.MatchSteps = int64(match.TotalSteps)

	return stats, nil
}

func UpdateUserCreate(aid, oid string) error {
	oper := &ActivityOper{}
	return oper.updateUserCreate(aid, oid)
}

// 更新用户在活动的排名
func UpdateUserRank(aid, oid string, steps int64) error {
	oper := &ActivityOper{}
	return oper.updateUserRank(aid, oid, steps)
}

func UpdateUserFundsRank(aid, oid string, funds int64) error {
	oper := &ActivityOper{}
	return oper.updateUserFundsRank(aid, oid, funds)
}

// 删除用户的活动排名
func DeleteUserRank(aid, oid string) error {
	oper := &ActivityOper{}
	return oper.deleteUserRank(aid, oid)
}

func ModifyActivityFunds(aid, oid string, funds int64) error {
	oper := &ActivityOper{}
	return oper.modifyActivityFunds(aid, oid, funds)
}

func ModifyActivitySteps(aid, oid string, steps int64) error {
	oper := &ActivityOper{}
	return oper.modifyActivitySteps(aid, oid, steps)
}

//  更新小队在活动的公益金排名
func InitTeamFundRank(aid, teamId string) error {
	var funds int64 = 0
	oper := &ActivityOper{}
	return oper.initTeamFundRank(aid, teamId, funds)
}

// 更新小队在活动的步数排名
func UpdateTeamRank(aid, teamId string) error {
	var steps int64
	oper := &ActivityOper{}
	return oper.updateTeamRank(aid, teamId, steps)
}

// RemoveActivityTeamRank remove team rank in activity
func RemoveActivityTeamRank(aid, teamId string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, aid)
	// 删除活动的小队排名
	if _, err := cacheclient.RedisClient.ZRem(key, teamId).Result(); err != nil && err != redis.Nil {
		logger.Error("aid = %s, teamId = %s, key = %s, remove team rank err = %v", aid, teamId, key, err)
		return err
	}

	logger.Debug("aid = %s, team_id = %s, key = %s, remove team rank successfully", aid, teamId, key)
	return nil
}

// GetActivityList ...
func GetActivityList(offset, limit int32, activityID string, status int32, name string) (int32, []*meta.CustomizedActivity, []*meta.ActivityWhiteList, error) {
	total, activityList, err := listActivityDB(offset, limit, activityID, status, name)
	if err != nil {
		if errors.IsDBNilError(err) {
			return -1, nil, nil, nil
		}
		logger.Error("listActivityDB error: %v", err)
		return -1, nil, nil, err
	}

	if len(activityList) == 0 {
		logger.Debug("activityList is nil")
		return 0, nil, nil, nil
	}

	var aids []string
	for _, v := range activityList {
		aids = append(aids, v.ActivityId)
	}

	actWhites, err := listActivityWhiteListDB(aids)
	if err != nil {
		logger.Error("listActivityWhiteListDB error: %v", err)
		return -1, nil, nil, err
	}

	result := []*meta.CustomizedActivity{}
	// get match info
	for _, a := range activityList {
		ca := &meta.CustomizedActivity{}
		ca.Activity = a
		matchResp, err := fundClient.GetMatchEvent(a.MatchId)
		if err != nil || matchResp.Header.Code != 0 {
			logger.Error("eid = %s GetMatchEvent err = %v", a.MatchId, err)
		} else {
			ca.Match = matchResp.MatchEvent
		}
		result = append(result, ca)
	}

	return total, result, actWhites, nil
}

func listActivityDB(offset, limit int32, activityID string, status int32, name string) (int32, []*meta.Activity, error) {
	var total int32
	list := []*meta.Activity{}
	sql := "select count(*) from t_activity_info "
	sqlFull := "select f_activity_id, f_match_event_id, f_activity_creator, f_company_id, f_type, f_slogan, f_route_id, f_desc, " +
		"f_bgpic, f_bgpic_status, f_start_time, f_end_time, f_team_mode, f_team_off, f_rule, f_color, f_supp_coupons, " +
		"f_team_member_limit, f_show_sponsor, f_match_off, f_white_type, f_cover, f_relation_desc, f_forward_pic, f_status, f_modify_time from t_activity_info "

	args := make([]interface{}, 0)
	// status filter
	if status != -1 {
		sql += " where f_status = ?"
		sqlFull += "where f_status = ?"
		args = append(args, status)
	}
	// activity filter
	if len(activityID) != 0 {
		if len(args) != 0 {
			sql += " and f_activity_id = ?"
			sqlFull += " and f_activity_id = ?"
		} else {
			sql += " where f_activity_id = ?"
			sqlFull += " where f_activity_id = ?"
		}
		args = append(args, activityID)
	}
	// name filter
	if len(name) != 0 {
		if len(args) != 0 {
			sql += " and f_desc like ?"
			sqlFull += " and f_desc like ?"
		} else {
			sql += " where f_desc like ?"
			sqlFull += " where f_desc like ?"
		}
		args = append(args, fmt.Sprintf("%%%v%%", name))
	}

	// get count
	res, err := cdbClient.Query(sql, args)
	if err != nil {
		return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
	}
	if len(res) == 0 {
		logger.Info("t_activity_info status: %v, actID: %v, name: %v item is 0", status, activityID, name)
		return 0, []*meta.Activity{}, nil
	}
	count, _ := strconv.Atoi(res[0]["count(*)"])
	total = int32(count)

	// get items
	sqlFull += " order by f_create_time desc limit ?, ?"
	args = append(args, offset, limit)
	res, err = cdbClient.Query(sqlFull, args)
	if err != nil {
		return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
	}
	if len(res) == 0 {
		logger.Info("t_activity_info status: %v, actID: %v, name: %v offset: %v, limit: %v item is 0", status, activityID, name, offset, limit)
		return total, []*meta.Activity{}, nil
	}
	oper := &ActivityOper{}
	for _, item := range res {
		a := &meta.Activity{}
		activityType, _ := strconv.Atoi(item["f_type"])
		bgpicStatus, _ := strconv.Atoi(item["f_bgpic_status"])
		teamMode, _ := strconv.Atoi(item["f_team_mode"])
		teamOff, _ := strconv.Atoi(item["f_team_off"])
		teamMemberLimit, _ := strconv.Atoi(item["f_team_member_limit"])
		showSponsor, _ := strconv.Atoi(item["f_show_sponsor"])
		matchOff, _ := strconv.Atoi(item["f_match_off"])
		status, _ := strconv.Atoi(item["f_status"])
		whiteType, _ := strconv.Atoi(item["f_white_type"])
		a.ActivityId = item["f_activity_id"]
		a.MatchId = item["f_match_event_id"]
		a.ActivityCreator = item["f_activity_creator"]
		a.CompanyId = item["f_company_id"]
		a.Type = meta.Activity_CreateType(activityType)
		a.Desc = item["f_desc"]
		a.Slogan = item["f_slogan"]
		a.RouteId = item["f_route_id"]
		a.Bgpic = item["f_bgpic"]
		a.BgpicStatus = int32(bgpicStatus)
		a.StartTime = item["f_start_time"]
		a.EndTime = item["f_end_time"]
		a.TeamMode = int32(teamMode)
		a.TeamOff = int32(teamOff)
		a.TeamMemberLimit = int32(teamMemberLimit)
		a.ShowSponsor = int32(showSponsor)
		a.MatchOff = int32(matchOff)
		a.Status = meta.Activity_Status(status)
		a.Rule = item["f_rule"]
		a.Color = item["f_color"]
		a.Cover = item["f_cover"]
		a.RelationDesc = item["f_relation_desc"]
		a.SuppCoupons, _ = strconv.ParseBool(item["f_supp_coupons"])
		a.WhiteType = meta.Activity_WhiteType(whiteType)
		a.ForwardPic = item["f_forward_pic"]
		oper.fillActivityStatus(a)
		list = append(list, a)
	}
	return total, list, nil

	/*
		if status != -1 {
			sql := "select count(*) from t_activity_info where f_status = ?"
			args := make([]interface{}, 0)
			args = append(args, status)
			if len(activityID) != 0 {
				sql += " and f_activity_id = ?"
				args = append(args, activityID)
			}
			res, err := cdbClient.Query(sql, args)
			if err != nil {
				return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
			}
			if len(res) == 0 {
				logger.Info("t_activity_info status: %v item is 0", status)
				return 0, []*meta.Activity{}, nil
			}
			count, _ := strconv.Atoi(res[0]["count(*)"])
			total = int32(count)
			if total == 0 {
				logger.Info("t_activity_info status: %v item is 0", status)
				return 0, []*meta.Activity{}, nil
			}

			sql = "select f_activity_id, f_match_event_id, f_activity_creator, f_type, f_slogan, f_route_id, f_desc, " +
				"f_bgpic, f_bgpic_status, f_start_time, f_end_time, f_team_mode, f_team_off, f_rule," +
				"f_team_member_limit, f_show_sponsor, f_match_off, f_status, f_modify_time from t_activity_info where f_status = ?"
			args = make([]interface{}, 0)
			args = append(args, status)
			if len(activityID) != 0 {
				sql += " and f_activity_id = ?"
				args = append(args, activityID)
			}
			sql += " order by f_create_time desc limit ?, ?"
			args = append(args, offset, limit)
			res, err = cdbClient.Query(sql, args)
			if err != nil {
				return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
			}
			if len(res) == 0 {
				logger.Info("db t_activity_info status: %v, offset: %v, limit: %v", status, offset, limit)
				return 0, []*meta.Activity{}, nil
			}
			oper := &ActivityOper{}
			for _, item := range res {
				a := &meta.Activity{}
				activityType, _ := strconv.Atoi(item["f_type"])
				bgpicStatus, _ := strconv.Atoi(item["f_bgpic_status"])
				teamMode, _ := strconv.Atoi(item["f_team_mode"])
				teamOff, _ := strconv.Atoi(item["f_team_off"])
				teamMemberLimit, _ := strconv.Atoi(item["f_team_member_limit"])
				showSponsor, _ := strconv.Atoi(item["f_show_sponsor"])
				matchOff, _ := strconv.Atoi(item["f_match_off"])
				status, _ := strconv.Atoi(item["f_status"])
				a.ActivityId = item["f_activity_id"]
				a.MatchId = item["f_match_event_id"]
				a.ActivityCreator = item["f_activity_creator"]
				a.Type = meta.Activity_CreateType(activityType)
				a.Desc = item["f_desc"]
				a.Slogan = item["f_slogan"]
				a.RouteId = item["f_route_id"]
				a.Bgpic = item["f_bgpic"]
				a.BgpicStatus = int32(bgpicStatus)
				a.StartTime = item["f_start_time"]
				a.EndTime = item["f_end_time"]
				a.TeamMode = int32(teamMode)
				a.TeamOff = int32(teamOff)
				a.TeamMemberLimit = int32(teamMemberLimit)
				a.ShowSponsor = int32(showSponsor)
				a.MatchOff = int32(matchOff)
				a.Status = meta.Activity_Status(status)
				a.Rule = item["f_rule"]
				oper.fillActivityStatus(a)
				list = append(list, a)
			}
		} else {
			sql := "select count(*) from t_activity_info"
			args := make([]interface{}, 0)
			if len(activityID) != 0 {
				sql += " where f_activity_id = ?"
				args = append(args, activityID)
			}
			res, err := cdbClient.Query(sql, args)
			if err != nil {
				return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
			}
			if len(res) == 0 {
				logger.Info("t_activity_info item is 0")
				return 0, []*meta.Activity{}, nil
			}
			count, _ := strconv.Atoi(res[0]["count(*)"])
			total = int32(count)
			if total == 0 {
				logger.Info("t_activity_info status: %v item is 0", status)
				return 0, []*meta.Activity{}, nil
			}

			sql = "select f_activity_id, f_match_event_id, f_activity_creator, f_type, f_slogan, f_route_id, f_desc, " +
				"f_bgpic, f_bgpic_status, f_start_time, f_end_time, f_team_mode, f_team_off, f_rule," +
				"f_team_member_limit, f_show_sponsor, f_match_off, f_status, f_modify_time from t_activity_info "
			args = make([]interface{}, 0)
			if len(activityID) != 0 {
				sql += " where f_activity_id = ?"
				args = append(args, activityID)
			}
			sql += " order by f_create_time desc limit ?, ?"
			args = append(args, offset, limit)
			res, err = cdbClient.Query(sql, args)
			if err != nil {
				return -1, nil, errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
			}
			if len(res) == 0 {
				logger.Info("db t_activity_info offset: %v, limit: %v", offset, limit)
				return 0, []*meta.Activity{}, nil
			}
			oper := &ActivityOper{}
			for _, item := range res {
				a := &meta.Activity{}
				activityType, _ := strconv.Atoi(item["f_type"])
				bgpicStatus, _ := strconv.Atoi(item["f_bgpic_status"])
				teamMode, _ := strconv.Atoi(item["f_team_mode"])
				teamOff, _ := strconv.Atoi(item["f_team_off"])
				teamMemberLimit, _ := strconv.Atoi(item["f_team_member_limit"])
				showSponsor, _ := strconv.Atoi(item["f_show_sponsor"])
				matchOff, _ := strconv.Atoi(item["f_match_off"])
				status, _ := strconv.Atoi(item["f_status"])
				a.ActivityId = item["f_activity_id"]
				a.MatchId = item["f_match_event_id"]
				a.ActivityCreator = item["f_activity_creator"]
				a.Type = meta.Activity_CreateType(activityType)
				a.Desc = item["f_desc"]
				a.Slogan = item["f_slogan"]
				a.RouteId = item["f_route_id"]
				a.Bgpic = item["f_bgpic"]
				a.BgpicStatus = int32(bgpicStatus)
				a.StartTime = item["f_start_time"]
				a.EndTime = item["f_end_time"]
				a.TeamMode = int32(teamMode)
				a.TeamOff = int32(teamOff)
				a.TeamMemberLimit = int32(teamMemberLimit)
				a.ShowSponsor = int32(showSponsor)
				a.MatchOff = int32(matchOff)
				a.Status = meta.Activity_Status(status)
				a.Rule = item["f_rule"]
				oper.fillActivityStatus(a)
				list = append(list, a)
			}
		}
		if len(activityID) != 0 {
			total = int32(len(list))
		}
		return total, list, nil
	*/
}

// DeleteActivity ...
func DeleteActivity(activityID string, opType activitypb.DeleteActivityRequest_Ops, operator string) error {
	// get activity status
	oper := &ActivityOper{}
	activityInfo, err := oper.fetchActivity(activityID)
	if err != nil {
		logger.Error("aid = %s fetchActivity err = %v", activityID, err)
		return err
	}
	if err = oper.fillActivityStatus(activityInfo); err != nil {
		logger.Error("aid = %s fillActivityStatus err = %v", activityID, err)
		return err
	}

	// filter forbid ops,
	// only for user create activity, platform create activity no need to check this condition
	if activityInfo.Type == meta.Activity_MOBILE && activityInfo.Status != meta.Activity_DRAFT {
		logger.Error("aid: %s, type: %v, status: %v not draft state", activityID, activityInfo.Type, activityInfo.Status)
		return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid: %s, type: %v, status: %v not draft state", activityID, activityInfo.Type, activityInfo.Status))
	}
	switch opType {
	case activitypb.DeleteActivityRequest_SYSTEM:
		logger.Info("admin try to delete activityID: %v", activityID)
	case activitypb.DeleteActivityRequest_USER:
		if activityInfo.Type != meta.Activity_MOBILE || activityInfo.ActivityCreator != operator {
			logger.Error("activityID: %v, ops: %v, operator: %v not allow to change, activity type: %v, creator: %v", activityID, opType, operator, activityInfo.Type, activityInfo.ActivityCreator)
			return errors.NewActivityMgrOpsForbidError(errors.WithMsg("activityID: %v, ops: %v, operator: %v not allow to change, activity type: %v, creator: %v", activityID, opType, operator, activityInfo.Type, activityInfo.ActivityCreator))
		}
		logger.Info("operator: %v try delete activityID: %v", operator, activityID)
	default:
		logger.Error("ops type unknown error, activityID: %v, ops: %v, operator: %v", activityID, opType, operator)
		err := errors.NewActivityMgrOpsForbidError(errors.WithMsg("ops type unknown error, activityID: %v, ops: %v, operator: %v", activityID, opType, operator))
		return err
	}

	// get activity properties
	status, err := oper.queryActivityStats(activityID)
	if err != nil {
		logger.Error("aid = %s queryActivityStats err = %v", activityID, err)
		return err
	}
	if status.TeamCnt != 0 {
		logger.Error("aid: %s, team_count: %v", activityID, status.TeamCnt)
		return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid: %s, team_count: %v", activityID, status.TeamCnt))
	}

	// set activity status
	sql := "UPDATE t_activity_info SET f_status = 4 WHERE f_activity_id = ?"
	args := make([]interface{}, 0)
	args = append(args, activityID)
	res, err := cdbClient.ExecSQL(sql, args)
	if err != nil {
		return errors.NewDBClientError(errors.WithMsg("QueryDB error: %v", err))
	}
	logger.Infof("delete from db activity: %v row: %v", activityID, res)

	// delete relative redis
	err = deleteActivityRelativeRedis(activityID, operator)
	if err != nil {
		return err
	}

	return nil
}

func deleteActivityRelativeRedis(activityID, oid string) error {
	// delete redis
	key := fmt.Sprintf("%s:%s", common.CUSTOMIZED_ACTIVITY_META_KEY_PREFIX, activityID)
	// can not use del command, just use expire
	if _, err := cacheclient.RedisClient.Expire(key, 0*time.Second).Result(); err != nil {
		logger.Error("redis Expire key: %s error: %v", key, err)
		return err
	}
	logger.Infof("delete redis key: %v ", key)

	// delete user create activity
	oper := &ActivityOper{}
	if err := oper.removeUserCreate(activityID, oid); err != nil {
		return err
	}

	// statistic redis do not remove know, because not record it
	/*
		key = fmt.Sprintf("%s:%s", common.ACTIVITY_STEPS_KEY_PREFIX, activityID)
		if _, err := cacheclient.RedisClient.Expire(key, 0*time.Second).Result(); err != nil {
			logger.Error("redis HDel key: %s error: %v", key, err)
			return err
		}
		logger.Infof("delete redis key: %v ", key)

		key = fmt.Sprintf("%s:%s", common.ACTIVITY_USER_RANK_KEY_PREFIX, activityID)
		if _, err := cacheclient.RedisClient.Expire(key, 0*time.Second).Result(); err != nil {
			logger.Error("redis HDel key: %s error: %v", key, err)
			return err
		}
		logger.Infof("delete redis key: %v ", key)

		key = fmt.Sprintf("%s:%s", common.ACTIVITY_TEAM_RANK_KEY_PREFIX, activityID)
		if _, err := cacheclient.RedisClient.Expire(key, 0*time.Second).Result(); err != nil {
			logger.Error("redis HDel key: %s error: %v", key, err)
			return err
		}
		logger.Infof("delete redis key: %v ", key)
	*/

	return nil
}

// 修改移动端的活动
func modifyActivityMobile(aid string, cfg *meta.Activity, newCfg *meta.Activity) error {
	//
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	oper := &ActivityOper{}
	before, _ := util.Pb2Json(cfg)

	// 可以修改活动名称、活动口号、活动头图
	foo := func(f, nf *string, enableDelete bool) {
		if enableDelete {
			if len(*nf) > 0 || *nf != *f {
				*f = *nf
			}
		} else {
			if len(*nf) > 0 {
				*f = *nf
			}
		}
	}
	foo(&cfg.Desc, &newCfg.Desc, false)
	foo(&cfg.Slogan, &newCfg.Slogan, false)
	foo(&cfg.OrgName, &newCfg.OrgName, false) // 可以修改团体名称、团体logo
	foo(&cfg.OrgHead, &newCfg.OrgHead, false)
	foo(&cfg.StartTime, &newCfg.StartTime, false) // 可以修改活动开始结束时间
	foo(&cfg.EndTime, &newCfg.EndTime, false)
	foo(&cfg.Bgpic, &newCfg.Bgpic, true) // TODO: 修改头图需要接入信安审核
	foo(&cfg.Color, &newCfg.Color, true)
	foo(&cfg.CompanyId, &newCfg.CompanyId, false)
	foo(&cfg.Cover, &newCfg.Cover, true)
	foo(&cfg.RelationDesc, &newCfg.RelationDesc, true)
	foo(&cfg.ForwardPic, &newCfg.ForwardPic, true)
	logger.Debug("modifyActivityMobile, cfg:%+v , newcfg:%+v", cfg, newCfg)

	/// 针对已结束活动又延长结束时间的情况  //状态变更为进行中
	logger.Debug("modifyActivityMobile, aid = %s, end_time = %s, start_time = %s,  Status = %d", aid, newCfg.EndTime, newCfg.StartTime, cfg.Status)
	if newCfg.EndTime > now && newCfg.StartTime < now && cfg.Status == meta.Activity_EXPIRED {
		newCfg.Status = meta.Activity_RUNNING
		cfg.Status = newCfg.Status
	}

	// 未支付可以修改地图id
	if cfg.Status == meta.Activity_DRAFT {
		if len(newCfg.RouteId) > 0 {
			cfg.RouteId = newCfg.RouteId
		}
	}
	// 移动端支持修改是否展示发起人信息
	cfg.ShowSponsor = newCfg.ShowSponsor

	// 更新redis和db
	if err := oper.syncActivityRedis(cfg); err != nil {
		return err
	}
	if err := oper.syncActivityDB(cfg); err != nil {
		return err
	}

	after, _ := util.Pb2Json(cfg)
	logger.Debug("aid = %s, before config = %s, after config = %s, mobile modify successfully", aid, before, after)
	return nil
}

// 修改运营平台的活动
func modifyActivityPlatform(aid string, cfg *meta.Activity, newCfg *meta.Activity) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	oper := &ActivityOper{}
	before, _ := util.Pb2Json(cfg) // 修改参数前的配置
	foo := func(f, nf *string, enableDelete bool) {
		if enableDelete {
			if len(*nf) > 0 || *nf != *f {
				*f = *nf
			}
		} else {
			if len(*nf) > 0 {
				*f = *nf
			}
		}
	}
	foo(&cfg.Desc, &newCfg.Desc, false)
	foo(&cfg.Slogan, &newCfg.Slogan, false)
	foo(&cfg.RouteId, &newCfg.RouteId, false)
	foo(&cfg.StartTime, &newCfg.StartTime, false)
	foo(&cfg.EndTime, &newCfg.EndTime, false)
	foo(&cfg.OrgName, &newCfg.OrgName, false)
	foo(&cfg.OrgHead, &newCfg.OrgHead, false)
	foo(&cfg.Bgpic, &newCfg.Bgpic, true)
	foo(&cfg.Color, &newCfg.Color, true)
	foo(&cfg.CompanyId, &newCfg.CompanyId, false)
	foo(&cfg.Cover, &newCfg.Cover, true)
	foo(&cfg.RelationDesc, &newCfg.RelationDesc, true)
	foo(&cfg.ForwardPic, &newCfg.ForwardPic, true)

	///
	if newCfg.SuppCoupons != cfg.SuppCoupons {
		cfg.SuppCoupons = newCfg.SuppCoupons
	}

	/// 针对已结束活动又延长结束时间的情况  //状态变更为进行中
	logger.Debug("modifyActivityPlatform, aid = %s, end_time = %s, start_time = %s,  Status = %d", aid, newCfg.EndTime, newCfg.StartTime, cfg.Status)
	if newCfg.EndTime > now && newCfg.StartTime < now && cfg.Status == meta.Activity_EXPIRED {
		newCfg.Status = meta.Activity_RUNNING
		cfg.Status = newCfg.Status
	}

	cfg.Rule = newCfg.Rule
	cfg.TeamMode = newCfg.TeamMode
	cfg.TeamOff = newCfg.TeamOff
	cfg.TeamMemberLimit = newCfg.TeamMemberLimit
	cfg.ShowSponsor = newCfg.ShowSponsor
	cfg.MatchOff = newCfg.MatchOff
	cfg.WhiteType = newCfg.WhiteType
	logger.Debug("%+v -- cfg:%+v , newcfg:%+v", util.GetCallee(), cfg, newCfg)

	// 更新redis和db
	if err := oper.syncActivityRedis(cfg); err != nil {
		return err
	}
	if err := oper.syncActivityDB(cfg); err != nil {
		return err
	}

	after, _ := util.Pb2Json(cfg) // 修改参数后的配置
	logger.Debug("aid = %s, before config = %s, after config = %s, platform modify successfully", aid, before, after)
	return nil
}

// UpdateActivity ...
func UpdateActivity(aid string, activity *meta.Activity, matchInfo *meta.MatchInfo, matchRule *meta.MatchRule, whiteList []string) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	// 获取活动信息
	oper := &ActivityOper{}
	info, err := oper.fetchActivity(aid)
	if err != nil {
		logger.Error("aid = %s fetchActivity err = %v", aid, err)
		return err
	}
	// 获取活动状态
	if err = oper.fillActivityStatus(info); err != nil {
		return err
	}

	logger.Debug("%v -- info:%+v,  activity:%+v", util.GetCallee(), info, activity)
	if info.Type == meta.Activity_MOBILE {
		// 移动端修改

		// 用户只能修改自己创建的活动
		// activity.ActivityCreator不为空，表明是C端用户修改，否则是运营平台修改
		if len(activity.ActivityCreator) != 0 && info.ActivityCreator != activity.ActivityCreator {
			logger.Error("aid = %s, origin creator = %s, change creator = %s, activity is not self can not modify",
				aid, info.ActivityCreator, activity.ActivityCreator)
			return errors.NewActivityMgrOpsNotSelfError(errors.WithMsg("aid = %s is not self can not to modify", aid))
		}

		// 如果未支付或者未开始, 开始时间和结束时间都能改
		if info.Status == meta.Activity_DRAFT || info.Status == meta.Activity_READY {
			logger.Debug("aid = %s, activity can modify time", aid)
		} else {
			// 活动已结束, 不能修改   /// 支持活动已结束，放宽结束时间
			/*
				if now > info.EndTime {
					logger.Error("aid = %s, end_time = %s, activity is finish can not modify", aid, info.EndTime)
					return errors.NewActivityMgrOpsOverTimeError(errors.WithMsg("aid = %s over time to modify", aid))
				}
			*/
			/// 结束时间可延长缩短，但不能晚于当天
			if activity.EndTime < util.GetLocalFormatTime() {
				logger.Error("aid = %s, now = %s, modi_end_time = %s,  activity end_time cann't earlier than now", aid, util.GetLocalFormatTime(), activity.EndTime)
				return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid = %s invalid time to modify", aid))
			}
			// 活动开始了, 开始时间不能改
			if now > info.StartTime &&
				len(activity.StartTime) > 0 &&
				activity.StartTime != info.StartTime {
				logger.Error("aid = %s, old start_time = %s, change start_time = %s activity is start can not modify start_time",
					aid, info.StartTime, activity.StartTime)
				return errors.NewActivityMgrOpsStartTimeError(errors.WithMsg("aid = %s already start can not modify start_time", aid))
			}
		}

		// 修改活动基础配置
		if err = modifyActivityMobile(aid, info, activity); err != nil {
			return err
		}
		// 修改配捐功能配置
		match, err := fundClient.GetMatchEvent(info.MatchId)
		if err != nil {
			logger.Error("eid = %s, fundClient.GetMatchEvent error: %v", info.MatchId, err)
			return err
		}
		if match.Header.Code != 0 {
			logger.Error("fundClient.GetMatchEvent error header: %v", match.Header)
			return err
		}
		// 只修改配捐捐步门槛, ruleType为-1表示不修改配捐规则
		originInfo := match.MatchEvent.MatchInfo
		originInfo.FStartTime = activity.StartTime
		originInfo.FEndTime = activity.EndTime
		originRule := match.MatchEvent.MatchRule
		originRule.RuleType = -1
		originRule.MatchQuota = matchRule.MatchQuota
		// 未支付的活动可以修改项目id和目标总金额
		if info.Status == meta.Activity_DRAFT {
			if matchInfo.FTargetFund > 0 {
				originInfo.FTargetFund = matchInfo.FTargetFund
			}
			if len(matchInfo.FPid) > 0 {
				originInfo.FPid = matchInfo.FPid
			}
		}
		rsp, err := fundClient.UpdateMatch(info.MatchId, originInfo, originRule)
		if err != nil {
			logger.Error("fundClient.UpdateMatch error: %v", err)
			return err
		}
		if rsp.Header.Code != 0 {
			logger.Error("fundClient.UpdateMatch error header: %v", rsp.Header)
			return err
		}
	} else if info.Type == meta.Activity_PLATFORM {
		// 运营平台修改

		// 活动已结束, 不能修改   /// 支持活动已结束，放宽结束时间
		/*
			if now > info.EndTime {
				logger.Error("aid = %s, end_time = %s, activity is finish can not modify", aid, info.EndTime)
				return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid = %s over time to modify", aid))
			}
		*/
		/// 结束时间可延长缩短，但不能晚于当天
		if activity.EndTime < util.GetLocalFormatTime() {
			logger.Error("aid = %s, now = %s, modi_end_time = %s,  activity end_time cann't earlier than now", aid, util.GetLocalFormatTime(), activity.EndTime)
			return errors.NewActivityMgrOpsForbidError(errors.WithMsg("aid = %s invalid time to modify", aid))
		}

		if err = modifyActivityPlatform(aid, info, activity); err != nil {
			return err
		}

		// 修改白名单列表，先清空，再存入db和redis
		if activity.WhiteType == meta.Activity_WhiteList {
			err = DelActivityWhiteList(aid)
			if err != nil {
				logger.Error("DelActivityWhiteList error: %v", err)
				return err
			}
			if len(whiteList) > 0 {
				err = SetActivityWhiteList(aid, whiteList)
				if err != nil {
					logger.Error("SetActivityWhiteList error: %v", err)
					return err
				}
			}
		}

		// change quota
		a, err := oper.fetchActivity(aid)
		if err != nil {
			logger.Error("get activityID: %v error: %v", aid, err)
			return err
		}
		rsp, err := fundClient.UpdateMatch(a.MatchId, matchInfo, matchRule)
		if err != nil {
			logger.Error("fundClient.UpdateMatch error: %v", err)
			return err
		}
		if rsp.Header.Code != 0 {
			logger.Error("fundClient.UpdateMatch error header: %v", rsp.Header)
			return err
		}
	}

	return nil
}

// UpdateActivityState ...
func UpdateActivityState(activityID string, state meta.Activity_Status) error {
	if err := UpdateActivityStatus(activityID, state); err != nil {
		logger.Error("aid: %s, status: %v, error: %v", activityID, state, err)
		return err
	}
	return nil
}


func GetActivitySuccessNum() (int, error) {
	numStr, err := cacheclient.RedisClient.Get(common.ACTIVITY_SUCCESS_NUM_KEY_PREFIX).Result()
	if err != nil {
		logger.Error("Set err = %v", err)
		return 0, err
	}
	num, err := strconv.Atoi(numStr)
	if err != nil {
		logger.Error("Atoi err = %v", err)
		return 0, err
	}
	return num, nil
}

func GetActivityBlacklist(aid, oid string) (bool, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_BLACKLIST_KEY_PREFIX, aid)
	val, err := cacheclient.RedisClient.ZScore(key, oid).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		logger.Error("ZScore err = %v", err)
		return false, err
	}
	if val == 1 {
		return true, nil
	}
	return false, nil
}

func SetActivityWhiteList(aid string, whiteList []string) error {
	err := SetActivityWhiteListDB(aid, whiteList)
	if err != nil {
		logger.Error("SetActivityWhiteListDB err = %v", err)
		return err
	}
	err = SetActivityWhiteListRedis(aid, whiteList)
	if err != nil {
		logger.Error("SetActivityWhiteListRedis err = %v", err)
		return err
	}
	return nil
}

func SetActivityWhiteListDB(aid string, whiteList []string) error {
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "INSERT INTO t_white_list (f_activity_id, f_phone_num, f_status, f_create_time, f_modify_time) " +
		" VALUES "

	args := make([]interface{}, 0)

	for _, v := range whiteList {
		sql += "(?, ?, ?, ?, ?), "
		args = append(args, []interface{}{aid, v, 0, now, now}...)
	}
	sql = strings.TrimRight(sql, ", ")

	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("aid = %s, sql = %s, update DB record err = %v", aid, sql, err)
		return err
	}

	logger.Debug("aid = %s, whiteList = %v, update white list DB record successfully", aid, whiteList)
	return nil
}

func SetActivityWhiteListRedis(aid string, whiteList []string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_WHITE_LIST_KEY_PREFIX, aid)

	var fields = make(map[string]interface{})
	for _, v := range whiteList {
		fields[v] = 1
	}

	if _, err := cacheclient.RedisClient.HMSet(key, fields).Result(); err != nil {
		logger.Error("key = %s, whiteList = %v, redis hset err = %v", key, whiteList, err)
		return err
	}

	logger.Debug("key = %s, whiteList = %v, sync to redis successfully", key, whiteList)
	return nil
}

func IsActivityWhiteList(aid, phoneNum string) (bool, error) {
	actWhite, err := GetActivityWhiteListRedis(aid, phoneNum)
	if err != nil {
		logger.Error("GetActivityWhiteListRedis err = %v", err)
		actWhite, err := GetActivityWhiteListDB(aid, phoneNum)
		if err != nil {
			logger.Error("GetActivityWhiteListDB err = %v", err)
			return false, err
		}
		if len(actWhite.ActivityId) > 0 {
			SetActivityWhiteListRedis(aid, []string{phoneNum})
			return true, nil
		}
		return false, err
	}
	if len(actWhite.ActivityId) > 0 {
		return true, nil
	}
	return false, nil
}

func GetActivityWhiteListDB(aid, phoneNum string) (*meta.ActivityWhiteList, error) {
	sql := "select f_activity_id, f_phone_num,  f_status, f_create_time from t_white_list where f_activity_id = ? and f_phone_num = ? "
	args := make([]interface{}, 0)
	args = append(args, aid, phoneNum)

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("aid = %s, QueryDB error, err = %v", aid, err)
		return nil, errors.NewDBClientError(errors.WithMsg("aid = %s, QueryDB error, err = %v", aid, err))
	}

	var actWhite = &meta.ActivityWhiteList{}
	if len(res) > 0 {
		actWhite.ActivityId = res[0]["f_activity_id"]
		actWhite.PhoneNum = res[0]["f_phone_num"]
		status, _ := strconv.Atoi(res[0]["f_status"])
		actWhite.Status = int32(status)
		actWhite.CreateTime = res[0]["f_create_time"]
	}

	return actWhite, nil
}

func GetActivityWhiteListRedis(aid, phoneNum string) (*meta.ActivityWhiteList, error) {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_WHITE_LIST_KEY_PREFIX, aid)

	val, err := cacheclient.RedisClient.HGet(key, phoneNum).Result()
	if err != nil {
		if err != redis.Nil {
			logger.Error("key = %s, phoneNum = %v, redis hset err = %v", key, phoneNum, err)
		}
		return nil, err
	}

	status, err := strconv.Atoi(val)
	if err != nil {
		logger.Error("key = %s, phoneNum = %v, Atoi err = %v", key, phoneNum, err)
		return nil, err
	}

	return &meta.ActivityWhiteList{
		ActivityId: aid,
		PhoneNum:   phoneNum,
		Status:     int32(status),
	}, nil
}

func DelActivityWhiteList(aid string) error {
	if err := DelActivityWhiteListDB(aid); err != nil {
		logger.Error("DelActivityWhiteListDB err = %v", err)
		return err
	}
	if err := DelActivityWhiteListRedis(aid); err != nil {
		logger.Error("DelActivityWhiteListRedis err = %v", err)
		return err
	}
	return nil
}

func DelActivityWhiteListDB(aid string) error {
	sql := "delete from t_white_list where f_activity_id = ?"
	args := make([]interface{}, 0)
	args = append(args, aid)

	_, err := cdbClient.ExecSQL(sql, args)
	if err != nil {
		logger.Error("aid = %s, ExecSQL err = %v", aid, err)
		return err
	}
	return nil
}

func DelActivityWhiteListRedis(aid string) error {
	key := fmt.Sprintf("%s:%s", common.ACTIVITY_WHITE_LIST_KEY_PREFIX, aid)

	_, err := cacheclient.RedisClient.Del(key).Result()
	if err != nil {
		logger.Error("key = %s, redis hset err = %v", key, err)
		return err
	}
	return nil
}

func listActivityWhiteListDB(aids []string) ([]*meta.ActivityWhiteList, error) {
	sql := "select f_activity_id, f_phone_num,  f_status, f_create_time from t_white_list where f_activity_id in("
	args := make([]interface{}, 0)

	for _, aid := range aids {
		sql += "?, "
		args = append(args, aid)
	}
	sql = strings.TrimRight(sql, ", ")
	sql += ")"

	res, err := cdbClient.Query(sql, args)
	if err != nil {
		logger.Error("aids = %v, QueryDB error, err = %v", aids, err)
		return nil, errors.NewDBClientError(errors.WithMsg("aids = %v, QueryDB error, err = %v", aids, err))
	}

	var list []*meta.ActivityWhiteList
	for _, v := range res {
		actWhite := &meta.ActivityWhiteList{}
		actWhite.ActivityId = v["f_activity_id"]
		actWhite.PhoneNum = v["f_phone_num"]
		status, _ := strconv.Atoi(v["f_status"])
		actWhite.Status = int32(status)
		actWhite.CreateTime = v["f_create_time"]
		list = append(list, actWhite)
	}

	return list, nil
}
