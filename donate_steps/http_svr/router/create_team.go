//

package yqz_router

import (
	"errors"
	"fmt"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/api/metadata"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

func init() {
	route := agw.HandleFunc("/yqz/create-team", handleCreateTeam).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.CreateTeamRequest{})
	if !util.CheckTestMode() {
		route.FilterFunc(gysess.CheckLoginWithoutToken)
	}
}

func checkCreateTeamParam(req *common.CreateTeamRequest) error {
	if 0 == len(req.Aid) {
		log.Error("aid is empty error params = %v", req)
		return errors.New("aid is empty error")
	}

	if 0 == len(req.Team.TeamName) || 0 == len(req.Team.TeamCreator) {
		log.Error("team_name or team_creator is empty error, params = %v", req)
		return errors.New("team_name or team_creator is empty error")
	}

	return nil
}

// 用户创建的小队
func handleCreateTeam(ctx *agw.Context) {
	request := ctx.Input.(*common.CreateTeamRequest)
	request.Team.TeamFlag = int(metadata.Team_USER_TEAM)
	err := checkCreateTeamParam(request)
	if err != nil {
		msg := fmt.Sprintf("checkCreateTeamParam error: %v, param: %v", err, request)
		log.Error("%v", msg)
		ctx.Error(msg)
		ctx.SetResult(common.PARAMS_ERROR, err.Error())
		return
	}

	// 查询定制型活动配置, 先查询活动是否存在
	activityResp, err := activitymgr.QueryActivity(request.Aid)
	if err != nil {
		ctx.Error("activitymgr.QueryActivity error: %v, rsp: %v, param: %v", err, activityResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	if activityResp.Header.Code == -20 {
		ctx.Error("activitymgr.QueryActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	} else if activityResp.Header.Code != 0 {
		ctx.Error("activitymgr.QueryActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// 创建小队
	teamResp, err := activitymgr.CreateTeam(request)
	if err != nil {
		ctx.Error("activitymgr.CreateTeam error: %v, rsp: %v, param: %v", err, teamResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	// 每个用户只能加入一个小队
	if -120003 == teamResp.Header.Code {
		ctx.Error("activitymgr.CreateTeam rsp: %v, param: %v", teamResp, request)
		ctx.SetResult(common.JOINED, common.ErrCodeMsgMap[common.JOINED])
		return
	} else if activityResp.Header.Code == -20 {
		ctx.Error("activitymgr.CreateTeam rsp: %v, param: %v", teamResp, request)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	} else if teamResp.Header.Code != 0 {
		ctx.Error("activitymgr.CreateTeam rsp: %v, param: %v", teamResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	result := common.CreateTeamResponse{Aid: teamResp.Team.ActivityId, TeamId: teamResp.Team.TeamId}

	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(result)
	log.Debug("%v - resp: %+v", util.GetCallee(), result)
}
