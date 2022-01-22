//
package handler

import (
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	gms "git.code.oa.com/gongyi/gomore/service"
	XGYProto "git.code.oa.com/gongyi/gongyi_base/proto/gy_agent"
	pb "git.code.oa.com/gongyi/yqz/api/fundmgr"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/client"
	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/mqclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/activity"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/coupons"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/match"
	"google.golang.org/protobuf/proto"
)

// for monitor
var (
	MissCache                           int64 = 0
	GetMatchEventSuccess                int64 = 0
	GetMatchEventFailed                 int64 = 0
	CreateMatchEventSuccess             int64 = 0
	CreateMatchEventFailed              int64 = 0
	GetUserMatchDonateSuccess           int64 = 0
	GetUserMatchDonateFailed            int64 = 0
	GetUserWeekMatchRecordSuccess       int64 = 0
	GetUserWeekMatchRecordFailed        int64 = 0
	GetUserMatchRecordByOffsetSuccess   int64 = 0
	GetUserMatchRecordByOffsetFailed    int64 = 0
	GetUserTodayMatchSuccess            int64 = 0
	GetUserTodayMatchFailed             int64 = 0
	UserMatchSuccess                    int64 = 0
	UserMatchFailed                     int64 = 0
	UpdateMatchSuccess                  int64 = 0
	UpdateMatchFailed                   int64 = 0
	GetActivityMatchInfoSuccess         int64 = 0
	GetActivityMatchInfoFailed          int64 = 0
	GetActivityMatchRankSuccess         int64 = 0
	GetActivityMatchRankFailed          int64 = 0
	RemoveUserActivityMatchInfoSuccess  int64 = 0
	RemoveUserActivityMatchInfoFailed   int64 = 0
	RecoverUserActivityMatchRankSuccess int64 = 0
	RecoverUserActivityMatchRankFailed  int64 = 0
	GetCompanyMatchRankSuccess          int64 = 0
	GetCompanyMatchRankFailed           int64 = 0
)

func GetMatchEvent(ctx *gms.Context) {
	defer util.GetUsedTime("GetMatchEvent")()
	usedStart := time.Now()
	var rsp *pb.GetMatchEventResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetMatchEventSuccess, 1)
		} else {
			logger.Error("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetMatchEventFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "GetMatchEvent", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.GetMatchEventRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetMatchEventResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	event, err := activity.GetMatchEvent(activity.WithEventID(req.Eid))
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		err = errors.NewFundMgrInternalError(errors.WithMsg("create match event error"))
		rsp = &pb.GetMatchEventResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc("create match event error")
		return
	}

	// response ...
	rsp = &pb.GetMatchEventResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		MatchEvent: event,
		Status:     pb.GetMatchEventResponse_SUCCESS,
	}
}

func CreateMatchEvent(ctx *gms.Context) {
	defer util.GetUsedTime("CreateMatchEvent")()
	usedStart := time.Now()
	var rsp *pb.CreateMatchEventResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&CreateMatchEventSuccess, 1)
		} else {
			logger.Error("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&CreateMatchEventFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "CreateMatchEvent", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.CreateMatchEventRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.CreateMatchEventResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	event, err := activity.CreateMatchEvent(req.MatchInfo, req.MatchRule)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		err = errors.NewFundMgrInternalError(errors.WithMsg("create match event error"))
		rsp = &pb.CreateMatchEventResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc("create match event error")
		return
	}

	// response ...
	rsp = &pb.CreateMatchEventResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		MatchEvent: event,
		Status:     pb.CreateMatchEventResponse_SUCCESS,
	}
}


