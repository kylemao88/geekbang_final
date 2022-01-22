package data_access

import (
	"errors"
	"fmt"
	"git.code.oa.com/gongyi/agw/log"
	"strconv"
	"strings"
)

type GlobalStep struct {
	UserId        string `json:"f_user_id"`
	RouteId       int    `json:"f_route_id"`
	Period        string `json:"f_period"`
	TotalSteps    int64  `json:"f_total_step"`
	DistancePoint int64  `json:"f_distance_point"`
	JoinDate      string `json:"f_join_date"`
	Status        int    `json:"f_status"`
	CreateTime    string `json:"f_create_time"`
	ModifyTime    string `json:"f_modify_time"`
}

type GlobalStepSlice []GlobalStep

func (p GlobalStepSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p GlobalStepSlice) Len() int           { return len(p) }
func (p GlobalStepSlice) Less(i, j int) bool { return p[i].TotalSteps > p[j].TotalSteps }

func QueryBigMapMyGlobalSteps(pId, myOid string) (*GlobalStep, error) {
	args := make([]interface{}, 0)
	sqlStr := "select f_user_id, f_route_id, f_period, f_total_step, f_distance_point, f_join_date, f_status, f_create_time, f_modify_time " +
		"from t_global_step where f_period = ? and f_user_id = ?"
	args = append(args, pId)
	args = append(args, myOid)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	var globalStep GlobalStep
	globalStep.UserId = myOid /*res[0]["f_user_id"]*/
	if len(res) > 0 {
		globalStep.Status, _ = strconv.Atoi(res[0]["f_status"])
		globalStep.TotalSteps, _ = strconv.ParseInt(res[0]["f_total_step"], 10, 64)
		globalStep.DistancePoint, _ = strconv.ParseInt(res[0]["f_distance_point"], 10, 64)
		globalStep.CreateTime = res[0]["f_create_time"]
		globalStep.JoinDate = res[0]["f_join_date"]
		globalStep.ModifyTime = res[0]["f_modify_date"]
		globalStep.Period = res[0]["f_period"]
	}
	return &globalStep, nil
}

func QueryBigMapStrangerGlobalSteps(pid, myOid string, mySteps int64, nearbyFriends []string) (*GlobalStepSlice, error) {
	args := make([]interface{}, 0)
	ret := make(GlobalStepSlice, 0)
	sqlStr := "select  f_user_id, f_route_id, f_total_step, f_distance_point, f_join_date, f_status, f_create_time, f_modify_time " +
		"from t_global_step where f_period = ? and f_total_step > 0"
	args = append(args, pid)
	if len(nearbyFriends) >= 3 {
		// 3个好友已达到条件，直接返回
		return &ret, nil
	} else if len(nearbyFriends) < 3 {
		sqlStr += " and (f_total_step > ?-50000 or f_total_step < ?+50000) and f_user_id not in( ?,"
		args = append(args, mySteps, mySteps, myOid)
		for _, oid := range nearbyFriends {
			sqlStr += "?,"
			// 以下顺序按照上面的字段的顺序
			args = append(args, oid)
		}
		sqlStr = strings.TrimRight(sqlStr, ",")
		sqlStr += ") limit ?;"
		args = append(args, 3-len(nearbyFriends))
	}
	debugSql := strings.Replace(sqlStr, "?", "%v", -1)
	debugSql = fmt.Sprintf(debugSql, args...)
	log.Debug("exec sql = %s", debugSql)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	for _, it := range res {
		var globalStep GlobalStep
		globalStep.UserId = it["f_user_id"]
		globalStep.Status, _ = strconv.Atoi(it["f_status"])
		globalStep.TotalSteps, _ = strconv.ParseInt(it["f_total_step"], 10, 64)
		globalStep.DistancePoint, _ = strconv.ParseInt(it["f_distance_point"], 10, 64)
		globalStep.CreateTime = it["f_create_time"]
		globalStep.JoinDate = it["f_join_date"]
		globalStep.ModifyTime = it["f_modify_date"]
		globalStep.RouteId, _ = strconv.Atoi(it["f_route_id"])
		ret = append(ret, globalStep)
	}

	return &ret, nil
}


