//
package handler

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	gms "git.code.oa.com/gongyi/gomore/service"
	pb "git.code.oa.com/gongyi/yqz/api/fundmgr"
	"git.code.oa.com/gongyi/yqz/api/metadata"
	"git.code.oa.com/gongyi/yqz/pkg/common/errors"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/match"
	"google.golang.org/protobuf/proto"
)

func GetActivityMatchRank(ctx *gms.Context) {
	defer util.GetUsedTime("GetActivityMatchRank")()
	usedStart := time.Now()
	var rsp *pb.GetActivityMatchRankResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetActivityMatchRankSuccess, 1)
		} else {
			logger.Error("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&GetActivityMatchRankFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeRead, "GetActivityMatchRank", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.GetActivityMatchRankRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.GetActivityMatchRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)
	total, rank, err := match.GetActivityMatchRank(req.ActivityId, req.Offset, req.Size)
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		rsp = &pb.GetActivityMatchRankResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.GetActivityMatchRankResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
		UserRank: rank,
		Total:    total,
	}
}

func RemoveUserActivityMatchInfo(ctx *gms.Context) {
	defer util.GetUsedTime("RemoveUserActivityMatchInfo")()
	usedStart := time.Now()
	var rsp *pb.RemoveUserActivityMatchInfoResponse
	defer func() {
		if rsp.Header.Code == errors.Success {
			rsp.Header.Msg = "success"
			logger.Info("traceID: %v call suc, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&RemoveUserActivityMatchInfoSuccess, 1)
		} else {
			logger.Error("traceID: %v call failed, rsp: %v", ctx.Request.TraceId, rsp)
			atomic.AddInt64(&RemoveUserActivityMatchInfoFailed, 1)
		}
		rsp.Header.Msg += fmt.Sprintf(" traceID: %v", ctx.Request.TraceId)
		rsp.Header.OpTime = time.Since(usedStart).Milliseconds()
		err := ctx.Marshal(rsp)
		if err != nil {
			logger.Error("ctx.Marshal rsp traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		}
		logger.API(logger.TypeWrite, "RemoveUserActivityMatchInfo", int(time.Since(usedStart).Microseconds()), strconv.Itoa(int(rsp.Header.Code)))
	}()

	// decode request
	req := &pb.RemoveUserActivityMatchInfoRequest{}
	err := proto.Unmarshal(ctx.Request.Body, req)
	if err != nil {
		msg := fmt.Sprintf("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		logger.Error(msg)
		err = errors.NewFundMgrParamInvalid(errors.WithMsg(msg))
		rsp = &pb.RemoveUserActivityMatchInfoResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
	}
	logger.Debug("traceID: %v, svc: %v, req: %v", ctx.Request.TraceId, ctx.Request.Method, req)
	err = match.RemoveUserActivityMatchInfo(req.ActivityId, req.Oid)
	if err != nil {
		logger.Error("traceID: %v err: %s", ctx.Request.TraceId, err.Error())
		rsp = &pb.RemoveUserActivityMatchInfoResponse{
			Header: &metadata.CommonHeader{
				Code: errors.ConvertAPIErrorCode(err),
				Msg:  err.Error(),
			},
		}
		return
	}

	// response ...
	rsp = &pb.RemoveUserActivityMatchInfoResponse{
		Header: &metadata.CommonHeader{
			Msg:  "success",
			Code: errors.Success,
		},
	}
}