func GetUserMatchRecordByOffset(ctx *gms.Context) {
	defer util.GetUsedTime("GetUserMatchRecordByOffset")()
	usedStart := time.Now()
	var rsp *pb.GetUserMatchRecordByOffsetResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetUserMatchRecordByOffsetSuccess, 1)
		} else {
			logger.Error("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetUserMatchRecordByOffsetFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "GetUserMatchRecordByOffset", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.GetUserMatchRecordByOffsetRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetUserMatchRecordByOffsetResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// get user record list
	records, err := match.GetUserMatchRecordByOffsetDB(req.Oid, req.ActivityId, int(req.Offset), int(req.Size), true)
	if err != nil {
		logger.Error("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		err = errors.NewFundMgrInternalError(errors.WithMsg("get user: %v record error", req.Oid))
		rsp = &pb.GetUserMatchRecordByOffsetResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("get user:%v record from db error: %v", req.Oid, err))
		return
	}

	// get user summary
	var totalFunds, totalSteps, totalTimes int32
	if len(req.ActivityId) == 0 {
		status, err := GetUserSummaryMatchInfoCache(req.Oid)
		if err != nil {
			err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserSummaryMatchInfoCache error"))
			rsp = &pb.GetUserMatchRecordByOffsetResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
		totalFunds = int32(status.Funds)
		totalSteps = int32(status.Steps)
		totalTimes = int32(status.Times)
	} else {
		status, err := GetUserMatchInfoCache(req.Oid, req.ActivityId)
		if err != nil {
			err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserSummaryMatchInfoCache error"))
			rsp = &pb.GetUserMatchRecordByOffsetResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
		totalFunds = int32(status.TFunds)
		totalSteps = int32(status.TSteps)
		totalTimes = int32(status.TTimes)
	}
	// get activity
	var activityInfo = make(map[string]*metadata.MatchActivity)
	var userActivity = make(map[string]*metadata.Activity)
	for _, val := range records {
		if len(val.ActivityId) == 0 {
			if _, ok := activityInfo[val.Item]; ok {
				continue
			}
			act, err := activity.GetMatchEvent(activity.WithEventID(val.Item))
			if err != nil {
				logger.Error("%v - can not get item: %v info, err:%v", val.Item, val.Item, err)
				continue
			}
			activityInfo[val.Item] = act
		} else {
			if act, ok := activityInfo[val.Item]; ok {
				// check if had coupons
				if act.MatchInfo.FLoveCoupon {
					val.Coupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix())}}
				} else {
					val.Coupons = nil
				}
				if _, ok = userActivity[val.ActivityId]; ok {
					continue
				}
			}
			activityRsp, rpcErr := client.QueryActivity(val.ActivityId)
			if rpcErr != nil {
				logger.Error("rpc get activity: %v info, err: %v", val.ActivityId, err)
				continue
			}
			if activityRsp.Header.Code != 0 {
				logger.Error("can not get activity: %v info, header: %v", val.ActivityId, activityRsp.Header)
				continue
			}
			activityInfo[val.Item] = activityRsp.Activity.Match
			userActivity[val.ActivityId] = activityRsp.Activity.Activity
			// check if had coupons
			if activityRsp.Activity.Match.MatchInfo.FLoveCoupon {
				val.Coupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix())}}
			} else {
				val.Coupons = nil
			}
		}
	}
	// response ...
	rsp = &pb.GetUserMatchRecordByOffsetResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		TotalFunds:   totalFunds,
		TotalSteps:   totalSteps,
		TotalTimes:   totalTimes,
		Records:      records,
		ActivityInfo: activityInfo,
		UserActivity: userActivity,
	}
}

