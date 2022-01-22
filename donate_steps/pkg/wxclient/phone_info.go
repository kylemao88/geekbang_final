package wxclient

/*
#cgo CFLAGS: -I../../common/wxlib/src -I../../common/wxlib/lib/include64
#cgo LDFLAGS: -L../../common/wxlib/src -lwxencrypt -L../../common/wxlib/lib/lib64 -lssl -L../../common/wxlib/lib/lib64 -lcrypto -lstdc++
#include "WXWapper.h"
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/go_common_v2/util"
	"io/ioutil"
	"strings"
	"unsafe"
)

type WxPhoneInfo struct {
	PhoneNum     string `json:"phoneNumber"`     // 用户绑定的手机号（国外手机号会有区号）
	PurePhoneNum string `json:"purePhoneNumber"` // 没有区号的手机号
	CountryCode  string `json:"countryCode"`     // 区号
}

// 旧版微信活动手机号码
// https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/deprecatedGetPhoneNumber.html

func decodeWxPhoneInfoOld(appid string, session_key string, en_data string, iv_data string) (WxPhoneInfo, error) {
	var ret WxPhoneInfo
	var ret_size int
	var ret_code int

	c_appid := C.CString(appid)
	defer C.free(unsafe.Pointer(c_appid))
	c_session_key := C.CString(session_key)
	defer C.free(unsafe.Pointer(c_session_key))
	c_en_data := C.CString(en_data)
	defer C.free(unsafe.Pointer(c_en_data))
	c_iv_data := C.CString(iv_data)
	defer C.free(unsafe.Pointer(c_iv_data))

	decode_data := C.DecryptData(c_appid, c_session_key, c_en_data, c_iv_data,
		(*C.int)(unsafe.Pointer(&ret_size)), (*C.int)(unsafe.Pointer(&ret_code)))
	defer C.free(unsafe.Pointer(decode_data))

	if ret_code != 0 {
		log.Error("decode wx data error, code = %d", ret_code)
		return ret, errors.New("decode wx data error")
	}
	tmp_decode_data := []byte(C.GoStringN(decode_data, (C.int)(ret_size)))
	if err := json.Unmarshal(tmp_decode_data, &ret); err != nil {
		log.Error("user json parse error, err = %s, json = %s", err.Error(), string(tmp_decode_data))
		return ret, errors.New("parse wx session data json error")
	}

	return ret, nil
}

func DecodeWxPhoneInfoOld(appid, unin_id, edata, eiv string) (WxPhoneInfo, error) {
	var wxPhoneInfo WxPhoneInfo
	skey, err := common.QueryWxSessionKey(appid, unin_id)
	if err != nil {
		log.Error("QueryWxSessionKey error, err = %v, appid = %s, uni_id = %s", err, appid, unin_id)
		return wxPhoneInfo, err
	}
	wxPhoneInfo, err = decodeWxPhoneInfoOld(appid, skey, edata, eiv)
	if err != nil {
		log.Error("decodeWxPhoneInfo error, err = %v", err)
		return wxPhoneInfo, err
	}

	return wxPhoneInfo, nil
}

// 新版微信获取手机号码
// https://developers.weixin.qq.com/miniprogram/dev/api-backend/open-api/phonenumber/phonenumber.getPhoneNumber.html

type wxPhoneInfoNewResp struct {
	ErrCode     int         `json:"errcode"`
	ErrMsg      string      `json:"errmsg"`
	WxPhoneInfo WxPhoneInfo `json:"phone_info"`
}

func GetWxAccessToken(appid string) (string, error) {
	key := fmt.Sprintf("%s", appid)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get data error, err = %v", key, err)
		return "", err
	}

	strs := strings.Split(string(ckvval), "|")
	if len(strs) < 6 {
		return "", errors.New("data length is short")
	}

	return strs[3], nil
}

func DecodeWxPhoneInfoNew(appid, code string) (WxPhoneInfo, error) {
	var wxPhoneInfo WxPhoneInfo
	accessToken, err := GetWxAccessToken(appid)
	if err != nil {
		log.Error("GetWxAccessToken error, err = %v, appid = %s", err, appid)
		return wxPhoneInfo, err
	}

	ser, err := getWxL5(client.config.ModIDH5, client.config.CmdIDH5)
	if err != nil {
		return wxPhoneInfo, err
	}

	reqData := fmt.Sprintf(`{"code":"%s"}`, code)

	url := fmt.Sprintf("http://%s:%d/wxa/business/getuserphonenumber?access_token=%s",
		ser.Ip(), ser.Port(), accessToken)
	log.Info("DecodeWxPhoneInfoNew req url: %v", url)
	resp, err := client.client.Post(url, "application/json", strings.NewReader(reqData))
	if err != nil {
		log.Error("Get url: %v rpc err: %s", url, err.Error())
		return wxPhoneInfo, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ReadAll url: %v read err: %s, rsp: %v", url, err.Error(), resp.Body)
		return wxPhoneInfo, err
	}

	var ret wxPhoneInfoNewResp
	if err := json.Unmarshal(body, &ret); err != nil {
		log.Error("Unmarshal json parse error, err = %s, json = %s", err.Error(), string(body))
		return wxPhoneInfo, err
	}

	if ret.ErrCode != 0 {
		err := errors.New(fmt.Sprintf("wx send error, msg = %s, code = %d", ret.ErrMsg, ret.ErrCode))
		log.Error("DecodeWxPhoneInfoNew err = %v", err)
		return wxPhoneInfo, err
	}

	return ret.WxPhoneInfo, nil
}
