//
//

package data_access

import (
	"errors"
	"git.code.oa.com/gongyi/agw/log"
)

type UserStatField struct {
	FEventId      string `json:"f_event_id"`
	FUserId       string `json:"f_user_id"`
	FUserNickname string `json:"f_user_nickname"`
	FUserPic      string `json:"f_user_pic"`
	FStep         int64  `json:"f_step"`
	FFund         int64  `json:"f_fund"`
	FRFund        int64  `json:"f_rfund"`
	FDonation     int64  `json:"f_donation"`
	FRDonation    int64  `json:"f_rdonation"`
	FLike         int64  `json:"f_like"`
	FInvited      int64  `json:"f_invited"`
	FDate         string `json:"f_date"`
	FCreateTime   string `json:"f_create_time"`
	FModifyTime   string `json:"f_modify_time"`
}

type UserStatTotalInfo struct {
	Count       int64
	SumSteps    int64
	SumDonation int64
}


func IsUserDonateByEidUidDate(event_id string, oid string, date string) (bool, error) {
	if len(event_id) == 0 || len(oid) == 0 || len(date) == 0 {
		return false, errors.New("event_id empty or oid empty or date empty")
	}
	sql_str :=
		"select " +
			"f_user_id " +
			"from " +
			"t_user_stat " +
			"where " +
			"f_event_id = ? and f_user_id = ? and f_date = ? limit 2"
	args := make([]interface{}, 0)
	args = append(args, event_id)
	args = append(args, oid)
	args = append(args, date)

	// 执行查询
	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return false, err
	}

	// 解析查询结果
	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return false, err
	}

	if len(res) > 1 {
		log.Error("size error, event_id = %s, oid = %s, date = %s", event_id, oid, date)
		return false, errors.New("size error")
	} else if len(res) == 1 {
		return true, nil
	} else {
		return false, nil
	}
}


