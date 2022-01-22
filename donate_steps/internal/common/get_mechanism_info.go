//
//

package common

import (
	"context"
	"errors"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/util"
)

type MechanismInfo struct {
	OrgId   string `json:"org_id"`
	OrgName string `json:"org_name"`
	Spid    string `json:"spid"`
	Spname  string `json:"spname"`
	WxSpid  string `json:"wx_spid"`
}

func GetMechanismInfo(key string) (MechanismInfo, error) {
	var ret MechanismInfo
	ckvVal, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get config info error = %v", key, err)
		return ret, errors.New("get ckv error")
	}

	if err := json.Unmarshal(ckvVal, &ret); err != nil {
		log.Error("json parse error, err = %v, json = %s", err, string(ckvVal))
		return ret, errors.New("parse json error")
	}

	log.Debug("get info = %+v", ret)
	return ret, nil
}
