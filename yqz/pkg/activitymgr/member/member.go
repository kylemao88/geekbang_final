package member

import (
	"time"

	meta "git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/common"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"github.com/go-redis/redis"
)

const (
	TIME_OUT             = 24 * time.Hour * 365
	MEMBER_RECOVER_QUEUE = "yqz:{activitymgr:member:recover}"
)

var (
	cdbClient dbclient.DBClient
	fl        *common.Fill
)

func InitComponent(db dbclient.DBClient) error {
	cdbClient = db
	fl = common.Create(MEMBER_RECOVER_QUEUE, 2, fillHandler)
	return nil
}

func GetActivityMemberSizeDBNew(oid string) (int, error) {
	if len(oid) == 0 {
		return 0, errors.NewError(errors.ParamInvalid)
	}
	a := ActivityMember{}

	size, err := a.queryMemberSizeDBNew(oid)
	if err != nil {
		logger.Error("GetActivityMemberSizeDBNew error, err = %v, oid = %s", err, oid)
		return 0, err
	}

	return size, nil
}

func GetActivityMemberSize(oid string) (int, error) {
	if len(oid) == 0 {
		return 0, errors.NewError(errors.ParamInvalid)
	}
	a := ActivityMember{}

	size, err := a.queryMemberSizeRedis(oid)
	if err != nil {
		if err == redis.Nil {
			ret, err := fl.PushFill("activityMemberFill", oid)
			if err != nil || ret == 0 {
				logger.Error("PushFill error: %v", oid)
			}
			return 0, errors.NewOpsNeedRetryError(errors.WithMsg("oid:%v recover now", oid))
		}
		logger.Error("queryMemberListRedis error, err = %v, oid = %v", err, oid)
		return 0, err
	}

	return size, nil
}

func GetActivityMember(oid string, offset, count int) ([]*meta.ActivityMember, error) {
	if len(oid) == 0 || offset < 0 || count < 0 {
		return nil, errors.NewError(errors.ParamInvalid)
	}
	a := ActivityMember{}
	list, err := a.queryMemberListRedis(oid, offset, count)
	if err != nil {
		if err == redis.Nil {
			ret, err := fl.PushFill("activityMemberFill", oid)
			if err != nil || ret == 0 {
				logger.Error("PushFill error: %v", oid)
			}
			return nil, errors.NewOpsNeedRetryError(errors.WithMsg("oid:%v recover now", oid))
		}
		logger.Error("queryMemberListRedis error, err = %v, oid = %v", err, oid)
		return nil, err
	}
	return list, nil
}

func SetActivityMember(actMember *meta.ActivityMember) error {
	a := ActivityMember{}

	am, err := a.queryMemberDB(actMember.ActivityId, actMember.UserId)
	if err != nil {
		logger.Error("queryMemberDB error, err = %v, actMember = %+v", err, actMember)
		return err
	}
	if am != nil && am.Status == actMember.Status {
		logger.Info("repeat set, actMember = %+v", actMember)
		return nil
	}

	if err = a.syncDB(actMember); err != nil {
		logger.Error("syncDB error, err = %v, actMember = %+v", err, actMember)
		return err
	}

	if actMember.Status == NOT_IN_ACTIVITY || actMember.Status == BLACKLIST {
		if err = a.deleteRedis(actMember); err != nil {
			logger.Error("deleteRedis error, err = %v, actMember = %+v", err, actMember)
			return err
		}
		return nil
	}

	if err = a.syncRedis(actMember); err != nil {
		logger.Error("syncRedis error, err = %v, actMember = %+v", err, actMember)
		return err
	}
	return nil
}

// 批量更新到达时间
func BatchUpdateActivityArrive(aid string, oids []string) error {
	if len(aid) == 0 || len(oids) == 0 {
		logger.Error("oids = %v or aid = %s is empty error", oids, aid)
		return errors.NewError(errors.ParamInvalid)
	}

	a := ActivityMember{}
	actMembers, err := a.queryUsersMemberListDB(aid, oids)
	if err != nil {
		logger.Error("queryUsersMemberListDB error, err = %v, aid = %s, oids = %v", err, aid, oids)
		return err
	}

	logger.Debug("actMembers = %v", actMembers)

	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	for _, actMember := range actMembers {
		logger.Debug("aid = %v, oid = %v, arr = %v", aid, actMember.UserId, actMember.ArriveTime)
		if len(actMember.ArriveTime) > 0 {
			continue
		}
		actMember.ArriveTime = now

		if err = a.syncDB(actMember); err != nil {
			logger.Error("syncDB error, err = %v, actMember = %+v", err, actMember)
			return err
		}

		if err = a.syncRedis(actMember); err != nil {
			logger.Error("syncRedis error, err = %v, actMember = %+v", err, actMember)
			return err
		}
	}

	return nil
}

// team
func GetTeamMemberSize(oid, aid string) (int, error) {
	if 0 == len(oid) || 0 == len(aid) {
		logger.Error("oid = %s or aid = %s is empty error", oid, aid)
		return 0, errors.NewError(errors.ParamInvalid)
	}

	t := TeamMember{}

	return t.queryMemberSizeDB(aid, oid)
}

func GetTeamMember(oid, aid string, offset, count int) ([]*meta.TeamMember, error) {
	if len(oid) == 0 || offset < 0 || count < 0 {
		return nil, errors.NewError(errors.ParamInvalid)
	}
	t := TeamMember{}
	list, err := t.queryMemberListRedis(aid, oid, offset, count)
	if err != nil {
		if err == redis.Nil {
			ret, err := fl.PushFill("teamMemberFill", aid, oid)
			if err != nil || ret == 0 {
				logger.Error("PushFill error: %v, %v", aid, oid)
			}
			return nil, errors.NewOpsNeedRetryError(errors.WithMsg("aid:%v,oid:%v recover now", aid, oid))
		}
		logger.Error("queryMemberListRedis error, err = %v, oid = %v, aid = %v", err, oid, aid)
		return nil, err
	}
	return list, nil
}

