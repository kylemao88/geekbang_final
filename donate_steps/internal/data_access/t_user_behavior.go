//
//

package data_access

import "git.code.oa.com/gongyi/agw/log"

type UserBehavior struct {
	FUserId     string `json:"f_user_id"`
	FIncidentId string `json:"f_incident_id"`
	// 1: get home event; 2: get week team; 3: get pk list
	FTypeId     int    `json:"f_type_id"`
	FCreateTime string `json:"f_create_time"`
	FModifyTime string `json:"f_modify_time"`
}

func InsertUserBehavior(db_proxy DBProxy, data UserBehavior) error {
	if len(data.FUserId) == 0 {
		return IdEmptyError
	}

	err := InsertDataToDB(db_proxy, "t_user_behavior", data)
	if err != nil {
		log.Error("InsertDataToDB error, err = %v", err)
		return err
	}
	return nil
}
