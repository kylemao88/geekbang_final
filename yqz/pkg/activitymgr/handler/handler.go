package handler

import (
	"crypto/md5"
	"fmt"
	"github.com/go-redis/redis"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"time"

	gms "git.code.oa.com/gongyi/gomore/service"
	pb "git.code.oa.com/gongyi/yqz/api/activitymgr"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	meta "git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/activity"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/comment"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/common"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/member"
	steps2 "git.code.oa.com/gongyi/yqz/pkg/activitymgr/steps"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/team"
	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/supplement"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	fundClient "git.code.oa.com/gongyi/yqz/pkg/fundmgr/client"
	"github.com/martinlindhe/base36"
	"google.golang.org/protobuf/proto"
)

const CKV_QUERY_MAX_SIZE = 10 // ckv每次批量最多查询10条记录

func CreateActivity(ctx *gms.Context) {
	defer util.GetUsedTime("CreateActivity")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.CreateActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.CreateActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.CreateActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 先将用户拉入user表, 保证拉取头像昵称正常
	if req.Activity.Type == metadata.Activity_MOBILE {
		if err = member.SetUserDB(req.Oid, req.Appid, req.UnionId, ""); err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("member.SetUserDB error"))
			return
		}
	}

	// 用户创建活动数量限制
	if req.Activity.Type == metadata.Activity_MOBILE {
		// 用户创建活动的db数据在 t_activity_info 表中
		size, err := activity.QueryUserCreateSize(req.Activity.ActivityCreator)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("QueryUserCreateSize error"))
			return
		}
		createLimit := config.GetConfig().ActivityConf.CreateLimit
		if createLimit <= 0 {
			createLimit = config.CreateLimitDefault
		}
		// 用户创建的活动超过上限
		if size >= int64(createLimit) {
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ActivityMgrCreateLimitError,
					Msg:  "user create activity over limit",
				},
			}
			logger.Debug("%v - oid = %s create activity over limit", util.GetCallee(), req.Activity.ActivityCreator)
			return
		}
	}

	// 1.创建活动
	ca, err := activity.CreateActivity(req.Activity, req.MatchInfo, req.MatchRule)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.CreateActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("create activity error"))
		return
	}

	// 2.创建默认小队（可选）
	for _, name := range req.Activity.DefaultTeams {
		t := &metadata.Team{
			ActivityId:  ca.Activity.ActivityId,
			TeamType:    1,       // 1 表示默认小队 2 表示私密小队，非队员不可围观
			TeamDesc:    name,    // 小队名称
			TeamCreator: "admin", // admin表示系统创建的小队
			TeamFlag:    metadata.Team_SYS_TEAM,
		}
		// 运营平台可以创建多个默认小队
		sysTeam, err := team.CreateTeam(t)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("create default team error"))
			return
		}
		// 更新活动的小队步数排名
		if err = activity.UpdateTeamRank(sysTeam.ActivityId, sysTeam.TeamId); err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("update activity team fund rank error"))
			return
		}

		// 更新活动的小队公益金排名
		if err = activity.InitTeamFundRank(sysTeam.ActivityId, sysTeam.TeamId); err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("update activity team rank error"))
			return
		}
	}

	// 3.用户创建的活动
	if metadata.Activity_MOBILE == req.Activity.Type {
		// 查询用户的活动步数
		startDate := util.Datetime2Date(ca.Activity.StartTime)
		endDate := util.Datetime2Date(ca.Activity.EndTime)
		steps, err := steps2.UpdateUserSteps(ca.Activity.ActivityId, ca.Activity.ActivityCreator, startDate, endDate)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity user steps error"))
			return
		}

		// 自动将用户加入活动 TODO: 补充appid
		if err = joinActivity(ca.Activity.ActivityId, ca.Activity.ActivityCreator, steps); err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("join activity user create error"))
			return
		}

		// 小队给空数据(预防加入后，拉取数据时触发回补)
		if err := member.SetTeamMember(&metadata.TeamMember{
			ActivityId: ca.Activity.ActivityId,
			TeamId:     "NULL",
			UserId:     ca.Activity.ActivityCreator,
			InTeam:     0,
			Status:     0,
			CreateTime: "",
		}); err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("create team null data error"))
			return
		}

		// 更新用户的创建活动列表
		if err = activity.UpdateUserCreate(ca.Activity.ActivityId, ca.Activity.ActivityCreator); err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("update activity user create error"))
			return
		}

		// 更新has_create字段为true
		if err = member.SetProfileHasCreate(ca.Activity.ActivityCreator, true); err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
	}

	// 有白名单列表，存入db和redis
	if ca.Activity.WhiteType == metadata.Activity_WhiteList {
		err = activity.SetActivityWhiteList(req.Activity.ActivityId, req.WhiteList)
		if err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.CreateActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
	}

	// add default comment for
	err = comment.SetDefaultComment(ca.Activity.ActivityId)
	if err != nil {
		// we just keep running, recover func will recover system comment
		logger.Error("SetDefaultComment activity: %v error: %v", ca.Activity.ActivityId, err)
	}

	// response ...
	rsp = &pb.CreateActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Activity: ca,
		Status:   pb.CreateActivityResponse_SUCCESS,
	}
}

func QueryActivity(ctx *gms.Context) {
	defer util.GetUsedTime("QueryActivity")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	act, err := activity.QueryActivity(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
		return
	}

	// response ...
	rsp = &pb.QueryActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Activity: act,
		Status:   pb.QueryActivityResponse_SUCCESS,
	}
}

func QueryTeams(ctx *gms.Context) {
	defer util.GetUsedTime("QueryTeams")()
	// default pack rsp
	//rsp := &pb.QueryTeamsResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryTeamsResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryTeamsRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryTeamsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	teams, err := team.QueryTeams(req.Aid, req.TeamIds)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryTeamsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query teams error"))
		return
	}

	var isWatch = make(map[string]bool)
	for _, v := range teams {
		if v.TeamType == 1 {
			isWatch[v.TeamId] = true
			continue
		}
		ok, _, err := team.QueryTeamUserRank(v.TeamId, req.Oid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryTeamsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("QueryTeamUserRank error"))
			return
		}
		if ok == true {
			isWatch[v.TeamId] = true
			continue
		}
		isWatch[v.TeamId] = false
	}

	// response ...
	rsp = &pb.QueryTeamsResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Teams:   teams,
		IsWatch: isWatch,
		Status:  pb.QueryTeamsResponse_SUCCESS,
	}
}