// GetUserTodayMatch get user today match info & today match item
func GetUserTodayMatch(ctx *gms.Context) {
	defer util.GetUsedTime("GetUserTodayMatch")()
	usedStart := time.Now()
	var rsp *pb.GetUserTodayMatchResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success, "
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetUserTodayMatchSuccess, 1)
		} else {
			logger.Error("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetUserTodayMatchFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "GetUserTodayMatch", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()
	// decode request
	req := &pb.GetUserTodayMatchRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetUserTodayMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)
	info, err := GetUserMatchInfoCache(req.Oid, req.ActivityId)
	if err != nil {
		err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserMatchInfoCache error"))
		rsp = &pb.GetUserTodayMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// judge the record is today or not for return msg
	timeNowFmt := time.Now().In(util.Loc).Format("2006-01-02")
	beforeTime, _ := time.ParseDuration("-24h")
	timeYesterdayFmt := time.Now().Add(beforeTime).In(util.Loc).Format("2006-01-02")
	matchTime, err := time.ParseInLocation("2006-01-02 15:04:05", info.Date, util.Loc)
	if err != nil {
		logger.Error("parse match time: %v err: %v", info.Date, err)
		err = errors.NewFundMgrInternalError(errors.WithMsg("record time format error"))
		rsp = &pb.GetUserTodayMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	lastMatchFmt := matchTime.In(util.Loc).Format("2006-01-02")
	rsp = &pb.GetUserTodayMatchResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
	// means match happen today/ yesterday/ others
	if lastMatchFmt == timeNowFmt {
		rsp.Records = info
		rsp.Combos = int32(info.Combo)
		rsp.MatchFlag = true
	} else if lastMatchFmt == timeYesterdayFmt {
		rsp.Combos = int32(info.Combo)
		rsp.Records = match.NullMatchRecord
	} else {
		rsp.Records = match.NullMatchRecord
	}

	// call activity mgr get activity's match item
	var matchItem *metadata.MatchActivity
	if len(req.ActivityId) != 0 {
		activityRsp, rpcErr := client.QueryActivity(req.ActivityId)
		if rpcErr != nil {
			logger.Error("rpc get activity: %v info, err: %v", req.ActivityId, err)
			rsp = &pb.GetUserTodayMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		if activityRsp.Header.Code != 0 {
			logger.Error("can not get activity: %v info, header: %v", req.ActivityId, activityRsp.Header)
			rsp = &pb.GetUserTodayMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		matchItem, err = activity.GetMatchEvent(activity.WithEventID(activityRsp.Activity.Activity.MatchId))
	} else {
		// get default match item
		matchItem, err = activity.GetMatchEvent()
	}

	if err != nil {
		if errors.IsFundMgrMatchNotExistError(err) {
			logger.Info("traceID: %v get no today match event err: %s", ctx.Request.TraceId, err.Error())
			rsp.Header = &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			}
		} else {
			logger.Error("traceID: %v get today match event err: %s", ctx.Request.TraceId, err.Error())
			err = errors.NewFundMgrInternalError(errors.WithMsg("get today match event"))
			rsp = &pb.GetUserTodayMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
		}
	} else {
		rsp.ActivityInfo = matchItem
		// check if had coupons
		if matchItem.MatchInfo.FLoveCoupon {
			rsp.Records.Coupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix())}}
		} else {
			rsp.Records.Coupons = nil
		}
	}
}

