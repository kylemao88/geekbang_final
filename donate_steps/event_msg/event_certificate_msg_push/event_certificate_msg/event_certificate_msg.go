package event_certificate_msg

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
)

func EventCertificateMsgPush(wc common.WeekChallenge) error {
	t, _ := common.WeekStartEnd(time.Now())
	weekID := t.Format("2006-01-02")
	opl, err := getWeekMap(t)
	if err != nil {
		log.Error("getWeekMap error, err = %v", err)
		return err
	}
	mapName := opl.Abbr

	db_proxy := data_access.DBProxy{
		DBHandler: data_access.DBHandler,
		Tx:        false,
	}

	ms, err := business_access.GetMsgStat(weekID, wc.Oid, data_access.MSG_STAT_CERTIFICATE)
	if err != nil {
		log.Error("GetMsgStat error, err = %v, oid = %v", err, wc.Oid)
		return err
	}

	if len(ms.FId) > 0 {
		log.Info("msg pushed oid = %v", wc.Oid)
		return nil
	}
	if common.IsBlack(wc.Oid) == true {
		log.Info("black oid = %v", wc.Oid)
		return nil
	}

	eids, err := business_access.GetUserJoinEvents(wc.Oid)
	if err != nil {
		log.Error("GetUserJoinEvents error, err = %v, oid = %v", err, wc.Oid)
		return err
	}

	nt := time.Now().Format("2006-01-02")
	page := fmt.Sprintf("pages/whole/index/main?weekid=%s&day=%s", weekID, nt)
	if len(eids) > 0 {
		i := rand.Intn(len(eids))
		page = fmt.Sprintf("pages/detail/main?aid=%s&weekid=%s&day=%s", eids[i], weekID, nt)
	}
	var status = 1
	err = pushMsg(mapName, page, wc.Oid, int(wc.Step))
	if err != nil {
		log.Error("pushMsg err = %v", err)
		status = 0
	}

	ms = data_access.MsgStat{
		FId:         weekID,
		FUserId:     wc.Oid,
		FMsgType:    data_access.MSG_STAT_CERTIFICATE,
		FStatus:     status,
		FCreateTime: time.Now().Format("2006-01-02 15:04:05"),
		FModifyTime: time.Now().Format("2006-01-02 15:04:05"),
	}
	err = business_access.InsertMsgStat(db_proxy, ms)
	if err != nil {
		log.Error("InsertMsgStat error, err = %v, ms = %v", err, ms)
	}

	err = business_access.SubMsgStatus(db_proxy, []string{wc.Oid}, common.Certificate_template_id)
	if err != nil {
		log.Error("SubMsgStatus error, err = %v", err)
	}

	return nil
}

func getWeekMap(t time.Time) (common.GlobalWeekLevel, error) {
	var opl common.GlobalWeekLevel
	wm, err := common.GetWeekMap()
	if err != nil {
		log.Error("GetWeekMap error, err = %v", err)
		return opl, err
	}
	orid := wm[t.Format("20060102")].MapId
	/*
		gwl, err := common.GetGlobalWeekLevel()
		if err != nil {
			log.Error("GetGlobalWeekLevel error, err = %v", err)
			return opl, err
		}
		opl = gwl[orid]
	*/
	// get from ckv first
	ridStr := strconv.Itoa(orid)
	opl, err = common.GetSpecifyMapInCKVByID(ridStr)
	if err != nil {
		log.Error("common.GetSpecifyMapInCKVByID rid: %v error: %v", ridStr, err)
		// get from mem second
		opl, err = common.GetSpecifyMapInMem(orid)
		if err != nil {
			log.Error("common.GetSpecifyMapInMem rid: %v error: %v", orid, err)
			return opl, err
		}
	}

	//log.Debug("getWeekMap opl = %+v,wm = %+v", opl, wm)
	return opl, nil
}

func pushMsg(mapName, page, oid string, step int) error {
	unin_id, err := data_access.GetUninid(oid)
	if err != nil {
		log.Error("GetUninid error, err = %v", err)
		return err
	}

	msg, err := common.GenerateCertificate(unin_id, mapName, page, step)
	if err != nil {
		log.Error("GenerateCertificate error, err = %v", err)
		return err
	}

	err = common.SendWxMsg(common.DonateAppId, msg)
	if err != nil {
		log.Error("send wx msg error, err = %v, msg = %s", err, msg)
		return err
	}

	return nil
}