func JoinActivity(ctx *gms.Context) {
	defer util.GetUsedTime("JoinActivity")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.JoinActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	t1 := time.Now()
	req := &pb.JoinActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)
	logger.Info("t1.GetUsedTime - (%s)", time.Since(t1))

	// 检查用户加入活动是否超过上限
	t2 := time.Now()
	if err = checkUserJoinOverLimit(req.Oid); err != nil {
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		// 用户加入活动超过上限, 这是已知错误, 不告警, 前端判断错误码
		if errors.IsActivityMgrOverMaxJoinLimitError(err) {
			logger.Info("oid = %s, aid = %s, join activity over limit", req.Oid, req.Aid)
		} else { // 待确定错误, 告警排查日志
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			alarm.CallAlarmFunc(fmt.Sprintf("checkUserJoinOverLimit error"))
		}
		return
	}
	logger.Info("t2.GetUsedTime - (%s)", time.Since(t2))

	// 检查用户加入活动是否黑名单
	t3 := time.Now()
	isBlack, err := activity.GetActivityBlacklist(req.Aid, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	if isBlack == true {
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.NoPermission,
				Msg:  "no permission",
			},
		}
		return
	}

	act, err := activity.QueryActivity(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
		return
	}
	if act == nil || act.Activity == nil || len(act.Activity.ActivityId) == 0 {
		// response ...
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Msg:  "Param Invalid",
				Code: errors.ParamInvalid,
			},
		}
		return
	}
	logger.Info("t3.GetUsedTime - (%s)", time.Since(t3))

	// 白名单验证手机号码
	if act.Activity.WhiteType == meta.Activity_WhiteList {
		isWhite, err := activity.IsActivityWhiteList(req.Aid, req.PhoneNum)
		if err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.JoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
		if isWhite == false {
			rsp = &pb.JoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.NoPermission,
					Msg:  "no permission",
				},
			}
			return
		}
		// 白名单存入手机号码
		err = member.SetProfilePhoneNum(req.Oid, req.PhoneNum)
		if err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.JoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
	}

	// 先将用户拉入user表, 保证拉取头像昵称正常
	t4 := time.Now()
	if err = member.SetUserDB(req.Oid, req.Appid, req.UniId, req.PhoneNum); err != nil {
		logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("member.SetUserDB error"))
		return
	}
	logger.Info("t4.GetUsedTime - (%s)", time.Since(t4))

	// 查询用户的活动步数
	t5 := time.Now()
	startDate := util.Datetime2Date(act.Activity.StartTime)
	endDate := util.Datetime2Date(act.Activity.EndTime)
	steps, err := steps2.UpdateUserSteps(req.Aid, req.Oid, startDate, endDate)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity user steps error"))
		return
	}
	logger.Info("t5.GetUsedTime - (%s)", time.Since(t5))

	// 用户加入活动
	t6 := time.Now()
	if err = joinActivity(req.Aid, req.Oid, steps); err != nil {
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		// 用户已经加入了活动, 这是已知错误, 不告警, 前端判断错误码
		if errors.IsActivityMgrRepeatOperError(err) {
			logger.Info("oid = %s, aid = %s, join activity repeat", req.Oid, req.Aid)
		} else { // 待确定错误, 告警排查日志
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			alarm.CallAlarmFunc(fmt.Sprintf("update activity user rank error"))
		}
		return
	}
	logger.Info("t6.GetUsedTime - (%s)", time.Since(t6))

	// 小队给空数据(预防加入后，拉取数据时触发回补)
	t7 := time.Now()
	if err := member.SetTeamMember(&metadata.TeamMember{
		ActivityId: req.Aid,
		TeamId:     "NULL",
		UserId:     req.Oid,
		InTeam:     0,
		Status:     0,
		CreateTime: "",
	}); err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.JoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("create team null data error"))
		return
	}
	logger.Info("t7.GetUsedTime - (%s)", time.Since(t7))

	// response ...
	rsp = &pb.JoinActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}

