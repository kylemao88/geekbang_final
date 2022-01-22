package business_access

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
)

// 用户充值总金额(历史汇总)
func GetTransactionV2RechargeSum(oids []string) (map[string]int64, error) {
	sums, err := data_access.GetTransactionV2RechargeSum(oids)
	if err != nil {
		log.Error("GetTransactionV2RechargeSum error, err = %v, oids = %v", err, oids)
		return sums, err
	}
	return sums, nil
}

func GetTransactionV2FunderId(tid string) (string, string, int, error) {
	id, id3, pid, err := data_access.GetTransactionV2FunderId(tid)
	if err != nil {
		log.Error("GetTransactionV2FunderId error, err = %s, tid = %s", err, tid)
		return "", "", 0, err
	}
	return id, id3, pid, nil
}
