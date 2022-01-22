//
package yqz_router

import (
	"errors"
	"time"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/donate_steps/pkg/wxclient"
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

func init() {
	route := agw.HandleFunc("/yqz/donate-yqz-steps-h5", handleDonateStepsH5).Author("kylemao").
		Title("yqz_http_server").RegisterType(&common.DonateStepsH5Req{})
	if !util.CheckTestMode() {
		route.FilterFunc(gysess.CheckLoginWithoutToken)
	}
}

func checkDonateStepsH5Parms(req *common.DonateStepsH5Req) (err error) {
	if len(req.Oid) == 0 {
		log.Error("params error, params = %v", req)
		return errors.New("params error")
	}
	return nil
}

func handleDonateStepsH5(ctx *agw.Context) {
	request := ctx.Input.(*common.DonateStepsH5Req)
	err := checkDonateStepsH5Parms(request)
	if err != nil {
		ctx.Error("params error, req = %v", request)
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	dateSteps, err := getH5Steps(request.Oid, request.Code, -29)
	if err != nil {
		if wxclient.IsWXTypeError(err, wxclient.WX_NO_PERM) {
			// need auth
			ctx.Error("getH5Steps auth error: %v", err)
			ctx.SetResult(common.WX_NO_PERM, common.ErrCodeMsgMap[common.WX_NO_PERM])
			return
		}
		// internal error
		ctx.Error("getH5Steps internal error: %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	var nick string
	var head string
	if err := common.GetNickHead(request.Oid, &nick, &head); err != nil {
		ctx.Error("GetNickHead error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	nt := time.Now()
	if err := common.DonateAllSteps(request.Oid, nick, nt, false, dateSteps, common.WithJoinBigMap(!request.NotJoin)); err != nil {
		ctx.Error("DonateAllSteps error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	/*
		start, _ := common.WeekStartEnd(nt)
		week := start.Format("2006-01-02")
		_, step, err := cache_access.GetMyWeekRank(request.Oid, week)
		if err != nil {
			log.Error("GetUserBigMapTotalUser error, err = %v, oid = %s", err, request.Oid)
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
			return
		}
	*/

	todayDate := time.Now().Format("2006-01-02")
	todayStep, ok := dateSteps[todayDate]
	if !ok {
		log.Error("get user: %v date: %v step error", request.Oid, todayStep)
	}

	var donate_steps_res common.DonateStepsH5Res
	donate_steps_res.Oid = common.AESEncrypt(request.Oid, common.AesKey)
	donate_steps_res.Nick = nick
	donate_steps_res.Head = head
	// donate_steps_res.WeekId = week
	// donate_steps_res.WeekStep = step
	donate_steps_res.TodayStep = todayStep

	ctx.Info("resp: %+v", donate_steps_res)
	ctx.SetResultBody(donate_steps_res)
	ctx.SetResult(common.SUCCESS, "success")
}

func getH5Steps(oid, code string, limit int8) (map[string]int64, error) {
	var dateSteps map[string]int64
	var err error
	if len(code) != 0 {
		_, err = wxclient.GetAccessToken(code)
		if err != nil {
			return nil, err
		}
		dateSteps, err = wxclient.FetchStepsByToken(oid, limit)
		if err != nil {
			return nil, err
		}
	} else {
		dateSteps, err = wxclient.FetchStepsByToken(oid, limit)
		if err != nil {
			// retry, no mater what return err
			if wxclient.IsWXTypeError(err, wxclient.WX_NO_PERM) {
				_, err = wxclient.RefreshAccessToken(oid)
				if err != nil {
					return nil, err
				}
				dateSteps, err = wxclient.FetchStepsByToken(oid, limit)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}
	return dateSteps, nil
}