func JoinTeam(ctx *gms.Context) {
	defer util.GetUsedTime("JoinTeam")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.JoinTeamResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.JoinTeamRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 检查活动是否是自动分配小队活动
	act, err := activity.QueryActivity(req.Aid)
	if err != nil {
		logger.Error("query act=%d err : %v", req.Aid, err)
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
		return
	}
	/// 不需要加入小队，直接返回
	if len(req.Tid) == 0 && act.Activity.TeamMode != 3 { //TeamMode == 3 表示自动分配小队
		// response ...
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Msg:  "success",
				Code: errors.Success,
			},
		}
		return
	}

	// 检查用户加入活动是否黑名单
	isBlack, err := activity.GetActivityBlacklist(req.Aid, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	if isBlack == true {
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.NoPermission,
				Msg:  "no permission",
			},
		}
		return
	}

	var userJoinTeam *meta.Team
	if req.Join && act.Activity.TeamMode == 3 { //自动加入分配小队
		/// 获取活动中人数最少的小队
		userJoinTeam, err = team.QueryMinMemberTeam(req.Aid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query min members team error"))
			return
		}
	} else { // 指定加入小队  //或推出小队
		// 检查小队是否存在
		teams, err := team.QueryTeams(req.Aid, []string{req.Tid})
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query team error"))
			return
		}
		if len(teams) == 0 {
			// response ...
			logger.Error("aid = %s teams is empty error", req.Aid)
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Msg:  "Param Invalid",
					Code: errors.ParamInvalid,
				},
			}
			return
		}
		userJoinTeam = teams[0]
	}
	logger.Debug("%v - traceID: %v aid:%s uid:%s join team:%+v", util.GetCallee(), ctx.Request.TraceId, req.Aid, req.Oid, userJoinTeam)

	// 同一个活动, 用户已经加入一个小队了, 不能再加入小队了, 除非先退出原来的小队, 才能继续加入小队
	if req.Join {
		joinTeamSize, err := member.GetTeamMemberSize(req.Oid, req.Aid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("GetTeamMemberSize error"))
			return
		}
		if joinTeamSize > 0 {
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ActivityMgrJoinTeamLimitError,
					Msg:  "user join team limit error",
				},
			}
			// 这是已知错误, 不告警, 前端判断错误码
			logger.Info("aid = %s, oid = %s, join_team_size = %d, reach limit", req.Aid, req.Oid, joinTeamSize)
			return
		}
	}

	// 用户加入或退出小队
	if err = joinTeam(req.Aid, userJoinTeam.TeamId, req.Oid, req.PhoneNum, req.Join, userJoinTeam, act); err != nil {
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		if errors.IsActivityMgrOverTeamMemberLimitError(err) { // 用户加入小队超过小队人数上限, 这是已知错误, 不告警, 前端判断错误码
			logger.Info("oid = %s, aid = %s, join team over team_member limit", req.Oid, req.Aid)
		} else if errors.IsActivityMgrOverMaxJoinLimitError(err) { // 用户加入活动超过上限, 这是已知错误, 不告警, 前端判断错误码
			logger.Info("oid = %s, aid = %s, join activity over limit", req.Oid, req.Aid)
		} else if errors.IsNoPermission(err) { // 用户无权限加入
			logger.Info("oid = %s, aid = %s, join activity no permission", req.Oid, req.Aid)
			rsp.Header.Code = errors.NoPermission
			rsp.Header.Msg = "join activity no permission"
		} else { // 待确定错误, 告警排查日志
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			alarm.CallAlarmFunc(fmt.Sprintf("join team error"))
		}
		return
	}

	// 先将用户拉入user表, 保证拉取头像昵称正常
	if err = member.SetUserDB(req.Oid, req.Appid, req.UniId, req.PhoneNum); err != nil {
		logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
		rsp = &pb.JoinTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("member.SetUserDB error"))
		return
	}
	// 白名单存入手机号码
	if len(req.PhoneNum) > 0 {
		err = member.SetProfilePhoneNum(req.Oid, req.PhoneNum)
		if err != nil {
			logger.Error("%v - traceID: %v err: %v", util.GetCallee(), ctx.Request.TraceId, err)
			rsp = &pb.JoinTeamResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
	}

	// response ...
	rsp = &pb.JoinTeamResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		TeamId:   userJoinTeam.TeamId,
		TeamName: userJoinTeam.TeamDesc,
	}
}

func ActivityTeamRank(ctx *gms.Context) {
	defer util.GetUsedTime("ActivityTeamRank")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.ActivityTeamRankResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.ActivityTeamRankRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	act, err := activity.QueryActivity(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
		return
	}
	if act == nil || act.Activity == nil || len(act.Activity.ActivityId) == 0 {
		// response ...
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Msg:  "Param Invalid",
				Code: errors.ParamInvalid,
			},
		}
		return
	}

	// 查询活动统计
	stats, err := activity.QueryActivityStats(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity stats error"))
		return
	}

	// 本人小队步数排名 or 公益金排名
	teamInfo, err := queryUserJoinTeam(req.Aid, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v queryUserJoinTeam error: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("queryUserJoinTeam error"))
		return
	}

	//
	var teamRank int64 = 0
	if req.RankType == common.RAND_TYPE_FUND {
		_, teamRank, err = activity.QueryActivityTeamFundRank(req.Aid, teamInfo.TeamId)
	} else { // req.RankType == common.RAND_TYPE_STEP
		_, teamRank, err = activity.QueryActivityTeamStepRank(req.Aid, teamInfo.TeamId)
	}
	if err != nil {
		logger.Error("%v - traceID: %v, query user-teamid[%s-%s] %s rank err: %s", util.GetCallee(), ctx.Request.TraceId, req.Oid, teamInfo.TeamId, req.RankType, err.Error())
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query user-team self rank error"))
		return
	}
	logger.Error("%v - traceID: %v, query self user-team[%s-%s] %s rank succ:%d", util.GetCallee(), ctx.Request.TraceId, req.Oid, teamInfo.TeamId, req.RankType, teamRank)

	//
	list := make([]*pb.ActivityTeamRank, 0)

	// 查询小队排名
	zs, err := activity.QueryTeamRank(req.Aid, req.Page, req.Size, req.RankType)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityTeamRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query team rank error"))
		return
	}

	if len(zs) > 0 {
		tids := make([]string, 0)
		for _, z := range zs {
			if z.Member == nil {
				continue
			}
			tid := z.Member.(string)
			tids = append(tids, tid)
		}
		// 查询小队信息
		teams, err := team.QueryTeams(req.Aid, tids)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.ActivityTeamRankResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query teams error"))
			return
		}

		mTeam := make(map[string]*metadata.Team)
		for _, v := range teams {
			mTeam[v.TeamId] = v
		}

		var start = req.Page*req.Size + 1
		for i, z := range zs {
			if z.Member == nil {
				continue
			}
			tid := z.Member.(string)
			t, ok := mTeam[tid]
			// 先判断小队是否存在, 避免core
			if !ok {
				logger.Error("aid = %s tid = %s is not exist", req.Aid, tid)
				continue
			}
			// 查询小队统计
			teamStats, err := team.QueryTeamStats(req.Aid, tid)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.ActivityTeamRankResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query team stats error"))
				return
			}
			var nickHead supplement.MNickHead
			if len(t.TeamLeader) > 0 {
				nickHead, err = supplement.GetNickHead(t.TeamLeader)
				if err != nil {
					logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
					rsp = &pb.ActivityTeamRankResponse{
						Header: &metadata.CommonHeader{
							Code: errors.ConvertAPIErrorCode(err),
							Msg:  err.Error(),
						},
					}
					alarm.CallAlarmFunc(fmt.Sprintf("query head error"))
					return
				}
			}

			var total_funds int64 = 0
			var total_steps int64 = 0
			if req.RankType == common.RAND_TYPE_FUND {
				total_funds = int64(z.Score)
			} else { // req.RankType == common.RAND_TYPE_STEP
				total_steps = int64(z.Score)
			}
			list = append(list, &pb.ActivityTeamRank{
				Rank:       start + int64(i),
				Name:       t.TeamDesc,
				Head:       nickHead.Head,
				TotalUsers: teamStats.TeamMember,
				TotalSteps: total_steps,
				Aid:        req.Aid,
				TeamId:     tid,
				TotalFunds: total_funds,
			})
		}
	}

	// response ...
	rsp = &pb.ActivityTeamRankResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Total:      stats.ActivityMember,
		UpdateTime: stats.UpdateTime,
		List:       list,
		TeamRank:   teamRank,
	}
}


