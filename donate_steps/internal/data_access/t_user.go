//
//

package data_access

import (
	"errors"
	"git.code.oa.com/gongyi/agw/log"
	"strconv"
)

type UserField struct {
	FUserId        string `json:"f_user_id"`
	FUninId        string `json:"f_unin_id"`
	FAppid         string `json:"f_appid"`
	FJoinEvent     int    `json:"f_join_event"`
	FBackendUpdate int    `json:"f_backend_update"`
	FStatus        int    `json:"f_status"`
	FSubTime       string `json:"f_sub_time"`
	FCreateTime    string `json:"f_create_time"`
	FModifyTime    string `json:"f_modify_time"`
}

//-----------------------------------------------------------------
func GetUserInfo(oid string) (UserField, error) {
	var user_field UserField
	if len(oid) == 0 {
		return user_field, errors.New("oid empty")
	}
	sql_str :=
		"select f_user_id, f_appid, f_join_event, f_backend_update, f_status, f_create_time, f_sub_time, f_unin_id from t_user " +
			"where f_user_id = ? limit 2"
	args := make([]interface{}, 0)
	args = append(args, oid)

	// 执行查询
	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return user_field, err
	}

	// 解析查询结果
	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return user_field, err
	}

	if len(res) > 1 {
		log.Error("res size error, oid = %s", oid)
		return user_field, errors.New("res size error")
	}

	for _, it := range res {
		user_field.FUserId = it["f_user_id"]
		user_field.FAppid = it["f_appid"]
		user_field.FJoinEvent, _ = strconv.Atoi(it["f_join_event"])
		user_field.FBackendUpdate, _ = strconv.Atoi(it["f_backend_update"])
		user_field.FStatus, _ = strconv.Atoi(it["f_status"])
		user_field.FCreateTime = it["f_create_time"]
		user_field.FSubTime = it["f_sub_time"]
		user_field.FUninId = it["f_unin_id"]
	}
	return user_field, nil
}

func GetUninid(oid string) (string, error) {
	if len(oid) == 0 {
		return "", errors.New("oid empty")
	}
	sql_str := "select f_unin_id from t_user where f_user_id = ? limit 2"
	args := make([]interface{}, 0)
	args = append(args, oid)

	// 执行查询
	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return "", err
	}

	// 解析查询结果
	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return "", err
	}

	if len(res) != 1 {
		log.Error("res size error, oid = %s", oid)
		return "", errors.New("res size error")
	}

	return res[0]["f_unin_id"], nil
}

func IsInUserTable(oid string) (bool, error) {
	if len(oid) == 0 {
		return false, errors.New("oid empty")
	}
	sql_str := "select f_user_id from t_user where f_user_id = ? limit 2"
	args := make([]interface{}, 0)
	args = append(args, oid)

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
		log.Error("size error, oid = %d", oid)
		return false, errors.New("size error")
	} else if len(res) == 1 {
		return true, nil
	} else {
		return false, nil
	}
}

// todo 以下默认更新一下f_modtify_time
func InsertUserInfo(db_proxy DBProxy, user_info UserField) error {
	if len(user_info.FUserId) == 0 {
		return errors.New("oid empty")
	}

	err := InsertDataToDB(db_proxy, "t_user", user_info)
	if err != nil {
		log.Error("InsertDataToDB error, err = %v", err)
		return err
	}

	return nil
}

func UpdateUserStatus(db_proxy DBProxy, oid string, status int) error {
	if len(oid) == 0 {
		return errors.New("oid empty")
	}

	sql_str := "update t_user set f_status = ?, f_modify_time = now() where f_user_id = ?"
	args := make([]interface{}, 0)
	args = append(args, status)
	args = append(args, oid)

	err := ExecDB(db_proxy, sql_str, args)
	if err != nil {
		log.Error("ExecDB error, err = %v", err)
		return err
	}
	return nil
}
