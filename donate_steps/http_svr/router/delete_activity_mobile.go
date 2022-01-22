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
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

func init() {
	if util.CheckTestMode() {
		agw.HandleFunc("/yqz/delete-activity-mobile", handleDeleteActivityMobile).
			Author("kylemao").Title("yqz_http_server").RegisterType(&common.DeleteActivityRequest{})
	} else {
		agw.HandleFunc("/yqz/delete-activity-mobile", handleDeleteActivityMobile).
			Author("kylemao").Title("yqz_http_server").RegisterType(&common.DeleteActivityRequest{}).
			FilterFunc(gysess.CheckLogin)
	}
}

func checkDeleteActivityMobileParam(req *common.DeleteActivityRequest) error {

	if len(req.ActivityID) == 0 || len(req.Operator) == 0 {
		log.Error("param error, req = %+v", *req)
		return errors.New("param error")
	}

	return nil
}

func handleDeleteActivityMobile(ctx *agw.Context) {
	request := ctx.Input.(*common.DeleteActivityRequest)
	err := checkDeleteActivityMobileParam(request)
	if err != nil {
		msg := fmt.Sprintf("checkDeleteActivityParam error: %v, param: %v", err, request)
		log.Error("%v", msg)
		ctx.Error(msg)
		ctx.SetResult(common.PARAMS_ERROR, err.Error())
		return
	}

	// 创建定制型活动
	rsp, err := activitymgr.DeleteActivity(request.ActivityID, 2, request.Operator)
	if err != nil || rsp.Header.Code != 0 {
		msg := fmt.Sprintf("activitymgr.DeleteActivity error: %v, rsp: %v, param: %v", err, rsp, request)
		ctx.Error(msg)
		if rsp.Header.Code == -120006 {
			ctx.SetResult(common.OPS_FOBBID, common.ErrCodeMsgMap[common.OPS_FOBBID])
		} else {
			ctx.SetResult(common.INNTER_ERROR, common.ErrCodeMsgMap[common.INNTER_ERROR])
		}
		return
	}

	result := common.DeleteActivityResponse{Result: true}
	ctx.SetResult(common.SUCCESS, "success")
	ctx.SetResultBody(result)
	log.Debug("%v - resp: %+v", util.GetCallee(), result)
}