func ActivityUserRank(ctx *gms.Context) {
	defer util.GetUsedTime("ActivityUserRank")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.ActivityUserRankResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.ActivityUserRankRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.ActivityUserRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	act, err := activity.QueryActivity(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityUserRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
		return
	}
	if act == nil || act.Activity == nil || len(act.Activity.ActivityId) == 0 {
		// response ...
		rsp = &pb.ActivityUserRankResponse{
			Header: &metadata.CommonHeader{
				Msg:  "Param Invalid",
				Code: errors.ParamInvalid,
			},
		}
		return
	}

	// 查询活动统计
	stats, err := activity.QueryActivityStats(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ActivityUserRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query activity stats error"))
		return
	}

	// 本人步数排名 or 公益金排名
	var userRank int64 = 0
	if len(req.Oid) > 0 {
		if req.RankType == common.RAND_TYPE_FUND {
			_, userRank, err = activity.QueryActivityUserFundRank(req.Aid, req.Oid)
		} else { // req.RankType == common.RAND_TYPE_STEP
			_, userRank, err = activity.QueryActivityUserRank(req.Aid, req.Oid)
		}
		if err != nil {
			logger.Error("%v - traceID: %v, query user %s rank err: %s", util.GetCallee(), ctx.Request.TraceId, req.RankType, err.Error())
			rsp = &pb.ActivityUserRankResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query user self rank error"))
			return
		}
		logger.Error("%v - traceID: %v, query self user[%s] %s rank succ:%d", util.GetCallee(), ctx.Request.TraceId, req.Oid, req.RankType, userRank)
	}

	// 查询用户排名  //type: step步数  fund公益金
	zs, err := activity.QueryUserRank(req.Aid, req.Page, req.Size, req.RankType)
	if err != nil {
		logger.Error("%v - traceID: %v, query user %s rank err: %s", util.GetCallee(), ctx.Request.TraceId, req.RankType, err.Error())
		rsp = &pb.ActivityUserRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query user rank error"))
		return
	}

	list := make([]*pb.ActivityUserRank, 0)

	if len(zs) > 0 {
		var oids []string
		for _, z := range zs {
			if z.Member == nil {
				continue
			}
			oids = append(oids, z.Member.(string))
		}

		mnh := make(map[string]supplement.MNickHead)
		if len(oids) > 0 {
			mnh, err = supplement.MGetNickHead(oids)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.ActivityUserRankResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query heads error"))
				return
			}
		}

		var start = req.Page*req.Size + 1
		for i, z := range zs {
			if z.Member == nil {
				continue
			}
			o := z.Member.(string)
			userMR, err := fundClient.GetUserMatchRecordByOffset(o, req.Aid, 0, 0)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.ActivityUserRankResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query team stats error"))
				return
			}

			// 查询用户步数
			userStats, err := activity.QueryActivityUserStats(req.Aid, o)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.ActivityUserRankResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query activity user stats error"))
				return
			}

			var total_fund int64 = 0
			var total_step int64 = 0
			if req.RankType == common.RAND_TYPE_FUND {
				total_fund = int64(z.Score)
			} else { //req.RankType == common.RAND_TYPE_STEP
				total_step = int64(z.Score)
			}

			list = append(list, &pb.ActivityUserRank{
				Rank:        start + int64(i),
				Nick:        mnh[o].Nick,
				Head:        mnh[o].Head,
				TotalSteps:  total_step,
				TodaySteps:  userStats.TodaySteps,
				DonateMoney: int64(userMR.TotalFunds),
				Oid:         o,
				CreateTime:  time.Now().In(util.Loc).Format("2006-01-02 15:04:05"),
				TotalFund:   total_fund,
			})
		}
	}

	// response ...
	rsp = &pb.ActivityUserRankResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Total:      stats.ActivityMember,
		UpdateTime: stats.UpdateTime,
		List:       list,
		UserRank:   userRank,
	}
}


func QueryActivityUserStats(ctx *gms.Context) {
	defer util.GetUsedTime("QueryActivityUserStats")()
	// default pack rsp
	//rsp := &pb.QueryActivityUserStatsResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryActivityUserStatsResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryActivityUserStatsRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryActivityUserStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 查询用户参与活动的小队 TODO: 用户一个活动加入多个小队?
	teamStats := &pb.TeamStats{}
	teamInfo, err := queryUserJoinTeam(req.Aid, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryActivityUserStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("queryUserJoinTeam error"))
		return
	}
	// 查询用户参与的小队统计信息
	if len(teamInfo.TeamId) > 0 {
		teamStats, err = team.QueryTeamStats(req.Aid, teamInfo.TeamId)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryActivityUserStatsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query team stats error"))
			return
		}
	}

	// 查询用户活动的统计
	userStats, err := activity.QueryActivityUserStats(req.Aid, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryActivityUserStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query QueryActivityUserStats error"))
		return
	}

	// 查询活动的统计
	activityStats, err := activity.QueryActivityStats(req.Aid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryActivityUserStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query QueryActivityStats error"))
		return
	}

	// 查询活动型证书, 只有有配捐才会有证书
	cert := &pb.Cert{}
	if userStats.MatchMoney > 0 {
		act, err := activity.QueryActivity(req.Aid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryActivityUserStatsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity error"))
			return
		}
		cert, err = calcCert(req.Aid, req.Oid, act.Match.MatchInfo.FPid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryActivityUserStatsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("calcCert error"))
			return
		}
	}

	// response ...
	rsp = &pb.QueryActivityUserStatsResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		UserStats:     userStats,
		ActivityStats: activityStats,
		Team:          teamInfo,
		TeamStats:     teamStats,
		Cert:          cert,
	}
}

