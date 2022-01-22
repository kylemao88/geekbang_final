package yqz_router

import (
	"errors"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
)

const (
	RAND_TYPE_STEP = "step"
	RAND_TYPE_FUND = "fund"
)

func init() {
	agw.HandleFunc("/yqz/get-activity-user-rank", handleGetActivityUserRank).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.GetActivityUserRankRequest{})
}

func checkGetActivityUserRankParams(req *common.GetActivityUserRankRequest) (err error) {
	if 0 == len(req.Aid) || 0 == len(req.Oid) || req.Page < 0 || req.Size < 0 {
		return errors.New("params error")
	}

	if len(req.RankType) == 0 {
		req.RankType = RAND_TYPE_STEP // default
	}

	if req.RankType != RAND_TYPE_STEP && req.RankType != RAND_TYPE_FUND {
		return errors.New("params error")
	}

	return nil
}

func handleGetActivityUserRank(ctx *agw.Context) {
	request := ctx.Input.(*common.GetActivityUserRankRequest)
	err := checkGetActivityUserRankParams(request)
	if err != nil {
		ctx.Error("err = %v", err)
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	actUserRank, err := activitymgr.ActivityUserRank(request.Oid, request.Aid, request.Page, request.Size, request.RankType)
	if err != nil {
		ctx.Error("ActivityUserRank error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	if actUserRank.Header.Code == -20 {
		ctx.Error("rpc error, err = %v", actUserRank.Header.Msg)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	}
	if actUserRank.Header.Code != 0 {
		ctx.Error("rpc error, err = %v", actUserRank.Header.Msg)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	var oids []string
	for i, v := range actUserRank.List {
		if i >= int(request.Size) {
			continue
		}
		oids = append(oids, v.Oid)
	}

	userTeams, err := activitymgr.GetTeamsByUsers(request.Aid, oids)
	if err != nil {
		ctx.Error("GetTeamsByUsers error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	if userTeams.Header.Code == -20 {
		ctx.Error("rpc error, err = %v", userTeams.Header.Msg)
		ctx.SetResult(common.RETRY, common.ErrCodeMsgMap[common.RETRY])
		return
	}
	if userTeams.Header.Code != 0 {
		ctx.Error("rpc error, err = %v", userTeams.Header.Msg)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	var list []common.ActivityUserRank
	for i, v := range actUserRank.List {
		if i >= int(request.Size) {
			continue
		}

		// 暂时每个活动每个人只有一个小队
		var tid, tname string
		if len(userTeams.Teams[v.Oid].List) > 0 {
			tid = userTeams.Teams[v.Oid].List[0].TeamId
			tname = userTeams.Teams[v.Oid].List[0].TeamDesc
		}

		isSelf := false
		if v.Oid == request.Oid {
			isSelf = true
		}
		list = append(list, common.ActivityUserRank{
			Oid:         common.AESEncrypt(v.Oid, common.AesKey),
			Rank:        v.Rank,
			Nick:        v.Nick,
			Head:        v.Head,
			TotalSteps:  v.TotalSteps,
			TodaySteps:  v.TodaySteps,
			DonateMoney: v.DonateMoney,
			IsSelf:      isSelf,
			TeamID:      tid,
			TeamName:    tname,
			TotalFunds:  v.TotalFunds,
		})
	}

	resp := common.GetActivityUserRankResponse{
		UpdateTime: actUserRank.UpdateTime,
		Total:      actUserRank.Total,
		List:       list,
		UserRank:   actUserRank.UserRank,
	}

	ctx.SetResultBody(resp)
	ctx.SetResult(common.SUCCESS, "success")

	log.Debug("resp = %+v", resp)
}