// UserMatch user match action
func UserMatch(ctx *gms.Context) {
	defer util.GetUsedTime("UserMatch")()
	usedStart := time.Now()
	var rsp *pb.UserMatchResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&UserMatchSuccess, 1)
		} else {
			logger.Error("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&UserMatchFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "UserMatch", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.UserMatchRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// get match item
	var matchItem *metadata.MatchActivity
	// call activity mgr get activity's match item
	if len(req.ActivityId) != 0 {
		activityRsp, rpcErr := client.QueryActivity(req.ActivityId)
		if rpcErr != nil {
			logger.Error("rpc get activity: %v info, err: %v", req.ActivityId, err)
			rsp = &pb.UserMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		if activityRsp.Header.Code != 0 {
			logger.Error("can not get activity: %v info, header: %v", req.ActivityId, activityRsp.Header)
			rsp = &pb.UserMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		matchItem, err = activity.GetMatchEvent(activity.WithEventID(activityRsp.Activity.Activity.MatchId))
	} else {
		// get activity
		matchItem, err = activity.GetMatchEvent()
	}
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		//err = errors.NewFundMgrInternalError(errors.WithMsg(msg))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// filter quota and match item status
	if req.Steps < matchItem.MatchRule.MatchQuota {
		// response ...
		logger.Info("%v - traceID: %v, user: %v, step: %v not satisfy quota: %v", util.GetCallee(), ctx.Request.TraceId, req.Oid, req.Steps, matchItem.MatchRule.MatchQuota)
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Msg:  "success",
				Code: errors.Success,
			},
			Result: pb.UserMatchResponse_FAILED_STEP_NOT_ENOUGH,
		}
		return
	} else if matchItem.MatchInfo.FStatus != 1 || matchItem.MatchStats.Remain < 1 {
		// response ...
		logger.Info("%v - traceID: %v user: %v activity is not in start status or no remain: %v", util.GetCallee(), ctx.Request.TraceId, req.Oid, matchItem.MatchStats.Remain)
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Msg:  "success",
				Code: errors.Success,
			},
			Result: pb.UserMatchResponse_FAILED_MATCH_END,
		}
		return
	}

	// GetUserMatchInfoCache
	info, err := GetUserMatchInfoCache(req.Oid, req.ActivityId)
	if err != nil {
		err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserMatchInfoCache error"))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// GetUserSummaryMatchInfoCache
	allInfo, err := GetUserSummaryMatchInfoCache(req.Oid)
	if err != nil {
		err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserSummaryMatchInfoCache error"))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// generate fund and check if already match today
	fund, buff, combo, matchDate, err := match.GenerateMatchFund(req.Oid, req.ActivityId, req.Steps, matchItem, info)
	if err != nil {
		if errors.IsFundMgrAlreadyMatchError(err) {
			info, err := match.GetUserMatchLastRecordDB(req.Oid, req.ActivityId)
			if err != nil {
				err = errors.NewFundMgrInternalError(errors.WithMsg("GetUserMatchLastRecordDB error"))
				rsp = &pb.UserMatchResponse{
					Header: &metadata.CommonHeader{
						Code: errors.ConvertAPIErrorCode(err),
						Msg:  err.Error(),
					},
				}
				return
			}
			// query user get pid coupons result
			loveCoupons := []*metadata.LoveCoupons{}
			if matchItem.MatchInfo.FLoveCoupon {
				loveCoupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix())}}
				/*
					loveCoupons, err = coupons.QueryUserAcquireCouponeResult(req.ActivityId, req.Oid)
					if err != nil {
						logger.Error("QueryUserAcquireCouponeResult act: %v, oid: %v, error: %v")
					}
				*/
			}
			rsp = &pb.UserMatchResponse{
				Header: &metadata.CommonHeader{
					Msg:  "success",
					Code: errors.Success,
				},
				Result: pb.UserMatchResponse_FAILED_ALREADY_MATCH,
				Records: &metadata.MatchRecord{
					Id:      info.Id,
					Funds:   int32(fund),
					Steps:   req.Steps,
					Combo:   int32(combo),
					Item:    matchItem.MatchInfo.FEventId,
					Op:      metadata.MatchRecord_OPType(buff),
					Date:    matchDate,
					Coupons: loveCoupons,
				},
			}
			return
		}
		logger.Error("%v - traceID: %v user: %v can not generate err: %v", util.GetCallee(), ctx.Request.TraceId, req.Oid, err)
		err = errors.NewFundMgrInternalError(errors.WithMsg("generate match fund error"))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// do match
	err = activity.HandleMatch(matchItem.MatchInfo.FEventId, int64(fund), int64(req.Steps))
	if err != nil {
		logger.Error("%v - traceID: %v, handle match: %v, fund: %v, err: %v", util.GetCallee(), ctx.Request.TraceId, matchItem.MatchInfo.FEventId, fund, err)
		err = errors.NewFundMgrInternalError(errors.WithMsg("user: %v do match error", req.Oid))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// add record first
	dateRaw := time.Now().In(util.Loc).Format("2006-01-02")
	date := time.Now().In(util.Loc).Format("2006-01-02 15:04:05")
	matchID, err := match.AddUserMatchRecordDB(req.Oid, req.ActivityId, fund, int(req.Steps), combo+1, int(buff), matchItem.MatchInfo.FEventId, matchItem.MatchInfo.FCompanyId, date)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate entry") {
			logger.Error("%v - traceID: %v, add user: %v match record to db error: %v", util.GetCallee(), ctx.Request.TraceId, req.Oid, err)
			alarm.CallAlarmFunc(fmt.Sprintf("match record duplicate error - user:%v, actID: %v, date: %v", req.Oid, req.ActivityId, date))
			info, err := match.GetUserMatchLastRecordDB(req.Oid, req.ActivityId)
			if err != nil {
				logger.Error("match.GetUserMatchLastRecordDB err: %v", err)
				// query user get pid coupons result
				loveCoupons := []*metadata.LoveCoupons{}
				if matchItem.MatchInfo.FLoveCoupon {
					loveCoupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix())}}
					/*
						loveCoupons, err = coupons.QueryUserAcquireCouponeResult(req.ActivityId, req.Oid)
						if err != nil {
							logger.Error("QueryUserAcquireCouponeResult act: %v, oid: %v, error: %v")
						}
					*/
				}
				rsp = &pb.UserMatchResponse{
					Header: &metadata.CommonHeader{
						Msg:  "success",
						Code: errors.Success,
					},
					Result: pb.UserMatchResponse_FAILED_ALREADY_MATCH,
					Records: &metadata.MatchRecord{
						Id:      info.Id,
						Funds:   int32(fund),
						Steps:   req.Steps,
						Combo:   int32(combo),
						Item:    matchItem.MatchInfo.FEventId,
						Op:      metadata.MatchRecord_OPType(buff),
						Date:    matchDate,
						Coupons: loveCoupons,
					},
				}
				return
			} else {
				fund = int(info.Funds)
				combo = int(info.Combo)
				date = info.Date
				matchID = info.Id
			}
		} else {
			logger.Error("%v - traceID: %v, add user: %v match record to db error: %v", util.GetCallee(), ctx.Request.TraceId, req.Oid, err)
			err = errors.NewFundMgrInternalError(errors.WithMsg("user: %v add match record error", req.Oid))
			rsp = &pb.UserMatchResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			alarm.CallAlarmFunc(fmt.Sprintf("add user:%v record to db error: %v", req.Oid, err))
			return
		}
	}
	var addDay int = 0
	if allInfo.Date != dateRaw {
		logger.Debug("check add date - user: %v add day, last: %v, now: %v", req.Oid, allInfo.Date, dateRaw)
		addDay = 1
	}

	// change match activity redis
	t_fund, t_step, t_times, sum_fund, sum_day, sum_time, err := match.UpdateUserAllMatchInfoCache(req.Oid, req.ActivityId, fund, int(req.Steps), combo+1, int(buff), matchItem.MatchInfo.FEventId, date, dateRaw, addDay, matchID)
	if err != nil {
		err = errors.NewFundMgrInternalError(errors.WithMsg("UpdateUserAllMatchInfoCache user: %v error: %v", req.Oid, err))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("update user:%v match info to redis error: %v", req.Oid, err))
		return
	}

	// change status db later
	err = match.UpdateUserAllMatchStatusDB(
		req.Oid, req.ActivityId, combo+1, int(buff), matchItem.MatchInfo.FEventId, matchItem.MatchInfo.FCompanyId, date, t_fund, t_step, t_times,
		dateRaw, int(sum_fund), int(sum_day), int(sum_time))
	if err != nil {
		err = errors.NewFundMgrInternalError(errors.WithMsg("UpdateUserAllMatchStatusDB user: %v error: %v", req.Oid, err))
		rsp = &pb.UserMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		alarm.CallAlarmFunc(fmt.Sprintf("update user:%v match status to db error: %v", req.Oid, err))
		return
	}

	// add activity rank
	if len(req.ActivityId) != 0 {
		err = match.UpdateActivityMatchRank(req.ActivityId, req.Oid, int(t_fund))
		if err != nil {
			logger.Error("update activity: %v user: %v rank error", req.ActivityId, req.Oid, err)
		}
	}

	// call async get coupons
	loveCoupons := []*metadata.LoveCoupons{}
	if matchItem.MatchInfo.FLoveCoupon {
		err = acquireLoveCoupons(req.ActivityId, req.Oid, matchID)
		if err != nil {
			logger.Error("acquireLoveCoupons act: %v, oid: %v, error: %v")
		} else {
			loveCoupons = []*metadata.LoveCoupons{{SendTime: int32(util.GetLocalTime().Unix()), SendResult: 2}}
		}
	}
	// send to kafka for update user rank
	if err = SendMsg2KafkaLocal(req.Oid, req.ActivityId, matchID, dateRaw, int64(fund)); err != nil {
		logger.Error("SendMsg2KafkaLocal error: %v", err)
	} else {
		logger.Info("SendMsg2KafkaLocal oid: %v, matchID: %v success", req.Oid, matchID)
	}

	// response ...
	rsp = &pb.UserMatchResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		Result: pb.UserMatchResponse_SUCCESS,
		Records: &metadata.MatchRecord{
			Id:      matchID,
			Funds:   int32(fund),
			Steps:   req.Steps,
			Combo:   int32(combo + 1),
			Item:    matchItem.MatchInfo.FEventId,
			Op:      metadata.MatchRecord_OPType(buff),
			Date:    date,
			Coupons: loveCoupons,
		},
	}
}

