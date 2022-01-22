//
//

package common

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/internal/business_access"
	"git.code.oa.com/gongyi/go_common_v2/util"
	"git.code.oa.com/gongyi/gongyi_base/component/gy_dirtyfilter"
	"git.code.oa.com/gongyi/gongyi_base/component/gy_seq"
)

var Loc *time.Location

func init() {
	Loc, _ = time.LoadLocation("Asia/Shanghai")
}

func TimeSubDays(start_time, end_time time.Time) int {
	start_time = time.Date(start_time.Year(), start_time.Month(), start_time.Day(), 0, 0, 0, 0, time.Local)
	end_time = time.Date(end_time.Year(), end_time.Month(), end_time.Day(), 0, 0, 0, 0, time.Local)

	return int(end_time.Sub(start_time).Hours() / 24)
}

func reverseString(s string) string {
	runes := []rune(s)
	for from, to := 0, len(runes)-1; from < to; from, to = from+1, to-1 {
		runes[from], runes[to] = runes[to], runes[from]
	}
	return string(runes)
}

func ConvertIteration(llsrc int64, base int) string {
	var result string
	var digit []byte = []byte("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

	if base >= 2 && base <= 36 {
		for llsrc > 0 {
			remainder := int(llsrc % int64(base))
			result = string(append([]byte(result), digit[remainder]))
			llsrc /= int64(base)
		}
	}

	return reverseString(result)
}

func UInt64Str2Base36(src string) (dest string, err error) {
	if len(src) == 0 || len(src) > 19 {
		return "", errors.New("invalid src")
	}

	llsrc, parse_err := strconv.ParseInt(src, 10, 64)
	if parse_err != nil {
		log.Error("strconv.ParseInt fail. err = %v", parse_err)
		return "", parse_err
	}

	dest = ConvertIteration(llsrc, 36)
	return dest, nil
}

func GetTransId(prefix string) (transid string, err error) {
	labale, seq, err := gy_seq.GetLabelSeq(0, 0)
	if err != nil {
		log.Error("GetLabelSeq fail.err:[%s]", err)
		return
	}
	curr_dt_stamp := time.Now().Unix()
	// log.Debug("GetLabelSeq:%d	%d	%d\n", labale, seq, curr_dt_stamp)

	uniq_id := fmt.Sprintf("%d%d", labale, seq)
	uniq_id2 := fmt.Sprintf("%d", curr_dt_stamp)

	dest, _ := UInt64Str2Base36(uniq_id)
	// log.Debug("UInt64Str2Base36 labale and seq:%s\n", dest)

	dest2, _ := UInt64Str2Base36(uniq_id2)
	// log.Debug("UInt64Str2Base36 timestamp:%s\n", dest2)

	transid = prefix + dest + dest2
	// log.Debug("UInt64Str2Base36:%s\n", transid)
	return
}

func WeekStartEnd(t time.Time) (time.Time, time.Time) {
	offset := int(time.Monday - t.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekEnd := weekStart.AddDate(0, 0, 6)

	return weekStart, weekEnd
}

func InsertBarrage(eid, oid, text string, check_point, ftype int) error {
	// todo 性能不行的时候，做成异步
	id, err := GetTransId("eb_")
	if err != nil {
		log.Error("GetTransId error, err = %v", err)
		return err
	}
	nt_date := time.Now().Format("2006-01-02 15:04:05")
	event_barrage := data_access.EventBarrage{
		FId:          id,
		FEventId:     eid,
		FCheckPoint:  check_point,
		FUserId:      oid,
		FBarrageText: text,
		FType:        ftype,
		FStatus:      0,
		FCreateTime:  nt_date,
		FModifyTime:  nt_date,
	}
	db_proxy := data_access.DBProxy{
		DBHandler: data_access.DBHandler,
		Tx:        false,
	}
	if err := business_access.InsertEventBarrage(db_proxy, event_barrage); err != nil {
		log.Error("InsertEventBarrage error, err = %v", err)
		return err
	}
	return nil
}

func DatetimeToPeriod(datetime string) string {
	newDate := datetime[:4] + datetime[5:7]
	return newDate
}

/**
获取本周周一的日期
*/
func GetFirstDateOfWeek() (weekMonday string) {
	now := time.Now()

	offset := int(time.Monday - now.Weekday())
	if offset > 0 {
		offset = -6
	}

	weekStartDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, offset)
	weekMonday = weekStartDate.Format("2006-01-02")
	return
}

/**
weekID转换为checkPoint
*/
func WeekID2CheckPoint(week string) int {
	ws := strings.Split(week, "-")
	cps := strings.Join(ws, "")
	cp, _ := strconv.Atoi(cps)
	return cp
}

func WeekByCurYear() int {
	t := time.Now()
	yearDay := t.YearDay()
	yearFirstDay := t.AddDate(0, 0, -yearDay)
	firstDayInWeek := int(yearFirstDay.Weekday())

	//今年第一周有几天
	firstWeekDays := 1
	if firstDayInWeek != 0 {
		firstWeekDays = 7 - firstDayInWeek + 1
	}
	var week int
	if yearDay <= firstWeekDays {
		week = 1
	} else {
		week = (yearDay-firstWeekDays)/7 + 2
	}
	return week
}

type WhiteList struct {
	WhiteArray []string `json:"white_list"`
}

func GetWhiteConfigInfoV2() (map[string]interface{}, error) {
	res := make(map[string]interface{})
	var whiteList WhiteList
	ckvval, err := util.GetCkv(context.Background(), "users", Yqzconfig.CkvKey.WhiteKey)
	if err != nil {
		log.Error("ckv key = %s, get config info error = %s", Yqzconfig.CkvKey.WhiteKey, err.Error())
		return nil, errors.New("get ckv error")
	}

	if err := json.Unmarshal(ckvval, &whiteList); err != nil {
		log.Error("json parse error, err = %v, json = %s", err, string(ckvval))
		return nil, errors.New("parse json error")
	}

	for _, oid := range whiteList.WhiteArray {
		res[oid] = nil
	}

	log.Debug("get white list num(%d)", len(res))
	return res, nil
}

func FilterWord(word string) (int, error) {
	filter, err := gy_dirtyfilter.NewforL5(Yqzconfig.Dirtyconfig.DirtyModid, Yqzconfig.Dirtyconfig.DirtyCmdid)
	if err != nil {
		log.Error("NewforL5 error, err = %v, modid = %v, cmdid = %v",
			err, Yqzconfig.Dirtyconfig.DirtyModid, Yqzconfig.Dirtyconfig.DirtyCmdid)
		return INNTER_ERROR, err
	}
	dirhit, dir_resp, err := filter.Inspect(gy_dirtyfilter.G_DIRTY_APPID, word)
	if err != nil {
		log.Error("dirty error, err = %v, dir_resp = %v, word = %s", err, dir_resp, word)
		return INNTER_ERROR, err
	}

	if len(dirhit) > 0 {
		log.Info("hir dirty eventName, word = %s, dir_resp = %v", word, dir_resp)
		return HAVE_DIRTY, errors.New("hir dirty eventName")
	}

	return 0, nil
}
