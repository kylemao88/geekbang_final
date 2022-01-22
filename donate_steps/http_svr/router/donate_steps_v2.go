//
//

package yqz_router

import (
	"errors"
	"time"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/statistic"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

func init() {
	route := agw.HandleFunc("/yqz/donate-yqz-steps-v2", handleDonateStepsV2).Author("winterfeng").
		Title("yqz_http_server").RegisterType(&common.DonateStepsReq{})
	if !util.CheckTestMode() {
		route.FilterFunc(gysess.CheckLogin)
	}
}

func checkDonateStepsParmsV2(req *common.DonateStepsReq) (err error) {
	if len(req.Oid) == 0 || len(req.Appid) == 0 || len(req.EcryptedIv) == 0 || len(req.EcryptedData) == 0 || len(req.UniId) == 0 {
		log.Error("params error, params = %v", req)
		return errors.New("params error")
	}
	return nil
}

func updateUserStatistic(oid string, nt time.Time) {
	var err error
	exist := false
	s, _ := common.WeekStartEnd(nt)
	startDate := s.Format("2006-01-02")

	// 检查用户是否在大地图 redis
	if exist, err = cache_access.IsInBigMap(oid, startDate); err != nil {
		log.Error("IsInBigMap error, err = %v, oid = %s, week = %s", err, oid, startDate)
		return
	}
	if !exist {
		// 检查用户是否在大地图 db
		if exist, err = business_access.IsExistInGlobalStep(oid, startDate); err != nil {
			log.Error("IsExistInGlobalStep error, err = %v, oid = %s, week = %s", err, oid, startDate)
			return
		}
		if !exist {
			if inUser, _ := statistic.IsExistUserResource(oid, 1); inUser {
				var _ = statistic.InsertUserTransform(oid, 1)
			}
		}
	}
}

func handleDonateStepsV2(ctx *agw.Context) {
	request := ctx.Input.(*common.DonateStepsReq)
	err := checkDonateStepsParmsV2(request)
	if err != nil {
		ctx.Error("params error, req = %v", request)
		ctx.SetResult(common.PARAMS_ERROR, "params error")
		return
	}

	dateSteps, err := common.DecodeWxSteps(request.Appid, request.UniId, request.EcryptedData, request.EcryptedIv)
	if err != nil {
		ctx.Error("DecodeWxSteps error, err = %v", err)
		ctx.SetResult(common.SESSION_CODE, common.ErrCodeMsgMap[common.SESSION_CODE])
		return
	}

	// 插入一条记录，保证捐步的时候一定能查到对应user表数据
	if err := business_access.InsertUnionId(request.Oid, request.UniId); err != nil {
		ctx.Error("InsertUnionId, err = %s, oid = %s, union_id = %s", err.Error(), request.Oid, request.UniId)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// get head
	var nick string
	var head string
	if err := common.GetNickHead(request.Oid, &nick, &head); err != nil {
		ctx.Error("GetNickHead error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}

	// add user source ops, not trigger frequently
	nt := time.Now()
	updateUserStatistic(request.Oid, nt)
	// main donate logical
	if err := common.DonateAllSteps(request.Oid, nick, nt, false, dateSteps, common.WithJoinBigMap(!request.NotJoin)); err != nil {
		ctx.Error("DonateAllSteps error, err = %v", err)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	// get cache from redis
	start, _ := common.WeekStartEnd(nt)
	week := start.Format("2006-01-02")
	_, step, err := cache_access.GetMyWeekRank(request.Oid, week)
	if err != nil {
		log.Error("GetUserBigMapTotalUser error, err = %v, oid = %s", err, request.Oid)
		ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		return
	}
	// get today step
	todayDate := time.Now().Format("2006-01-02")
	todayStep, ok := dateSteps[todayDate]
	if !ok {
		log.Error("get user: %v date: %v step error", request.Oid, todayStep)
	}

	var donate_steps_res common.DonateStepsRes
	donate_steps_res.Oid = common.AESEncrypt(request.Oid, common.AesKey)
	donate_steps_res.Nick = nick
	donate_steps_res.Head = head
	donate_steps_res.WeekId = week
	donate_steps_res.WeekStep = step
	donate_steps_res.TodayStep = todayStep

	ctx.SetResultBody(donate_steps_res)
	ctx.SetResult(common.SUCCESS, "success")
	log.Debug("resp = %+v", donate_steps_res)
}