func QueryUserJoinActivityNew(ctx *gms.Context) {
	defer util.GetUsedTime("QueryUserJoinActivityNew")()
	// default pack rsp
	var rsp *pb.QueryUserJoinActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryUserJoinActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("request unmarshal activity error"))
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 查询用户参与活动的数量
	total, err := member.GetActivityMemberSize(req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		if !errors.IsOpsNeedRetryError(err) {
			alarm.CallAlarmFunc("query user join activity size error")
		}
		return
	}

	// 根据状态分组
	var running []*meta.CustomizedActivity
	var notStart []*meta.CustomizedActivity
	var over []*meta.CustomizedActivity
	var offset, count = 0, 10
	var time1, time2 time.Duration
	for offset < total {
		now := time.Now()
		// 查询用户参与的活动列表
		actMembers, err := member.GetActivityMember(req.Oid, offset, count)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryUserJoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity member error"))
			return
		}
		time1 += time.Since(now)
		now2 := time.Now()
		for _, v := range actMembers {
			// 查询活动基础信息
			ca, err := activity.QueryActivityMeta(v.ActivityId)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.QueryUserJoinActivityResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query activity meta error"))
				return
			}

			if ca.Activity.Status == meta.Activity_RUNNING {
				// 正在运行
				running = append(running, ca)
			} else if ca.Activity.Status == meta.Activity_DRAFT || ca.Activity.Status == meta.Activity_READY {
				// 未开始
				notStart = append(notStart, ca)
			} else {
				// 已结束
				over = append(over, ca)
			}
		}
		time2 += time.Since(now2)
		offset = offset + count
	}
	logger.Info("GetActivityMember time elapse part1: %v, part2: %v", time1, time2)

	// 整合不同状态的活动
	var actList []*meta.CustomizedActivity
	actList = append(actList, running...)
	actList = append(actList, notStart...)
	actList = append(actList, over...)

	// 分页处理
	var pageList []*meta.CustomizedActivity
	offset = int(req.Page * req.Size)
	size := int(req.Size)
	for i := offset; i < offset+size; i++ {
		if i > len(actList)-1 {
			break
		}
		pageList = append(pageList, actList[i])
	}

	time1 = 0
	time2 = 0
	// 查询活动
	list := make([]*pb.UserJoinActivity, 0)
	for _, ca := range pageList {
		if ca.Activity.Status == meta.Activity_RUNNING {
			matchResp, err := fundClient.GetMatchEvent(ca.Activity.MatchId)
			if err != nil || matchResp.Header.Code != 0 {
				logger.Error("aid = %s, eid = %s GetMatchEvent err = %v", ca.Activity.ActivityId, ca.Activity.MatchId, err)
				ca.Match = &meta.MatchActivity{
					Company:    &metadata.CompanyInfo{},
					MatchStats: &metadata.MatchStats{},
				}
			} else {
				ca.Match = matchResp.MatchEvent
			}
		} else {
			ca.Match = &meta.MatchActivity{
				Company:    &metadata.CompanyInfo{},
				MatchStats: &metadata.MatchStats{},
			}
		}

		now := time.Now()
		// 查询活动统计
		activityStats, err := activity.QueryActivityStats(ca.Activity.ActivityId)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryUserJoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity stats error"))
			return
		}
		time1 += time.Since(now)
		now2 := time.Now()

		res := &pb.UserJoinActivity{
			CustomizedActivity: ca,
			ActivityStats:      activityStats,
		}

		list = append(list, res)
		time2 += time.Since(now2)
	}
	logger.Info("QueryActivityStats time elapse part1: %v, part2: %v", time1, time2)

	// 查询用户是否创建活动
	hasCreate, err := getHasCreate(req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.QueryUserJoinActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Total:              int64(total),
		HasCreate:          hasCreate,
		UserJoinActivities: list,
	}
}

func getHasCreate(oid string) (bool, error) {
	has, err := member.GetProfileHasCreate(oid)
	// 回补
	if err != nil && err == redis.Nil {
		// 从db查询用户创建活动的数量（状态未支付除外）
		createTotal, err := activity.QueryUserAllCreateSizeDB(oid)
		if err != nil {
			logger.Error("QueryUserAllCreateSizeDB err: %v", err)
			return false, err
		}
		has = false
		if createTotal > 0 {
			has = true
		}
		err = member.SetProfileHasCreate(oid, has)
		if err != nil {
			logger.Error("SetProfileHasCreate err: %v", err)
			return false, err
		}
		return has, nil
	}
	if err != nil {
		logger.Error("GetProfileHasCreate err: %v", err)
		return false, err
	}

	return has, nil
}


func QueryTeamStats(ctx *gms.Context) {
	defer util.GetUsedTime("QueryTeamStats")()
	// default pack rsp
	//rsp := &pb.QueryTeamStatsResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryTeamStatsResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryTeamStatsRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryTeamStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("request unmarshal error"))
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 查询小队统计
	teamStats, err := team.QueryTeamStats(req.Aid, req.TeamId)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryTeamStatsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query team stats error"))
		return
	}

	// response ...
	rsp = &pb.QueryTeamStatsResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		TeamStats: teamStats,
	}
}