func IsExistInGlobalStep(oid string, pid string) (bool, error) {
	if len(oid) == 0 || len(pid) <= 0 {
		return false, errors.New("event_id empty or date empty")
	}
	sql_str := "select count(*) from t_global_step where f_user_id = ? and f_period = ?"
	args := []interface{}{oid, pid}

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

	size := len(res)
	if size == 1 {
		count, _ := strconv.Atoi(res[0]["count(*)"])
		if count == 1 {
			return true, nil
		} else {
			return false, nil
		}
	} else {
		return false, errors.New("res size error")
	}
}

func IsExistUserInGlobalStep(oid string) (bool, error) {
	if len(oid) == 0 {
		return false, errors.New("oid empty")
	}
	sql_str := "select f_user_id from t_global_step where f_user_id = ? limit 1"
	args := []interface{}{oid}

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

	if len(res) > 0 {
		return true, nil
	}
	return false, nil
}

func GetGlobalStepUserJoinDate(oid string) ([]string, error) {
	if len(oid) == 0 {
		return nil, errors.New("oid empty")
	}
	sql_str := "select f_join_date from t_global_step where f_user_id = ? order by f_join_date desc"
	args := []interface{}{oid}

	// 执行查询
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

	ret := make([]string, len(res))
	for i, it := range res {
		ret[i] = it["f_join_date"]
	}
	return ret, nil
}

func GetCountGloableStepByPeriod(period string) (int64, error) {
	args := make([]interface{}, 0)
	sqlStr := "select count(f_user_id) from t_global_step where f_period = ? "
	args = append(args, period)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	var count int64
	if len(res) > 0 {
		count, _ = strconv.ParseInt(res[0]["count(f_user_id)"], 10, 64)
	}
	return count, nil
}

func GetRankGloableStepByPeriod(step int, period string) (int64, error) {
	args := make([]interface{}, 0)
	sqlStr := "select count(f_user_id) from t_global_step where f_total_step < ? and f_period = ?"
	args = append(args, step)
	args = append(args, period)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	var count int64
	if len(res) > 0 {
		count, _ = strconv.ParseInt(res[0]["count(f_user_id)"], 10, 64)
	}
	return count, nil
}

func GetCountGlobalStep(pid string) (int64, error) {
	args := make([]interface{}, 0)
	sqlStr := "select count(f_user_id) from t_global_step where f_period = ?"
	args = append(args, pid)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	var count int64
	if len(res) > 0 {
		count, _ = strconv.ParseInt(res[0]["count(f_user_id)"], 10, 64)
	}
	return count, nil
}

func GetRandBigMapUserOid(pid string) ([]string, error) {
	args := make([]interface{}, 0)
	sqlStr := "select f_user_id from t_global_step where f_period = ? limit 6"
	args = append(args, pid)

	// 执行查询
	rows, err := QueryDB(sqlStr, args)
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

	ret := make([]string, len(res))
	for i, it := range res {
		ret[i] = it["f_user_id"]
	}
	return ret, nil
}

func GetGlobalStepList(oid string) ([]GlobalStep, error) {
	if len(oid) == 0 {
		return nil, errors.New("oid empty")
	}
	sql_str := "select f_user_id, f_route_id, f_period, f_total_step, f_distance_point, f_join_date, f_status, f_create_time, f_modify_time from t_global_step where f_user_id = ? order by f_create_time desc"
	args := []interface{}{oid}

	// 执行查询
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

	var ret []GlobalStep
	for _, it := range res {
		var globalStep GlobalStep
		globalStep.UserId = it["f_user_id"]
		globalStep.Status, _ = strconv.Atoi(it["f_status"])
		globalStep.TotalSteps, _ = strconv.ParseInt(it["f_total_step"], 10, 64)
		globalStep.DistancePoint, _ = strconv.ParseInt(it["f_distance_point"], 10, 64)
		globalStep.CreateTime = it["f_create_time"]
		globalStep.JoinDate = it["f_join_date"]
		globalStep.ModifyTime = it["f_modify_date"]
		globalStep.RouteId, _ = strconv.Atoi(it["f_route_id"])
		globalStep.Period = it["f_period"]
		ret = append(ret, globalStep)
	}
	return ret, nil
}

