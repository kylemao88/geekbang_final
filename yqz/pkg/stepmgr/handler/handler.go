//
package handler

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	gms "git.code.oa.com/gongyi/gomore/service"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	pb "git.code.oa.com/gongyi/yqz/api/stepmgr"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/client"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/steps"
	"google.golang.org/protobuf/proto"
)

const (
	DEFAULT_SIZE int = 1
)

var (
	GetUsersStepsSuccess     int64 = 0
	GetUsersStepsFailed      int64 = 0
	SetUsersStepsSuccess     int64 = 0
	SetUsersStepsFailed      int64 = 0
	GetUsersPkProfileSuccess int64 = 0
	GetUsersPkProfileFailed  int64 = 0
	SetUsersPkProfileSuccess int64 = 0
	SetUsersPkProfileFailed  int64 = 0
	GetPkInteractSuccess     int64 = 0
	GetPkInteractFailed      int64 = 0
	SetPkInteractSuccess     int64 = 0
	SetPkInteractFailed      int64 = 0
	SetActivePkSuccess       int64 = 0
	SetActivePkFailed        int64 = 0
	SetResponsePkSuccess     int64 = 0
	SetResponsePkFailed      int64 = 0
	GetPassivePkSuccess      int64 = 0
	GetPassivePkFailed       int64 = 0
	GetPkNotificationSuccess int64 = 0
	GetPkNotificationFailed  int64 = 0
)

func HandleGetUsersSteps(ctx *gms.Context) {
	// defer util.GetUsedTime("HandleGetUsersSteps")()
	usedStart := time.Now()
	var rsp *pb.GetUsersStepsResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetUsersStepsSuccess, 1) //加操作
		} else {
			logger.Error("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			if rsp.Header.Code == errors.StepMgrIncompleteError {
				atomic.AddInt64(&GetUsersStepsSuccess, 1) //加操作
			} else {
				atomic.AddInt64(&GetUsersStepsFailed, 1) //加操作
			}
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "HandleGetUsersSteps", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.GetUsersStepsRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewStepMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetUsersStepsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// handle request
	var result = make(map[string]*metadata.UserSteps)
	if req.RangeFlag {
		if err = checkDate(req.StartTime, req.EndTime); err != nil {
			err = errors.NewStepMgrParamInvalid(errors.WithMsg(err.Error()))
			rsp = &pb.GetUsersStepsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		}
		result, err = steps.GetUsersRangeSteps(req.UserIds, req.StartTime, req.EndTime)
	} else {
		result, err = steps.GetUsersSteps(req.UserIds)
	}
	if err != nil {
		if !errors.IsStepMgrIncompleteError(err) {
			logger.Error("%v - GetUsersSteps error: %v", util.GetCallee(), err)
			err = errors.NewStepMgrInternalError(errors.WithMsg(err.Error()))
			rsp = &pb.GetUsersStepsResponse{
				Header: &metadata.CommonHeader{
					Code: errors.ConvertAPIErrorCode(err),
					Msg:  err.Error(),
				},
			}
			return
		} else {
			rsp = &pb.GetUsersStepsResponse{
				Header: &metadata.CommonHeader{
					Msg:  err.Error(),
					Code: errors.ConvertAPIErrorCode(err),
				},
				UserStep: result,
			}
			return
		}
	}

	// response ...
	rsp = &pb.GetUsersStepsResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		UserStep: result,
	}
}

func checkDate(start, end string) error {
	if len(start) == 0 || len(end) == 0 {
		return fmt.Errorf("start or end is empty")
	}
	startTime, err := time.ParseInLocation("2006-01-02", start, util.Loc)
	if err != nil {
		return fmt.Errorf("start: %v time format is error", start)
	}
	endTime, err := time.ParseInLocation("2006-01-02", end, util.Loc)
	if err != nil {
		return fmt.Errorf("end: %v time format is error", end)
	}
	if endTime.Unix() <= startTime.Unix() {
		return fmt.Errorf("start: %v is smaller than end: %v", start, end)
	}
	return nil
}

func HandleSetUsersSteps(ctx *gms.Context) {
	// defer util.GetUsedTime("HandleSetUsersSteps")()
	usedStart := time.Now()
	var rsp *pb.SetUsersStepsResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("%v - traceID: %v call suc, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&SetUsersStepsSuccess, 1) //加操作
		} else {
			logger.Info("%v - traceID: %v call failed, rsp: %v", util.GetCaller(), ctx.Request.TraceId, rsp)
			atomic.AddInt64(&SetUsersStepsFailed, 1) //加操作
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("%v - ctx.Marshal rsp traceID: %v err: %s", util.GetCaller(), ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "HandleSetUsersSteps", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.SetUsersStepsRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("%v - traceID: %v err: %s", util.GetCallee(), ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewStepMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.SetUsersStepsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	logger.Debug("%v - traceID: %v, svc: %v, req: %v", util.GetCallee(), ctx.Request.TraceId, ctx.Request.Method, req)

	// handle request
	err = steps.SetUsersSteps(req.UserStep, req.Background)
	if err != nil {
		logger.Error("%v - SetUsersSteps error: %v", util.GetCallee(), err)
		err = errors.NewStepMgrInternalError(errors.WithMsg(err.Error()))
		rsp = &pb.SetUsersStepsResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}
	// call activity mgr send user today steps
	if !req.Background {
		for userID, steps := range req.UserStep {
			go client.UpdateActivityRankStats(userID, steps)
			/*
				err := client.UpdateActivityRankStats(userID, steps)
				if err != nil {
					logger.Error("call activity mgr UpdateActivityStats user: %v error: %v", userID, err)
				} else {
					logger.Info("call activity update user: %v success", userID)
				}
			*/
		}
	}
	// response ...
	rsp = &pb.SetUsersStepsResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}
