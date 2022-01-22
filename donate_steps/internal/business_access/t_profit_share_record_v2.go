//
//

package business_access

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
)

func InsertProfitShareRecordV2(db_proxy data_access.DBProxy, record data_access.ProfitShareRecordV2) error {
	if err := data_access.InsertProfitShareRecordV2(db_proxy, record); err != nil {
		log.Error("InsertProfitShareRecordV2 error, err = %v, record = %+v", err, record)
		return err
	}
	return nil
}

func UpdateProfitShareStatus(db_proxy data_access.DBProxy, id string, status int) error {
	if err := data_access.UpdateProfitShareStatus(db_proxy, id, status); err != nil {
		log.Error("UpdateProfitShareStatus error, err = %v, id = %s, status = %d", err, id, status)
		return err
	}
	return nil
}

func IsProfitShareDone(rid string, actionType int) (bool, error) {
	done, err := data_access.IsProfitShareDone(rid, actionType)
	if err != nil {
		log.Error("IsProfitShareDone error, err = %v, tid = %s, type = %d", err, rid, actionType)
		return false, err
	}
	return done, nil
}

func GetProfitShareAccount(rid string) (string, error) {
	account, err := data_access.GetProfitShareAccount(rid)
	if err != nil {
		log.Error("GetProfitShareAccount error, err = %v, rid = %s", err, rid)
		return "", err
	}
	return account, nil
}

func IsTransferDone(rid string) (bool, error) {
	done, err := data_access.IsTransferDone(rid)
	if err != nil {
		log.Error("IsTransferDone error, err = %v, tid = %s", err, rid)
		return false, err
	}
	return done, err
}