func QueryUserCreateActivity(ctx *gms.Context) {
	defer util.GetUsedTime("QueryUserCreateActivity")()
	// default pack rsp
	//rsp := &pb.QueryUserJoinActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryUserJoinActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryUserJoinActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("request unmarshal activity error"))
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// 查询用户创建活动的数量
	total, err := activity.QueryUserCreateSize(req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query user create activity size error"))
		return
	}

	// 根据状态分组
	var running []*meta.CustomizedActivity
	var notStart []*meta.CustomizedActivity
	var over []*meta.CustomizedActivity
	var offset, count int64 = 0, 10
	for offset < total {
		// 查询用户参与的活动列表
		actMembers, err := activity.QueryUserCreate(req.Oid, offset, count)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryUserJoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity member error"))
			return
		}

		for _, v := range actMembers {
			if nil == v.Member {
				continue
			}
			aid := v.Member.(string)
			// 查询活动基础信息
			ca, err := activity.QueryActivity(aid)
			if err != nil {
				logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
				rsp = &pb.QueryUserJoinActivityResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				alarm.CallAlarmFunc(fmt.Sprintf("query activity meta error"))
				return
			}

			if ca.Activity.Status == meta.Activity_RUNNING {
				// 正在运行
				running = append(running, ca)
			} else if ca.Activity.Status == meta.Activity_DRAFT || ca.Activity.Status == meta.Activity_READY {
				// 未开始
				notStart = append(notStart, ca)
			} else {
				// 已结束
				over = append(over, ca)
			}
		}

		offset = offset + count
	}

	// 整合不同状态的活动
	var actList []*meta.CustomizedActivity
	actList = append(actList, running...)
	actList = append(actList, notStart...)
	actList = append(actList, over...)

	// 分页处理
	var pageList []*meta.CustomizedActivity
	offset = req.Page * req.Size
	size := req.Size
	for i := offset; i < offset+size; i++ {
		if int(i) > len(actList)-1 {
			break
		}
		pageList = append(pageList, actList[i])
	}

	// 查询活动
	list := make([]*pb.UserJoinActivity, 0)
	for _, ca := range pageList {
		// 查询活动统计
		activityStats, err := activity.QueryActivityStats(ca.Activity.ActivityId)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryUserJoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity stats error"))
			return
		}
		// 查询用户统计
		userStats, err := activity.QueryActivityUserStats(ca.Activity.ActivityId, req.Oid)
		if err != nil {
			logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
			rsp = &pb.QueryUserJoinActivityResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("query activity user stats error"))
			return
		}

		res := &pb.UserJoinActivity{
			CustomizedActivity: ca,
			ActivityStats:      activityStats,
			ActivityUserStats:  userStats,
		}

		list = append(list, res)
	}

	// 查询用户是否创建活动
	hasCreate, err := getHasCreate(req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryUserJoinActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.QueryUserJoinActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Total:              total,
		HasCreate:          hasCreate,
		UserJoinActivities: list,
	}
}

// 用户加入活动
func joinActivity(aid, oid string, steps int64) error {
	// 更新用户在活动的排名 /// 步数排名
	if err := activity.UpdateUserRank(aid, oid, steps); err != nil {
		return err
	}
	// 初略调整活动的总步数
	if err := activity.ModifyActivitySteps(aid, oid, steps); err != nil {
		return err
	}

	/// 获取用户在该活动下的公益金存量
	funds, err := activity.GetActUserFunds(aid, oid)
	if err != nil {
		return err
	}

	/// 更新用户在活动的公益金排名
	if err := activity.UpdateUserFundsRank(aid, oid, funds); err != nil {
		return err
	}
	// 调整活动的总公益金
	if err := activity.ModifyActivityFunds(aid, oid, funds); err != nil {
		return err
	}

	// 更新用户加入活动状态
	actMember := &metadata.ActivityMember{
		ActivityId: aid,
		UserId:     oid,
		Status:     member.IN_ACTIVITY,
		CreateTime: time.Now().In(util.Loc).Format("2006-01-02 15:04:05"),
	}
	if err := member.SetActivityMember(actMember); err != nil {
		return err
	}

	return nil
}

// 查询用户在活动加入的小队信息
func queryUserJoinTeam(aid, oid string) (*metadata.Team, error) {
	// 查询用户参与活动的小队 TODO: 用户一个活动加入多个小队?
	teamInfo := &metadata.Team{}

	members, err := member.GetTeamMember(oid, aid, 0, 30)
	if err != nil {
		logger.Error("aid = %s, oid = %s, GetTeamMember err = %v", aid, oid, err)
		return teamInfo, err
	}

	for _, item := range members {
		if member.IN_TEAM == item.InTeam { // 用户在活动有加入小队
			// 查询小队信息
			teams, err := team.QueryTeams(aid, []string{item.TeamId})
			if err != nil {
				return teamInfo, err
			}
			// 找到用户加入的小队
			if len(teams) > 0 {
				teamInfo = teams[0]
				return teamInfo, nil
			}
		}
	}

	return teamInfo, nil
}

// 检查小队,如果没人并且是自建小队。删除小队
func checkTeam(aid, teamId string) {
	// if team member is 0, rm it from redis and update db
	stat, err := team.QueryTeamStats(aid, teamId)
	if err != nil {
		logger.Error("aid = %s, team_id = %s get team error = %v", aid, teamId)
	} else {
		logger.Info("aid = %s, team_id = %s has member：%v", aid, teamId, stat.TeamMember)
		var loop = true
		for loop {
			loop = false
			if stat.TeamMember == 0 {
				teamsInfo, err := team.QueryTeams(aid, []string{teamId})
				if err != nil {
					logger.Error("team.QueryTeams error: %v", err)
					break
				}
				if len(teamsInfo) == 0 {
					logger.Error("team.QueryTeams result is 0, no aid: %v, teamID: %v info", aid, teamId)
					break
				}
				if teamsInfo[0].TeamFlag == metadata.Team_SYS_TEAM {
					logger.Info("team is system team, aid: %v, teamID: %v", aid, teamId)
					break
				}
				err = team.DeleteTeam(aid, teamId)
				if err != nil {
					logger.Error("aid = %s, team_id = %s delete team error = %v", aid, teamId, err)
				} else {
					logger.Info("aid = %s, team_id = %s delete team success", aid, teamId)
				}
				err = activity.RemoveActivityTeamRank(aid, teamId)
				if err != nil {
					logger.Error("aid = %s, team_id = %s delete rm in activity rank error = %v", aid, teamId, err)
				} else {
					logger.Info("aid = %s, team_id = %s delete rm in activity rank success", aid, teamId)
				}
			}
		}

	}
}

