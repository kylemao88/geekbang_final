//

package yqz_router

import (
	"errors"
	"fmt"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	pb "git.code.oa.com/gongyi/donate_steps/api/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/api/metadata"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
)

func init() {
	agw.HandleFunc("/yqz/get-user-join-activity", handleGetUserJoinActivity).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.GetUserJoinActivityRequest{})
}

func checkUserJoinActivityParam(req *common.GetUserJoinActivityRequest) error {
	if len(req.Oid) == 0 {
		log.Error("params error, params = %v", req)
		return errors.New("params error")
	}
	return nil
}

func handleGetUserJoinActivity(ctx *agw.Context) {
	request := ctx.Input.(*common.GetUserJoinActivityRequest)
	err := checkUserJoinActivityParam(request)
	if err != nil {
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	activityResp := &pb.QueryUserJoinActivityResponse{}

	if 0 == request.Type { // 查询用户创建的活动列表
		activityResp, err = activitymgr.QueryUserCreateActivity(request.Oid, request.Page, request.Size)
		if err != nil {
			msg := fmt.Sprintf("activitymgr.QueryUserCreateActivity error: %v, rsp: %v, param: %v", err, activityResp, request)
			log.Error("%v", msg)
			ctx.Error(msg)
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
			return
		}
		if activityResp.Header.Code == -20 {
			ctx.Error("rpc error, err = %v", activityResp.Header.Msg)
			ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
			return
		}
		if activityResp.Header.Code != 0 {
			ctx.Error("rpc error, err = %v", activityResp.Header.Msg)
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
			return
		}
	} else { // 查询用户参与的活动列表
		activityResp, err = activitymgr.QueryUserJoinActivityNew(request.Oid, request.Page, request.Size)
		if err != nil {
			msg := fmt.Sprintf("activitymgr.QueryUserJoinActivityNew error: %v, rsp: %v, param: %v", err, activityResp, request)
			ctx.Error(msg)
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
			return
		}
		if activityResp.Header.Code == -20 {
			ctx.Error("rpc error, err = %v", activityResp.Header.Msg)
			ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
			return
		}
		if activityResp.Header.Code != 0 {
			ctx.Error("rpc error, err = %v", activityResp.Header.Msg)
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
			return
		}
	}

	results := common.GetUserJoinActivityResponse{List: make([]common.ActivityStats, 0)}
	results.Total = activityResp.Total
	results.HasCreate = activityResp.HasCreate

	for _, item := range activityResp.UserJoinActivities {
		activityInfo := item.CustomizedActivity.Activity
		matchInfo := item.CustomizedActivity.Match

		matchName := activityInfo.OrgName
		matchLogo := activityInfo.OrgHead
		if metadata.Activity_PLATFORM == activityInfo.Type {
			matchName = matchInfo.Company.CompanySn
			matchLogo = matchInfo.Company.CompanyLogo
		}

		// 运营平台发起，给定企业/配捐企业的名称。移动端创建者的昵称
		var sponsorName string
		if activityInfo.Type == metadata.Activity_MOBILE {
			if len(activityInfo.ActivityCreator) > 0 {
				// 获取昵称失败仅打印日志, 不影响主流程?
				var _head string
				if err = common.GetNickHead(activityInfo.ActivityCreator, &sponsorName, &_head); err != nil {
					log.Error("oid = %s, get sponsor nick err = %v", activityInfo.ActivityCreator, err)
				}
			}
		} else if activityInfo.Type == metadata.Activity_PLATFORM {
			// 没有指定企业id，给配捐的企业名称
			sponsorName = matchInfo.Company.CompanySn
			// 给定了企业id
			if len(activityInfo.CompanyId) > 0 {
				var companyInfo common.CKVCompanyInfo
				err = common.QueryCompanyInfo(activityInfo.CompanyId, &companyInfo)
				if err != nil {
					log.Error("CompanyId = %s, QueryCompanyInfo err = %v", activityInfo.CompanyId, err)
				}
				sponsorName = companyInfo.ShortName
			}
		}

		stats := common.ActivityStats{
			Aid:             activityInfo.ActivityId,
			Name:            activityInfo.Desc,
			SponsorName:     sponsorName,
			Bgpic:           activityInfo.Bgpic,
			BgpicStatus:     int(activityInfo.BgpicStatus),
			Type:            int(activityInfo.Type),
			TeamMode:        int(activityInfo.TeamMode),
			Status:          int(activityInfo.Status),
			MatchOff:        int(activityInfo.MatchOff),
			ShowSponsor:     int(activityInfo.ShowSponsor),
			MatchName:       matchName,
			MatchLogo:       matchLogo,
			TeamCnt:         item.ActivityStats.TeamCnt,
			ActivityMember:  item.ActivityStats.ActivityMember,
			ActivitySteps:   item.ActivityStats.ActivitySteps,
			MatchRemainFund: matchInfo.MatchStats.Remain,
			Color:           activityInfo.Color,
			Cover:           activityInfo.Cover,
			RelationDesc:    activityInfo.RelationDesc,
		}
		results.List = append(results.List, stats)
	}

	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(results)
	log.Debug("%v - resp: %+v", util.GetCallee(), results)
}