func SetTeamMember(teamMember *meta.TeamMember) error {
	t := TeamMember{}

	tm, err := t.queryMemberDB(teamMember.ActivityId, teamMember.UserId, teamMember.TeamId)
	if err != nil {
		logger.Error("queryMemberDB error, err = %v, teamMember = %+v", err, teamMember)
		return err
	}
	if tm != nil && tm.InTeam == teamMember.InTeam {
		logger.Error("repeat set, teamMember = %+v", teamMember)
		return nil
	}

	// todo: 根据配置，过滤加入多个小队
	if teamMember.InTeam == IN_TEAM {
		size, err := t.queryMemberSizeDB(teamMember.ActivityId, teamMember.UserId)
		if err != nil {
			logger.Error("queryMemberSizeDB error, err = %v, teamMember = %+v", err, teamMember)
			return err
		}
		if size > 0 {
			err = errors.NewActivityMgrJoinTeamLimitError()
			logger.Error("join limit error, err = %v, teamMember = %+v", err, teamMember)
			return err
		}
	}

	err = t.syncDB(teamMember)
	if err != nil {
		logger.Error("syncDB error, err = %v, teamMember = %+v", err, teamMember)
		return err
	}

	if teamMember.InTeam == NOT_IN_TEAM {
		err = t.deleteRedis(teamMember)
		if err != nil {
			logger.Error("deleteRedis error, err = %v, teamMember = %+v", err, teamMember)
			return err
		}
		return nil
	}

	err = t.syncRedis(teamMember)
	if err != nil {
		logger.Error("syncRedis error, err = %v, teamMember = %+v", err, teamMember)
		return err
	}
	return nil
}

func IsJoinTeam(aid, tid, oid string) (bool, error) {
	t := TeamMember{}

	tm, err := t.queryMemberRedis(aid, tid, oid)
	if err != nil {
		if err == redis.Nil {
			ret, err := fl.PushFill("teamMemberFill", aid, oid)
			if err != nil || ret == 0 {
				logger.Error("PushFill error: %v, %v", aid, oid)
			}
			return false, errors.NewOpsNeedRetryError(errors.WithMsg("aid:%v,oid:%v recover now", aid, oid))
		}
		logger.Error("queryMemberRedis error, err = %v", err)
		return false, err
	}
	if tm == nil {
		return false, nil
	}
	return true, nil

	//// todo: 先走db，redis考虑不周
	//tm, err := t.queryMemberDB(aid, tid, oid)
	//if err != nil {
	//	logger.Error("queryMemberRedis error, err = %v", err)
	//	return false, err
	//}
	//if tm == nil {
	//	return false, nil
	//}
	//if tm.InTeam == NOT_IN_TEAM {
	//	return false, nil
	//}
	//return true, nil
}

func GetTeamsByUsers(aid string, oids []string) (map[string][]*meta.TeamMember, error) {
	var list []*meta.TeamMember
	t := TeamMember{}
	for _, oid := range oids {
		clist, err := t.queryMemberListRedis(aid, oid, 0, 10)
		if err != nil {
			if err == redis.Nil {
				ret, err := fl.PushFill("teamMemberFill", aid, oid)
				if err != nil || ret == 0 {
					logger.Error("PushFill error: %v, %v", aid, oid)
				}
				return nil, errors.NewOpsNeedRetryError(errors.WithMsg("aid:%v,oid:%v recover now", aid, oid))
			}
			logger.Error("queryMemberListRedis error, err = %v, oid = %v, aid = %v", err, oid, aid)
			return nil, err
		}
		list = append(list, clist...)
	}

	//list, err := t.queryTeamsByUserDB(aid, oids)
	//if err != nil {
	//	logger.Error("queryTeamsByUserDB error, err = %v", err)
	//	return nil, err
	//}

	var mTeams = make(map[string][]*meta.TeamMember)
	for _, v := range list {
		mTeams[v.UserId] = append(mTeams[v.UserId], &meta.TeamMember{
			ActivityId: v.ActivityId,
			TeamId:     v.TeamId,
			UserId:     v.UserId,
			InTeam:     v.InTeam,
			Status:     v.Status,
			CreateTime: v.CreateTime,
		})
	}

	return mTeams, nil
}

// 用户加入活动, 会将用户拉入t_user表, 表示一块走的用户
func SetUserDB(oid, appid, unionId, phoneNum string) error {
	// 先不影响主流程
	if len(appid) == 0 || len(unionId) == 0 || len(oid) == 0 {
		logger.Debug("oid = %s, appid = %s, unionId = %s is empty continue", oid, appid, unionId)
		return nil
	}

	joinEvent := 0
	backendUpdate := 1
	status := 0
	args := make([]interface{}, 0)
	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	sql := "INSERT INTO t_user (f_user_id, f_appid, f_unin_id, f_join_event, f_backend_update, f_phone_num, f_status, f_sub_time, f_create_time, f_modify_time) " +
		" VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) " +
		" ON DUPLICATE KEY UPDATE f_modify_time=?, f_phone_num=?"
	args = append(args, oid, appid, unionId, joinEvent, backendUpdate, phoneNum, status, now, now, now)
	args = append(args, now, phoneNum)

	if _, err := cdbClient.ExecSQL(sql, args); err != nil {
		logger.Error("sql=%s, update DB record err = %v error", sql, err)
		return err
	}

	logger.Debug("oid = %s, appid = %s, unionId = %s, update t_user record successfully", oid, appid, unionId)
	return nil
}