func acquireLoveCoupons(pid, oid, order string) error {
	// async get coupons from coupons system
	if err := coupons.AcquirePidCoupons(pid, oid, order); err != nil {
		logger.Error("AcquirePidCoupons aid: %v, oid: %v, order: %v, error: %v", pid, oid, order, err)
		return err
	}
	return nil
}

// GetUserSummaryMatchInfoCache ...
func GetUserSummaryMatchInfoCache(oid string) (*metadata.MatchStatus, error) {
	var err error
	allInfo, cacheErr := match.GetUserSummaryMatchInfoCache(oid)
	if cacheErr != nil {
		if errors.IsRedisNilError(cacheErr) {
			logger.Info("match.GetUserSummaryMatchInfoCache oid: %v, err: %v, so try get from db", oid, cacheErr)
			atomic.AddInt64(&MissCache, 1)
		} else {
			logger.Error("match.GetUserSummaryMatchInfoCache oid: %v, err: %v, so try get from db", oid, cacheErr)
			alarm.CallAlarmFunc(fmt.Sprintf("get user:%v summary match info from redis error: %v", oid, cacheErr))
		}
		// recover from record
		allInfo, err = match.RecoverUserSummaryMatchStatusDB(oid)
		if err != nil {
			if !errors.IsDBNilError(err) {
				alarm.CallAlarmFunc(fmt.Sprintf("RecoverUserSummaryMatchStatusDB user:%v summary match status from db error: %v", oid, err))
				return nil, err
			} else if errors.IsDBNilError(err) {
				logger.Info("oid: %v no db summary match status", oid)
				allInfo = match.NullMatchStatus
			}
		}
		/*
			// get from redis failed so we get from db
			allInfo, err = match.GetUserSummaryMatchStatusDB(oid)
			if err != nil {
				if !errors.IsDBNilError(err) {
					alarm.CallAlarmFunc(fmt.Sprintf("get user:%v summary match status from db error: %v", oid, err))
					return nil, err
				} else {
					// recover from record, this will only happen first time
					allInfo, err = match.RecoverUserSummaryMatchStatusDB(oid)
					if err != nil {
						if !errors.IsDBNilError(err) {
							alarm.CallAlarmFunc(fmt.Sprintf("RecoverUserSummaryMatchStatusDB user:%v summary match status from db error: %v", oid, err))
							return nil, err
						} else if errors.IsDBNilError(err) {
							logger.Info("oid: %v no db summary match status", oid)
							allInfo = match.NullMatchStatus
						}
					}
				}
			}
		*/
	}
	// set redis, if redis is nil. this happen for user never donate before or redis instance change
	if errors.IsRedisNilError(cacheErr) {
		err := match.SetUserSummaryMatchInfoCache(oid, allInfo.Date, int(allInfo.Funds), int(allInfo.Days), int(allInfo.Times))
		if err != nil {
			logger.Error("match.SetUserSummaryMatchInfoCache err: %v", err)
		}
	}
	return allInfo, nil
}