// 用户加入/退出小队
func joinTeam(aid, teamId, oid, phoneNum string, join bool, t *metadata.Team, a *meta.CustomizedActivity) error {
	teamStats, err := team.QueryTeamStats(aid, teamId)
	if err != nil {
		return err
	}
	// 用户加入小队时, 超过小队人数上限
	teamMemberLimit := a.Activity.TeamMemberLimit
	if join && teamMemberLimit > 0 && teamStats.TeamMember >= int64(teamMemberLimit) {
		logger.Debug("aid = %s, team_id = %s, oid = %s, join team over limit = %d", aid, teamId, oid, teamMemberLimit)
		return errors.NewActivityMgrOverTeamMemberLimitError(errors.WithMsg("oid = %s join team over limit", oid))
	}

	// 查询用户的活动步数
	startDate := util.Datetime2Date(a.Activity.StartTime)
	endDate := util.Datetime2Date(a.Activity.EndTime)
	steps, err := steps2.UpdateUserSteps(aid, oid, startDate, endDate)
	if err != nil {
		return err
	}

	// 查询用户是否有加入活动
	stats, err := activity.QueryActivityUserStats(aid, oid)
	if err != nil {
		logger.Error("aid = %s, oid = %s is QueryActivityUserStats err = %v", aid, oid, err)
		return err
	}
	// 如果没有加入活动, 自动将用户加入活动 TODO： 补充appid
	if !stats.Join {
		// 检查用户加入活动是否超过上限
		if err = checkUserJoinOverLimit(oid); err != nil {
			return err
		}

		// 验证手机号码
		if a.Activity.WhiteType == meta.Activity_WhiteList {
			isWhite, err := activity.IsActivityWhiteList(aid, phoneNum)
			if err != nil {
				logger.Error("aid = %s, oid = %s is IsActivityWhiteList err = %v", aid, oid, err)
				return err
			}
			if isWhite == false {
				return errors.NewNoPermission(errors.WithMsg("no permission"))
			}
		}

		// 用户加入活动
		if err = joinActivity(aid, oid, steps); err != nil {
			logger.Error("aid = %s, oid = %s is joinActivity err = %v", aid, oid, err)
			return err
		}
	}

	var inTeam int32 = 2
	if join == false {
		inTeam = 1
	}

	if join {
		if err = team.UpdateUserRank(aid, teamId, oid, steps); err != nil {
			return err
		}
	} else {
		if err = team.DeleteUserRank(aid, teamId, oid); err != nil {
			return err
		}
	}
	// 初略更新小队总步数
	if err = team.ModifyTeamSteps(aid, teamId, oid, steps, join); err != nil {
		return err
	}

	// 更新小队排名及总公益金数据
	if err = team.ModifyTeamFunds(aid, teamId, oid, join); err != nil {
		return err
	}

	// 2.更新用户加入小队状态
	teamMember := &metadata.TeamMember{
		ActivityId: aid,
		UserId:     oid,
		TeamId:     teamId,
		InTeam:     inTeam,
		Status:     0,
		CreateTime: time.Now().In(util.Loc).Format("2006-01-02 15:04:05"),
	}
	if err = member.SetTeamMember(teamMember); err != nil {
		logger.Error("aid = %s, team_id = %s, oid = %s, join = %v set team member err = %v", aid, teamId, oid, join, err)
		return err
	}

	// 3.加入或者退出小队可能会触发队长变更
	if err = team.ElectTeamLeader(oid, join, t); err != nil {
		return err
	}

	// 4.检查小队，如果没人并且是自建小队。删除小队
	checkTeam(aid, teamId)

	return nil
}

// 用户主动更新步数, 更新下面几个数据:
// 1.活动的用户排行榜
// 2.小队的用户排行榜
// 3.活动的小队排行榜
// 4.活动的总步数
// 5.小队的总步数
func updateActivityRankStats(oid string, userSteps *meta.UserSteps) error {
	// 查询用户所有的活动
	total, err := member.GetActivityMemberSize(oid)
	if err != nil {
		logger.Error("oid = %s GetActivityMemberSize size err = %v", oid, err)
		return err
	}

	// 查询每个活动
	acts, err := member.GetActivityMember(oid, 0, total)
	if err != nil {
		logger.Error("oid = %s GetActivityMember err = %v", oid, err)
		return err
	}

	now := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")

	// 更新每个活动的排名
	for _, item := range acts {
		// 查询活动信息
		aid := item.ActivityId
		ca, err := activity.QueryActivity(aid)
		if err != nil {
			return err
		}
		// 进行中的活动才更新排名
		if ca.Activity.StartTime > now || ca.Activity.EndTime < now {
			logger.Info("aid = %s, oid = %s, start_time = %s, end_time = %s, activity time not match",
				aid, oid, ca.Activity.StartTime, ca.Activity.EndTime)
			continue
		}
		// 查询用户加入的小队, 用户可能没加入任何小队
		t, err := queryUserJoinTeam(aid, oid)
		if err != nil {
			return err
		}
		// 查询用户原来的活动步数
		oldSteps, err := steps2.QueryUserSteps(aid, oid)
		if err != nil {
			return err
		}
		// 获取用户的最新步数, TODO: 用传参计算
		startDate := util.Datetime2Date(ca.Activity.StartTime)
		endDate := util.Datetime2Date(ca.Activity.EndTime)
		newSteps, err := steps2.UpdateUserSteps(aid, oid, startDate, endDate)
		if err != nil {
			return err
		}
		// 用户手动更新步数时, 检查用户的活动总步数是否发生改变
		deltaStep := newSteps - oldSteps.ActivitySteps
		if deltaStep < 0 {
			logger.Error("aid = %s, oid = %s, old_steps = %d, new_steps = %d change small err", aid, oid, oldSteps.ActivitySteps, newSteps)
			deltaStep = 0
		}
		logger.Info("aid = %s, oid = %s, team_id = %s, old_steps = %d, new_steps = %d prepare to update rank",
			aid, oid, t.TeamId, oldSteps.ActivitySteps, newSteps)
		// 1.更新活动的用户排行榜
		if err = activity.UpdateUserRank(aid, oid, newSteps); err != nil {
			return err
		}
		// 2.更新活动的总步数
		if err = activity.ModifyActivitySteps(aid, oid, deltaStep); err != nil {
			return err
		}
		if len(t.TeamId) > 0 { // 用户加入了活动的小队
			// 3.更新小队的用户排行榜
			if err = team.UpdateUserRank(aid, t.TeamId, oid, newSteps); err != nil {
				return err
			}
			// 4.更新小队的总步数
			// 5.更新活动的小队排行榜
			if err = team.ModifyTeamSteps(aid, t.TeamId, oid, deltaStep, true); err != nil {
				return err
			}
		}
	}

	return nil
}

