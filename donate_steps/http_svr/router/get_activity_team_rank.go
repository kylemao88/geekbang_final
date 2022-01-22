package yqz_router

import (
	"errors"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
)

func init() {
	agw.HandleFunc("/yqz/get-activity-team-rank", handleGetActivityTeamRank).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.GetActivityTeamRankRequest{})
}

func checkGetActivityTeamRankParams(req *common.GetActivityTeamRankRequest) (err error) {
	if len(req.Aid) == 0 || req.Page < 0 || req.Size < 0 {
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

func handleGetActivityTeamRank(ctx *agw.Context) {
	request := ctx.Input.(*common.GetActivityTeamRankRequest)
	err := checkGetActivityTeamRankParams(request)
	if err != nil {
		ctx.Error("err = %v", err)
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	actTeamRank, err := activitymgr.ActivityTeamRank(request.Oid, request.Aid, request.Page, request.Size, request.RankType)
	if err != nil {
		ctx.Error("ActivityTeamRank error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	if actTeamRank.Header.Code != 0 {
		ctx.Error("rpc error, err = %v", actTeamRank.Header.Msg)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	var list []common.ActivityTeamRank
	for i, v := range actTeamRank.List {
		if i >= int(request.Size) {
			continue
		}

		list = append(list, common.ActivityTeamRank{
			Rank:       v.Rank,
			Name:       v.Name,
			Head:       v.Head,
			TotalUsers: v.TotalUsers,
			TotalSteps: v.TotalSteps,
			Aid:        v.Aid,
			TeamId:     v.TeamId,
			TotalFunds: v.TotalFunds,
		})
	}

	resp := common.GetActivityTeamRankResponse{
		UpdateTime: actTeamRank.UpdateTime,
		Total:      actTeamRank.Total,
		List:       list,
		TeamRank:  actTeamRank.TeamRank,
	}

	ctx.SetResultBody(resp)
	ctx.SetResult(common.SUCCESS, "success")

	log.Debug("resp = %+v", resp)
}