// GetUserMatchInfoCache ...
func GetUserMatchInfoCache(oid, activityId string) (*metadata.MatchRecord, error) {
	// get redis first, if redis null, mean this guy has never match
	// but this should not happen ,because user must call GetUserTodayMatch first
	var err error
	info, cacheErr := match.GetUserMatchInfoCache(oid, activityId)
	if cacheErr != nil {
		if errors.IsRedisNilError(cacheErr) {
			logger.Info("match.GetUserMatchInfoCache oid: %v activity: %v err: %v, so try get from db", oid, activityId, cacheErr)
			atomic.AddInt64(&MissCache, 1)
		} else {
			logger.Error("match.GetUserMatchInfoCache oid: %v activity: %v err: %v, so try get from db", oid, activityId, cacheErr)
			alarm.CallAlarmFunc(fmt.Sprintf("get user:%v activity: %v match info from redis error: %v", oid, activityId, cacheErr))
		}
		// get from redis failed so we get from db
		info, err = match.GetUserMatchLastRecordDB(oid, activityId)
		if err != nil {
			if !errors.IsDBNilError(err) {
				logger.Error("match.GetUserMatchLastRecordDB user:%v activity: %v error: %v", oid, activityId, err)
				alarm.CallAlarmFunc(fmt.Sprintf("match.GetUserMatchLastRecordDB user:%v activity: %v error: %v", oid, activityId, err))
				return nil, err
			} else {
				logger.Info("oid: %v activity: %v no db match record", oid, activityId)
				info = match.NullMatchRecord
			}
		}
	}
	// set redis, if redis is nil. this happen for user never donate before or redis instance change
	if errors.IsRedisNilError(cacheErr) {
		status, err := match.GetUserMatchStatusDB(oid, activityId)
		if err != nil && !errors.IsDBNilError(err) {
			logger.Error("match.GetUserMatchStatusDB user:%v activity: %v error: %v", oid, activityId, err)
			alarm.CallAlarmFunc(fmt.Sprintf("match.GetUserMatchStatusDB user:%v activity: %v error: %v", oid, activityId, err))
		} else {
			err = match.SetUserMatchInfoCache(oid, activityId,
				int(info.Funds), int(info.Steps), int(info.Combo), int(info.Op), info.Item, info.Date,
				status.Steps, status.Funds, int(status.Times), info.Id)
			if err != nil {
				logger.Error("match.SetUserMatchInfoCache oid: %v activity: %v error: %v", oid, activityId, err)
			}
		}
	}
	return info, nil
}

