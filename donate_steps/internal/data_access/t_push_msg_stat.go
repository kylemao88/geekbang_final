//
//

package data_access

import (
	"errors"
	"git.code.oa.com/gongyi/agw/log"
	"strconv"
)

type PushMsgStat struct {
	FId         int    `json:"f_id"`
	FEventId    string `json:"f_event_id"`
	FUserId     string `json:"f_user_id"`
	FMsgType    int    `json:"f_msg_type"`
	FPushData   string `json:"f_push_data"`
	FStatus     int    `json:"f_status"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

func GetPushMsgStatsSize() (int, error) {
	sql_str := "select count(*) from t_push_msg_stat where f_status = 0"
	args := []interface{}{}

	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return 0, err
	}

	// 解析查询结果
	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return 0, err
	}

	count, _ := strconv.Atoi(res[0]["count(*)"])

	return count, nil
}

func GetPushMsgStats(offset int, count int) ([]PushMsgStat, error) {
	sql_str := "select f_id, f_event_id, f_user_id, f_msg_type, f_push_data, f_status " +
		"from t_push_msg_stat where f_status = 0 order by f_modify_time limit ?, ?"
	args := []interface{}{offset, count}

	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return nil, err
	}

	// 解析查询结果
	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return nil, err
	}

	ret := make([]PushMsgStat, len(res))
	for i, it := range res {
		ret[i].FId, _ = strconv.Atoi(it["f_id"])
		ret[i].FEventId = it["f_event_id"]
		ret[i].FUserId = it["f_user_id"]
		ret[i].FMsgType, _ = strconv.Atoi(it["f_msg_type"])
		ret[i].FPushData = it["f_push_data"]
		ret[i].FStatus, _ = strconv.Atoi(it["f_status"])
	}

	return ret, nil
}

type InsertPushMsgData struct {
	FEventId    string `json:"f_event_id"`
	FUserId     string `json:"f_user_id"`
	FMsgType    int    `json:"f_msg_type"`
	FPushData   string `json:"f_push_data"`
	FStatus     int    `json:"f_status"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

func InsertPushMsgStat(db_proxy DBProxy, push_msg_stat InsertPushMsgData) error {
	err := InsertDataToDB(db_proxy, "t_push_msg_stat", push_msg_stat)
	if err != nil {
		log.Error("InsertDataToDB error, err = %v", err)
		return err
	}

	return nil
}

func UpdatePushMsgStatStatus(db_proxy DBProxy, id int, status int) error {
	if id <= 0 {
		return errors.New("oid empty")
	}

	sql_str := "update t_push_msg_stat set f_status = ?, f_modify_time = now() where f_id = ?"
	args := []interface{}{status, id}

	err := ExecDB(db_proxy, sql_str, args)
	if err != nil {
		log.Error("ExecDB error, err = %v", err)
		return err
	}
	return nil
}
