//

package yqz_router

import (
	"errors"
	"fmt"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
)

func init() {
	// 运营平台创建活动, 走L5, 屏蔽登录态
	agw.HandleFunc("/yqz/delete-activity-team-plantform", handleDeleteActivityTeam).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.DeleteActivityTeamRequest{})
}

func checkDeleteActivityTeamParam(req *common.DeleteActivityTeamRequest) error {

	if req.Token != CREATE_TOKEN {
		log.Error("token = %s error, req = %+v", req.Token, *req)
		return errors.New("token error")
	}

	if len(req.ActivityID) == 0 {
		log.Error("param error, req = %+v", *req)
		return errors.New("param error")
	}

	return nil
}

func handleDeleteActivityTeam(ctx *agw.Context) {
	request := ctx.Input.(*common.DeleteActivityTeamRequest)
	err := checkDeleteActivityTeamParam(request)
	if err != nil {
		msg := fmt.Sprintf("checkDeleteActivityTeamParam error: %v, param: %v", err, request)
		log.Error("%v", msg)
		ctx.Error(msg)
		ctx.SetResult(common.PARAMS_ERROR, err.Error())
		return
	}

	// 创建定制型活动
	rsp, err := activitymgr.DeleteActivityTeam(request.ActivityID, request.TeamID)
	if err != nil || rsp.Header.Code != 0 {
		msg := fmt.Sprintf("activitymgr.DeleteActivityTeam error: %v, rsp: %v, param: %v", err, rsp, request)
		ctx.Error(msg)
		if rsp.Header.Code == -120006 {
			ctx.SetResult(common.OPS_FOBBID, common.ErrCodeMsgMap[common.OPS_FOBBID])
		} else {
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		}
		return
	}

	result := common.DeleteActivityTeamResponse{Result: true}
	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(result)
	log.Debug("%v - resp: %+v", util.GetCallee(), result)
}