func UpdateMatch(ctx *gms.Context) {
	defer util.GetUsedTime("UpdateMatch")()
	usedStart := time.Now()
	var rsp *pb.UpdateMatchResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&UpdateMatchSuccess, 1)
		} else {
			logger.Error("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&UpdateMatchFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf("traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "UpdateMatch", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.UpdateMatchRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.UpdateMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)

	err = activity.UpdateMatch(req.MatchId, req.MatchInfo, req.MatchRule)
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		err = errors.NewFundMgrInternalError(errors.WithMsg("update match: %v error", req.MatchId))
		rsp = &pb.UpdateMatchResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.UpdateMatchResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}

func GetActivityMatchInfo(ctx *gms.Context) {
	defer util.GetUsedTime("GetActivityMatchInfo")()
	usedStart := time.Now()
	var rsp *pb.GetActivityMatchInfoResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success,"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetActivityMatchInfoSuccess, 1)
		} else {
			logger.Error("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetActivityMatchInfoFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "GetActivityMatchInfo", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.GetActivityMatchInfoRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("unmarshal req err: %s", err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetActivityMatchInfoResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)

	// call activity mgr get activity's match item
	var activityInfo *metadata.MatchActivity
	if len(req.ActivityId) != 0 {
		activityRsp, rpcErr := client.QueryActivity(req.ActivityId)
		if rpcErr != nil {
			logger.Error("rpc get activity: %v info, err: %v", req.ActivityId, err)
			rsp = &pb.GetActivityMatchInfoResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		if activityRsp.Header.Code != 0 {
			logger.Error("can not get activity: %v info, header: %v", req.ActivityId, activityRsp.Header)
			rsp = &pb.GetActivityMatchInfoResponse{
				Header: &metadata.CommonHeader{
					Code: errors.FundMgrInternalError,
					Msg:  fmt.Sprintf("can not get activity: %v info", req.ActivityId),
				},
			}
			return
		}
		activityInfo, err = activity.GetMatchEvent(activity.WithEventID(activityRsp.Activity.Activity.MatchId))
	} else {
		activityInfo, err = activity.GetMatchEvent()
	}

	if err != nil {
		if errors.IsFundMgrMatchNotExistError(err) {
			logger.Info("traceID: %v activity: %v has no match item", ctx.Request.TraceId, req.ActivityId)
		} else {
			logger.Error("traceID: %v get activity: %v match item error: %s", ctx.Request.TraceId, req.ActivityId, err.Error())
		}
		rsp = &pb.GetActivityMatchInfoResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.GetActivityMatchInfoResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		ActivityInfo: activityInfo,
	}
}

// SendMsg2KafkaLocal send local kafka client
func SendMsg2KafkaLocal(oid, actID, matchID, matchDate string, fund int64) error {
	msg, err := generateMsg(oid, actID, matchID, matchDate, fund)
	if err != nil {
		return fmt.Errorf("oid: %v, matchID: %v generateMsg error: %v", oid, matchID, err)
	}
	if err = mqclient.KafkaSendMsg(msg); err != nil {
		return fmt.Errorf("oid: %v, matchID: %v KafkaSendMsg error: %v", oid, matchID, err)
	}
	return nil
}

func generateMsg(oid, actID, matchID, matchDate string, fund int64) ([]byte, error) {
	var msg XGYProto.ReportMsg
	msg.MsgType = XGYProto.MSG_TYPE_MSG_TYPE_CUSTOM.Enum()
	metrics := "metrics"
	msg.Metrics = &metrics
	topic := "yqz-act-fund-rank"
	msg.Topic = &topic
	msg.Key = &oid
	nt_date := util.GetLocalFormatTime()
	msg.Dt = &nt_date
	msg.Fields = make([]*XGYProto.KeyVal, 0)
	msg.Fields = mqclient.AddKeySValField(msg.Fields, "busi_type", []byte("yqz_user_match_fund_rank_type"))
	msg.Fields = mqclient.AddKeySValField(msg.Fields, "oid", []byte(oid))
	msg.Fields = mqclient.AddKeySValField(msg.Fields, "actid", []byte(actID))
	msg.Fields = mqclient.AddKeySValField(msg.Fields, "matchID", []byte(matchID))
	msg.Fields = mqclient.AddKeySValField(msg.Fields, "matchDate", []byte(matchDate))
	msg.Fields = mqclient.AddKeyIntValField(msg.Fields, "funds", float64(fund))
	logger.Debug("msg = %+v", msg)
	return proto.Marshal(&msg)
}