// DeleteActivity ...
func DeleteActivity(ctx *gms.Context) {
	defer util.GetUsedTime("DeleteActivity")()
	// default pack rsp
	var rsp *pb.DeleteActivityResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
		} else {
			logger.Info("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
	}()
	// decode request
	req := &pb.DeleteActivityRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.DeleteActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)

	err = activity.DeleteActivity(req.ActivityId, req.OpType, req.Operator)
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		rsp = &pb.DeleteActivityResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// response ...
	rsp = &pb.DeleteActivityResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}

// DeleteActivityTeam ...
func DeleteActivityTeam(ctx *gms.Context) {
	defer util.GetUsedTime("DeleteActivityTeam")()
	// default pack rsp
	var rsp *pb.DeleteActivityTeamResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
		} else {
			logger.Info("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf("traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error(" ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
	}()
	// decode request
	req := &pb.DeleteActivityTeamRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.DeleteActivityTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)
	err = team.DeleteTeam(req.ActivityId, req.TeamId)
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		rsp = &pb.DeleteActivityTeamResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// response ...
	rsp = &pb.DeleteActivityTeamResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}


// UpdateActivityState ...
func UpdateActivityState(ctx *gms.Context) {
	defer util.GetUsedTime("UpdateActivityState")()
	// default pack rsp
	var rsp *pb.UpdateActivityStateResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
		} else {
			logger.Info("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf("traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
	}()
	// decode request
	req := &pb.UpdateActivityStateRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.UpdateActivityStateResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)
	err = activity.UpdateActivityState(req.ActivityId, meta.Activity_Status(req.State))
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		rsp = &pb.UpdateActivityStateResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// response ...
	rsp = &pb.UpdateActivityStateResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}

type actMembersSort []*metadata.ActivityMember

func (p actMembersSort) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p actMembersSort) Len() int           { return len(p) }
func (p actMembersSort) Less(i, j int) bool { return p[i].CreateTime > p[j].CreateTime }




func queryUserJoinSizeNew(oid string) (int, error) {

	t211N := time.Now()
	num, err := member.GetActivityMemberSizeDBNew(oid)
	if err != nil {
		return 0, err
	}
	if num == 0 {
		return 0, nil
	}
	logger.Info("t211N.GetUsedTime - (%s)", time.Since(t211N))
	return num, nil
}

func checkUserJoinOverLimit(oid string) error {
	// 查询用户加入活动的数量
	//joinSize, err := queryUserJoinSize(oid)
	joinSize, err := queryUserJoinSizeNew(oid)
	if err != nil {
		return err
	}

	// 用户加入的活动超过上限（排除删除和已结束的活动）
	joinLimit := config.GetConfig().ActivityConf.JoinLimit
	if joinLimit <= 0 {
		joinLimit = config.JoinLimitDefault
	}
	if joinSize >= joinLimit {
		logger.Debug("%v - oid = %s, size = %d, join activity over limit = %d", util.GetCallee(), oid, joinSize, joinLimit)
		return errors.NewActivityMgrOverMaxJoinLimitError(errors.WithMsg("oid = %s join activity over limit", oid))
	}

	return nil
}

func ChangeTeamLeader(ctx *gms.Context) {
	defer util.GetUsedTime("ChangeTeamLeader")()
	// default pack rsp
	//rsp := &pb.QueryUserJoinActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.ChangeTeamLeaderResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.ChangeTeamLeaderRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("request unmarshal activity error"))
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	if req.OperatorOid == req.Oid {
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Msg:  "Param Invalid",
				Code: errors.ParamInvalid,
			},
		}
		logger.Debug("%v - traceID: %v, teams size is 0", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)
		return
	}

	teams, err := team.QueryTeams(req.ActivityId, []string{req.TeamId})
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("query teams error"))
		return
	}

	if len(teams) == 0 {
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Msg:  "success",
				Code: errors.Success,
			},
		}
		logger.Debug("%v - traceID: %v, teams size is 0", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)
		return
	}

	teamInfo := teams[0]
	if teamInfo.TeamLeader != req.OperatorOid {
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Msg:  "no permission",
				Code: errors.NoPermission,
			},
		}
		logger.Debug("%v - traceID: %v, no permission", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)
		return
	}

	isJoin, err := member.IsJoinTeam(req.ActivityId, req.TeamId, req.Oid)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("IsJoinTeam error"))
		return
	}
	if isJoin == false {
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Msg:  "Param Invalid",
				Code: errors.ParamInvalid,
			},
		}
		logger.Debug("%v - traceID: %v, oid is not join team", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)
		return
	}

	// 更换队长
	teamInfo.TeamLeader = req.Oid
	err = team.ModifyTeam(teamInfo)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.ChangeTeamLeaderResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("modify team error"))
		return
	}

	// response ...
	rsp = &pb.ChangeTeamLeaderResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}

func QueryActivitySuccessNum(ctx *gms.Context) {
	defer util.GetUsedTime("QueryActivitySuccessNum")()
	// default pack rsp
	//rsp := &pb.GetActivityResponse{Header: &metadata.CommonHeader{}}
	var rsp *pb.QueryActivitySuccessNumResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Now().Unix()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
	}()

	// decode request
	req := &pb.QueryActivitySuccessNumRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewActivityMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.QueryActivitySuccessNumResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	num, err := activity.GetActivitySuccessNum()
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		rsp = &pb.QueryActivitySuccessNumResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.QueryActivitySuccessNumResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Num: int32(num),
	}
}


