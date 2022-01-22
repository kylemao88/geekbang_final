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
	"git.code.oa.com/gongyi/donate_steps/pkg/statistic"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
)

func init() {
	agw.HandleFunc("/yqz/get-activity", handleGetActivity).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.GetActivityRequest{})
}

func checkGetActivityParam(req *common.GetActivityRequest) error {
	if len(req.Oid) == 0 || len(req.Aid) == 0 {
		log.Error("params error, params = %v", req)
		return errors.New("params error")
	}
	return nil
}

func handleGetActivity(ctx *agw.Context) {
	request := ctx.Input.(*common.GetActivityRequest)
	err := checkGetActivityParam(request)
	if err != nil {
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	// statistic data should not effect normal logical
	err = statistic.InsertUserIncidentBehavior(request.Oid, request.Aid, 6)
	if err != nil {
		log.Error("InsertUserBehavior error: %v", err)
	}

	// 查询用户头像昵称
	var nick, head string
	if err = common.GetNickHead(request.Oid, &nick, &head); err != nil {
		msg := fmt.Sprintf("GetNickHead error: %v, param: %v", err, request)
		log.Error("%v", msg)
		ctx.Error(msg)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// 查询定制型活动配置
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
	}
	if activityResp.Header.Code != 0 {
		ctx.Error("activitymgr.QueryActivity rsp: %v, param: %v", activityResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// 查询活动统计
	statsResp, err := activitymgr.QueryActivityUserStats(request.Aid, request.Oid)
	if err != nil {
		ctx.Error("activitymgr.QueryActivityUserStats error: %v, rsp: %v, param: %v", err, statsResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	if statsResp.Header.Code == -20 {
		ctx.Error("activitymgr.QueryActivityUserStats rsp: %v, param: %v", statsResp, request)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	}
	if statsResp.Header.Code != 0 {
		ctx.Error("activitymgr.QueryActivityUserStats rsp: %v, param: %v", statsResp, request)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// 查询用户参与的小队信息
	team := common.TeamInfo{}
	if len(statsResp.Team.TeamId) > 0 {
		// 查询队长的头像昵称
		var leaderNick string
		var leaderHead string
		if len(statsResp.Team.TeamLeader) > 0 {
			// 获取昵称失败仅打印日志, 不影响主流程?
			if err = common.GetNickHead(statsResp.Team.TeamLeader, &leaderNick, &leaderHead); err != nil {
				log.Error("oid = %s, get nick err = %v", statsResp.Team.TeamLeader, err)
			}
		}

		team.TeamId = statsResp.Team.TeamId
		team.TeamName = statsResp.Team.TeamDesc
		team.TeamCreator = common.AESEncrypt(statsResp.Team.TeamCreator, common.AesKey)
		team.TeamType = int(statsResp.Team.TeamType)
		team.TeamLeaderNick = leaderNick
		team.TeamRank = int(statsResp.TeamStats.TeamRank)
		team.TeamSteps = statsResp.TeamStats.TeamSteps
		team.TeamFunds = statsResp.Team.TeamFunds
	}

	activityMeta := activityResp.Activity.Activity
	matchMeta := activityResp.Activity.Match

	// 查询活动发起人信息（适用于移动端发起）
	sponsorHead := ""
	sponsorNick := ""
	if activityMeta.Type == metadata.Activity_MOBILE &&
		activityMeta.ShowSponsor == 1 &&
		len(activityMeta.ActivityCreator) > 0 {
		// 获取昵称失败仅打印日志, 不影响主流程?
		if err = common.GetNickHead(activityMeta.ActivityCreator, &sponsorNick, &sponsorHead); err != nil {
			log.Error("oid = %s, get sponsor nick err = %v", activityMeta.ActivityCreator, err)
		}
	}

	// 活动型证书
	cert := common.Cert{
		OrgId:        statsResp.Cert.OrgId,
		OrgName:      statsResp.Cert.OrgName,
		OrgSeal:      statsResp.Cert.OrgSeal,
		SerialNumber: statsResp.Cert.SerialNumber,
	}

	// 运营平台发起，给定企业/配捐企业的名称。移动端创建者的昵称
	var sponsorName string
	if activityMeta.Type == metadata.Activity_MOBILE {
		if len(activityMeta.ActivityCreator) > 0 {
			// 获取昵称失败仅打印日志, 不影响主流程?
			var _head string
			if err = common.GetNickHead(activityMeta.ActivityCreator, &sponsorName, &_head); err != nil {
				log.Error("oid = %s, get sponsor nick err = %v", activityMeta.ActivityCreator, err)
			}
		}
	} else if activityMeta.Type == metadata.Activity_PLATFORM {
		// 没有指定企业id，给配捐的企业名称
		sponsorName = matchMeta.Company.CompanySn
		// 给定了企业id
		if len(activityMeta.CompanyId) > 0 {
			var companyInfo common.CKVCompanyInfo
			err = common.QueryCompanyInfo(activityMeta.CompanyId, &companyInfo)
			if err != nil {
				log.Error("CompanyId = %s, QueryCompanyInfo err = %v", activityMeta.CompanyId, err)
			}
			sponsorName = companyInfo.ShortName
		}
	}

	activityInfo := common.ActivityInfo{
		Aid:             activityMeta.ActivityId,
		Name:            activityMeta.Desc,
		SponsorName:     sponsorName,
		Slogan:          activityMeta.Slogan,
		Creator:         common.AESEncrypt(activityMeta.ActivityCreator, common.AesKey),
		StartTime:       activityMeta.StartTime,
		EndTime:         activityMeta.EndTime,
		RouteId:         activityMeta.RouteId,
		Bgpic:           activityMeta.Bgpic,
		Type:            int(activityMeta.Type),
		BgpicStatus:     int(activityMeta.BgpicStatus),
		TeamMode:        int(activityMeta.TeamMode),
		TeamOff:         int(activityMeta.TeamOff),
		TeamMemberLimit: int(activityMeta.TeamMemberLimit),
		Status:          int(activityMeta.Status),
		DefaultTeams:    activityMeta.DefaultTeams,
		OrgName:         activityMeta.OrgName,
		OrgHead:         activityMeta.OrgHead,
		ShowSponsor:     int(activityMeta.ShowSponsor),
		MatchOff:        int(activityMeta.MatchOff),
		Rule:            activityMeta.Rule,
		Color:           activityMeta.Color,
		WhiteType:       int(activityMeta.WhiteType),
		Cover:           activityMeta.Cover,
		RelationDesc:    activityMeta.RelationDesc,
		ForwardPic:      activityMeta.ForwardPic,
	}

	companyInfo := common.CompanyInfo{
		CompanyId:   matchMeta.Company.CompanyId,
		CompanyName: matchMeta.Company.CompanySn,
		CompanyLogo: matchMeta.Company.CompanyLogo,
	}

	//
	var supp_class int64 = 0 // default
	log.Debug("%v - act_createtime:%s, supp_fund_rank_date:%s", util.GetCallee(), activityResp.Activity.Activity.CreateTime, common.Yqzconfig.AccessConf.SuppFundRankDate)
	if activityResp.Activity.Activity.CreateTime > common.Yqzconfig.AccessConf.SuppFundRankDate {
		supp_class = 1 //支持公益金排名
	}
	//
	result := &common.GetActivityResponse{
		Activity:       activityInfo,
		Team:           team,
		Company:        companyInfo,
		Head:           head,
		Nick:           nick,
		SponsorHead:    sponsorHead,
		SponsorNick:    sponsorNick,
		Join:           statsResp.UserStats.Join,
		IsCreator:      statsResp.UserStats.IsCreator,
		UserRank:       statsResp.UserStats.Rank,
		TodaySteps:     statsResp.UserStats.TodaySteps,
		TotalSteps:     statsResp.UserStats.TotalSteps,
		MatchMoney:     statsResp.UserStats.MatchMoney,
		MatchSteps:     statsResp.UserStats.MatchSteps,
		ActivityMember: statsResp.ActivityStats.ActivityMember,
		ActivitySteps:  statsResp.ActivityStats.ActivitySteps,
		TeamCnt:        statsResp.ActivityStats.TeamCnt,
		Cert:           cert,
		Class:          supp_class,
	}

	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(result)
	log.Debug("%v - resp: %+v", util.GetCallee(), result)
}
