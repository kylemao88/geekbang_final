

package statistic

import (
	"errors"
	"strconv"
	"time"

	"git.code.oa.com/gongyi/agw/log"
)

type UserBehavior struct {
	FUserId     string `json:"f_user_id"`
	FIncidentId string `json:"f_incident_id"`
	// 1: get home event; 2: get week team; 3: get pk list
	FTypeId     int    `json:"f_type_id"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

type UserSource struct {
	FUserId string `json:"f_user_id"`
	// 1:pk, 2:big map
	FTypeId     int    `json:"f_type_id"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

type UserTransform struct {
	FUserId string `json:"f_user_id"`
	// 1:pk->big map, 2:big map->pk
	FTypeId     int    `json:"f_type_id"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

type UserDAU struct {
	FUserId     string `json:"f_user_id"`
	VisitTime   int    `json:"f_visit_times"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

type UserWAU struct {
	FUserId     string `json:"f_user_id"`
	FVisitTime  int    `json:"f_visit_times"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

// InsertUserBehavior ...
// params:
// - type_id: 0=visit index 1=donate 2=visit team 3=visit pk list 4=visit pk detail
func InsertUserBehavior(oid string, type_id int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	data := UserBehavior{
		FUserId:     oid,
		FIncidentId: "",
		FTypeId:     type_id,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}
	err := dbClient.Insert("t_user_behavior", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}

// InsertUserIncidentBehavior ...
// params:
// - type_id: 0=visit index 1=donate 2=visit team 3=visit pk list 4=visit pk detail
func InsertUserIncidentBehavior(oid, incident string, type_id int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	data := UserBehavior{
		FUserId:     oid,
		FIncidentId: incident,
		FTypeId:     type_id,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}
	err := dbClient.Insert("t_user_behavior", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}

// InsertUserTransform ...
// params:
// - source: 1=pk->big map, 2=big map->pk
func InsertUserTransform(oid string, source int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	data := UserTransform{
		FUserId:     oid,
		FTypeId:     source,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}

	err := dbClient.Insert("t_user_transform", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}

// InsertUserResource ...
// params:
// - source: 1=pk, 2=big map
func InsertUserResource(oid string, trans int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	data := UserSource{
		FUserId:     oid,
		FTypeId:     trans,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}

	err := dbClient.Insert("t_user_source", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}

// QueryCntUserResourceByDates
// params:
// - date: 2021-06-13
// - source: 1=pk, 2=big map
func QueryCntUserResourceByDate(date string, source int) (int, error) {
	sql_str := "select count(*) from t_user_source where DATE(f_create_time) = ? and f_type_id = ? limit 1"
	args := make([]interface{}, 0)
	args = append(args, date, source)

	// 执行查询
	res, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return 0, err
	}

	total := 0
	if len(res) == 1 {
		total, _ = strconv.Atoi(res[0]["count(*)"])
	}

	return total, nil
}

// IsExistUserResource
// params:
// - oid
// - source: 1=pk, 2=big map
func IsExistUserResource(oid string, source int) (bool, error) {
	if len(oid) == 0 {
		return false, errors.New("oid empty")
	}

	sql_str := "select f_user_id from t_user_source where f_user_id = ? and f_type_id = ? limit 1"
	args := make([]interface{}, 0)
	args = append(args, oid, source)

	// 执行查询
	res, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return false, err
	}

	if len(res) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

// InsertUserDAU ...
// params:
// - times: visit times
func InsertUserDAU(oid string, times int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := time.Now().Format("2006-01-02 15:04:05")
	data := UserWAU{
		FUserId:     oid,
		FVisitTime:  times,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}

	err := dbClient.Insert("t_day_active_user", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}

// InsertUserWAU ...
// params:
// - times: visit times
func InsertUserWAU(oid string, times int) error {
	if len(oid) == 0 {
		return errors.New("id empty error")
	}
	nowTime := getFirstDateOfWeek().Format("2006-01-02 15:04:05")
	data := UserWAU{
		FUserId:     oid,
		FVisitTime:  times,
		FCreateTime: nowTime,
		FModifyTime: nowTime,
	}

	err := dbClient.Insert("t_week_active_user", data)
	if err != nil {
		log.Error("Insert error, err = %v", err)
		return err
	}
	return nil
}
