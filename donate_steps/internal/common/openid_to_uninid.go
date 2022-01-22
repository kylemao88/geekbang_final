//
//


package common

import (
	"context"
	"errors"
	"fmt"
	"git.code.oa.com/going/l5"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/util"
	"io/ioutil"
	"net/http"
	"strings"
)

const DonateAppId = "wxff244f6b82a094d2"

type OpenidToUninidReq struct {
	TargetAppId string `json:"target_app_id"`
	AppName     string `json:"appname"`
	ThirdOpenId string `json:"third_open_id"`
}

type OpenidToUninidResp struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	OpenId string `json:"open_id"`
}

func OpenidToUninid(oid string) (string, error) {
	req := OpenidToUninidReq{
		TargetAppId: "tencentgy_xiaochengxu",
		AppName:     DonateAppId,
		ThirdOpenId: oid,
	}

	ser, err := l5.ApiGetRoute(1966080, 1303233)
	if err != nil {
		log.Error("get wx server ip and port error, err = %s, modid = 1966080, cmdid = 1303233", err.Error())
		return "", err
	}

	req_str, err := json.Marshal(req)
	if err != nil {
		log.Error("json marshal error, err = %v, req = %v", err, req)
		return "", err
	}

	url := fmt.Sprintf("http://%s:%d/", ser.Ip(), ser.Port())
	resp, err := http.Post(url, "application/json", strings.NewReader(string(req_str)))
	defer resp.Body.Close()
	if err != nil {
		log.Error("post wx msg error, err = %v, url = %s, data= %s", err, url, string(req_str))
		return "", err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("data read error, err = %v", err)
		return "", err
	}

	var ser_resp OpenidToUninidResp
	err = json.Unmarshal(body, &ser_resp)
	if err != nil {
		log.Error("json unmarshal error, err = %v, json = %s", err, string(body))
		return "", err
	}

	if ser_resp.Code != 0 {
		log.Error("resp error, err = %v", ser_resp)
		return "", errors.New("resp error")
	}

	return ser_resp.OpenId, nil
}

type OpenRes struct {
	OpenId string `json:"openid"`
}

func GetUninOidFromCkv(oid string) (string, error) {
	key := fmt.Sprintf("%s_%s", DonateAppId, oid)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get unin error = %v", key, err)
		return "", err
	}

	var open_res OpenRes
	err = json.Unmarshal(ckvval, &open_res)
	if err != nil {
		log.Error("json unmarshal error, err = %v, json = %s", err, string(ckvval))
		return "", err
	}

	return open_res.OpenId, nil
}