func GetGlobalStepListByPeriodOids(period string, oids []string) ([]GlobalStep, error) {
	sql_str := "select f_user_id, f_route_id, f_period, f_total_step, f_distance_point, f_join_date, f_status, f_create_time, f_modify_time from t_global_step " +
		"where f_period = ? and " +
		"f_user_id in ("
	args := make([]interface{}, 0)
	args = append(args, period)

	for _, it := range oids {
		sql_str += "?, "
		args = append(args, it)
	}
	sql_str = strings.TrimRight(sql_str, ", ")
	sql_str += ")"

	// 执行查询
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

	var ret []GlobalStep
	for _, it := range res {
		var globalStep GlobalStep
		globalStep.UserId = it["f_user_id"]
		globalStep.Status, _ = strconv.Atoi(it["f_status"])
		globalStep.TotalSteps, _ = strconv.ParseInt(it["f_total_step"], 10, 64)
		globalStep.DistancePoint, _ = strconv.ParseInt(it["f_distance_point"], 10, 64)
		globalStep.CreateTime = it["f_create_time"]
		globalStep.JoinDate = it["f_join_date"]
		globalStep.ModifyTime = it["f_modify_date"]
		globalStep.RouteId, _ = strconv.Atoi(it["f_route_id"])
		globalStep.Period = it["f_period"]
		ret = append(ret, globalStep)
	}
	return ret, nil
}

func InsertGlobalStep(db_proxy DBProxy, global_step GlobalStep) error {
	if len(global_step.UserId) == 0 || global_step.Period == "" {
		return errors.New("oid empty")
	}

	err := InsertDataToDB(db_proxy, "t_global_step", global_step)
	if err != nil {
		log.Error("InsertDataToDB error, err = %v", err)
		return err
	}

	return nil
}

type OidStep struct {
	Oid  string
	Step int64
}

func GetBigMapDistanceZoneOid(pid string, st, et int64, limit int) ([]OidStep, error) {
	if len(pid) == 0 {
		return nil, IdEmptyError
	}
	sql_str := "select f_user_id, f_total_step from t_global_step " +
		"where f_period = ? and f_total_step >= ? and f_total_step < ? and f_total_step != 0 limit ?"
	args := []interface{}{pid, st, et, limit}

	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return nil, err
	}

	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return nil, err
	}

	var ret []OidStep
	for _, it := range res {
		step, _ := strconv.ParseInt(it["f_total_step"], 10, 64)
		tmp := OidStep{
			Oid:  it["f_user_id"],
			Step: step,
		}
		ret = append(ret, tmp)
	}
	return ret, nil
}

func GetBigMapDistanceZoneNum(pid string, st, et int64) (int, error) {
	if len(pid) == 0 {
		return 0, IdEmptyError
	}
	sql_str := "select count(f_user_id) from t_global_step " +
		"where f_period = ? and f_total_step >= ? and f_total_step < ? and f_total_step != 0"
	args := []interface{}{pid, st, et}

	rows, err := QueryDB(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return 0, err
	}

	res, err := ParseRows(rows)
	if err != nil {
		log.Error("parse rows error, err = %v", err)
		return 0, err
	}

	var num int
	for _, it := range res {
		num, _ = strconv.Atoi(it["count(f_user_id)"])
	}
	return num, nil
}

