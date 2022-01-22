//

package yqz_router

import (
	"errors"
	"fmt"
	"time"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/api/metadata"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

// curl -H 'Content-Type: application/json' 'http://9.69.29.152:10012/yqz/create-activity-mobile'   -d '{"activity":{"name":"测试活动5","slogan":"口号555","creator":"oproJj77iceu6FWXf7jCARaRaeSs","start_time":"2021-07-20 12:30:00","end_time":"2022-01-20 12:30:00","route_id":"","team_mode":0,"team_off":0},"match_mobile":{"pid":"12","match_mode":1,"step_quota":6000,"target_fund":1000}}'
func init() {
	// 用户移动端创建活动, 需要登录态
	r := agw.HandleFunc("/yqz/create-activity-mobile", handleCreateActivityMobile).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.CreateActivityMobileRequest{})
	if !util.CheckTestMode() {
		r.FilterFunc(gysess.CheckLogin)
	}
}

func checkCreateActivityMobileParam(req *common.CreateActivityMobileRequest) error {
	if 0 == req.MatchMobile.StepQuota {
		req.MatchMobile.StepQuota = 10000 // 默认值为10000步开始配捐
	}

	if 0 == len(req.Activity.Name) ||
		0 == len(req.Activity.Slogan) ||
		0 == len(req.Activity.Creator) {
		log.Error("name or slogan or creator is empty error, req = %+v", *req)
		return errors.New("name or slogan or creator is empty error")
	}

	if 0 == len(req.Activity.StartTime) ||
		0 == len(req.Activity.EndTime) {
		log.Error("activity time is empty error, req = %+v", *req)
		return errors.New("activity time is empty error")
	}

	if 0 == len(req.MatchMobile.Pid) {
		log.Error("pid is empty error, req = %+v", *req)
		return errors.New("pid is empty error")
	}

	return nil
}

func serialRequest(req *common.CreateActivityMobileRequest) common.CreateActivityPlatformRequest {
	activity := common.ActivityInfo{
		Name:            req.Activity.Name,
		Slogan:          req.Activity.Slogan,
		OrgName:         req.Activity.OrgName,
		OrgHead:         req.Activity.OrgHead,
		RouteId:         req.Activity.RouteId,
		Creator:         req.Activity.Creator,
		StartTime:       req.Activity.StartTime,
		EndTime:         req.Activity.EndTime,
		Type:            1, // 0表示运营平台发起,1表示手机端发起
		TeamMode:        req.Activity.TeamMode,
		TeamOff:         req.Activity.TeamOff,
		TeamMemberLimit: 0, // 0表示小队人数无上限
		ShowSponsor:     req.Activity.ShowSponsor,
		Bgpic:           req.Activity.Bgpic,
		Color:           req.Activity.Color,
		CreateTime:      time.Now().Format("2006-01-02 15:04:05"),
	}
	match := common.MatchInfo{
		MatchMode:  req.MatchMobile.MatchMode,      // 配捐模式 0表示平均配捐（小池子每日上限） 1表示累计配捐（大池子）
		RuleType:   int(metadata.MatchRule_REMAIN), // 移动端发起的活动固定为剩余配捐规则
		StepQuota:  req.MatchMobile.StepQuota,
		TargetFund: req.MatchMobile.TargetFund,
		Pid:        req.MatchMobile.Pid,
	}

	return common.CreateActivityPlatformRequest{
		Oid:      req.Oid,
		Appid:    req.Appid,
		UniId:    req.UniId,
		Activity: activity,
		Match:    match,
	}
}

func handleCreateActivityMobile(ctx *agw.Context) {
	request := ctx.Input.(*common.CreateActivityMobileRequest)
	log.Debug("request = %+v", request)

	err := checkCreateActivityMobileParam(request)
	if err != nil {
		msg := fmt.Sprintf("checkCreateActivityMobileParam error: %v, param: %v", err, request)
		log.Error("%v", msg)
		ctx.Error(msg)
		ctx.SetResult(common.PARAMS_ERROR, err.Error())
		return
	}

	// 创建定制型活动
	platReq := serialRequest(request)
	activityResp, err := activitymgr.CreateActivity(&platReq)
	if err != nil {
		ctx.Error("activitymgr.CreateActivity error: %v, rsp: %v, param: %v", err, activityResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	// 创建活动超过上限
	if activityResp.Header.Code == -120007 {
		ctx.Error("activitymgr.CreateActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.CREATE_ACTIVITY_LIMIT, common.ErrCodeMsgMap[common.CREATE_ACTIVITY_LIMIT])
		return
	} else if activityResp.Header.Code == -20 {
		ctx.Error("activitymgr.CreateActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	} else if activityResp.Header.Code != 0 {
		ctx.Error("activitymgr.CreateActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	result := common.CreateActivityMobileResponse{Aid: activityResp.Activity.Activity.ActivityId}

	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(result)
	log.Debug("%v - resp: %+v", util.GetCallee(), result)
}
