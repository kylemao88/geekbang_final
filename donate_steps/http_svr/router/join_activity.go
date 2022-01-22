package yqz_router

import (
	"fmt"
	"git.code.oa.com/gongyi/donate_steps/pkg/wxclient"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/fass/gy_busi/gysess"
)

type oprJoinActivity struct {
	ctx     *agw.Context
	req     *common.JoinActivityRequest
	resp    *common.JoinActivityResponse
	err     error
	retcode int
}

func (o *oprJoinActivity) genErr(ret int, errformat string, a ...interface{}) {
	o.retcode = ret
	o.err = fmt.Errorf(errformat, a)
}

func (o *oprJoinActivity) handleRspHeadCode(code int32, msg, teamid, teamname string) {
	// succ
	if code == 0 {
		o.resp = &common.JoinActivityResponse{}
		o.resp.TeamID = teamid
		o.resp.TeamName = teamname
		return
	}

	// fail
	o.ctx.Error("rpc error, err = %v", msg)

	// 加入活动超过上限
	if code == -120003 { // 用户已经加入了小队
		o.genErr(common.JOINED, "req = %v, join already", o.req)
	} else if code == -120012 {
		o.genErr(common.JOIN_ACTIVITY_LIMIT, "req = %v, join over limit", o.req)
	} else if code == -120013 {
		o.genErr(common.JOIN_TEAM_MEMBER_LIMIT, "req = %v, join team over team_member limit", o.req)
	} else if code == -20 {
		o.genErr(common.RETRY, "rpc error, err = %v", msg)
	} else if code == -11 {
		o.genErr(common.JOIN_ACTIVITY_NO_PERMISSION, "rpc error, err = %v", msg)
	} else if code == -2 {
		o.genErr(common.PARAMS_ERROR, "rpc params error, err = %v", msg)
	} else {
		o.genErr(common.INNTER_ERROR, "rpc error, err = %v", msg)
	}

	return
}

func (o *oprJoinActivity) checkParams() {
	o.req = o.ctx.Input.(*common.JoinActivityRequest)
	if 0 == len(o.req.Aid) || 0 == len(o.req.Oid) || 0 == len(o.req.UniId) || 0 == len(o.req.Appid) {
		o.genErr(common.PARAMS_ERROR, "req = %v, params error", o.req)
	}
	return
}

func (o *oprJoinActivity) getWxPhoneNum() {
	// 上一步出错，即返回
	if o.err != nil {
		return
	}

	if len(o.req.Code) > 0 {
		log.Debug("appid = %v, uniId = %v, code = %v", o.req.Appid, o.req.UniId, o.req.Code)
		phoneInfo, err := wxclient.DecodeWxPhoneInfoNew(o.req.Appid, o.req.Code)
		if err != nil {
			o.genErr(common.INNTER_ERROR, "DecodeWxPhoneInfoNew error, err = %v", err)
			return
		}
		o.req.PhoneNum = phoneInfo.PurePhoneNum
	} else if len(o.req.EncryptedData) > 0 && len(o.req.EncryptedIV) > 0 {
		log.Debug("appid = %v, uniId = %v, edata = %v, eiv = %v", o.req.Appid, o.req.UniId, o.req.EncryptedData, o.req.EncryptedIV)
		phoneInfo, err := wxclient.DecodeWxPhoneInfoOld(o.req.Appid, o.req.UniId, o.req.EncryptedData, o.req.EncryptedIV)
		if err != nil {
			o.genErr(common.INNTER_ERROR, "DecodeWxPhoneInfoOld error, err = %v", err)
			return
		}
		o.req.PhoneNum = phoneInfo.PurePhoneNum
	}
	log.Debug("phoneNum = %v", o.req.PhoneNum)
}

func (o *oprJoinActivity) joinActivity() {
	// 上一步出错，即返回
	if o.err != nil {
		return
	}

	// 加入活动
	if len(o.req.TeamID) == 0 {
		arsp, err := activitymgr.JoinActivity(o.req.Oid, o.req.Aid, o.req.PhoneNum, o.req.Appid, o.req.UniId)
		if err != nil {
			o.genErr(common.INNTER_ERROR, "JoinActivity error, err = %v", err)
			return
		}
		o.handleRspHeadCode(arsp.Header.Code, arsp.Header.Msg, "", "")
	}
	return
}

func (o *oprJoinActivity) joinTeam() {
	// 上一步出错，即返回
	if o.err != nil {
		return
	}

	// 加入/退出小队

	log.Debug("%v - req:%+v", util.GetCallee, o.req)
	trsp, err := activitymgr.JoinTeam(o.req.Oid, o.req.Aid, o.req.TeamID, o.req.PhoneNum, o.req.Join, o.req.Appid, o.req.UniId)
	if err != nil {
		o.genErr(common.INNTER_ERROR, "JoinTeam error, err = %v", err)
		return
	}
	o.handleRspHeadCode(trsp.Header.Code, trsp.Header.Msg, trsp.TeamId, trsp.TeamName)
	return
}

func (o *oprJoinActivity) output() {
	// failed
	if o.err != nil {
		o.ctx.Error(o.err.Error())
		o.ctx.SetResult(o.retcode, common.ErrCodeMsgMap[o.retcode])
		return
	}

	// succ
	o.ctx.SetResultBody(o.resp)
	o.ctx.SetResult(common.SUCCESS, common.ErrCodeMsgMap[common.SUCCESS])

	log.Debug("resp = %+v", o.resp)
	return
}

/////
func init() {
	route := agw.HandleFunc("/yqz/join-activity", handleJoinActivity).
		Author("kylemao").Title("yqz_http_server").RegisterType(&common.JoinActivityRequest{})
	if !util.CheckTestMode() {
		route.FilterFunc(gysess.CheckLoginWithoutToken)
	}
}

func handleJoinActivity(ctx *agw.Context) {

	//
	var o oprJoinActivity = oprJoinActivity{ctx: ctx, req: nil, resp: nil, err: nil, retcode: common.SUCCESS}

	// check
	o.checkParams()

	// wx get phone
	o.getWxPhoneNum()

	// 加入活动
	o.joinActivity()

	// 加入小队
	o.joinTeam()

	//
	o.output()
	return
}
