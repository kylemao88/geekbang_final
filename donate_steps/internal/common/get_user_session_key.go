//
//

package common

import (
	"context"
	"fmt"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/go_common_v2/util"
)

type UserAdInfo struct {
	SessionKey string `json:"session_key"`
	Nick       string `json:"nick"`
	Head       string `json:"head"`
	Mopenid    string `json:"mopenid"`
}

type WxSessionData struct {
	AdStr string `json:"ad"`
}

func QueryWxSessionData(code string, user_ad_info *UserAdInfo) error {
	key := fmt.Sprintf("wx_session_%s", code)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get wx session data error = %v", key, err)
		return err
	}

	var wx_session_data WxSessionData
	info_str := fmt.Sprintf("%s", string(ckvval))
	if err := json.Unmarshal(ckvval, &wx_session_data); err != nil {
		log.Error("user json parse error, err = %v, json = %s", err, info_str)
		return err
	}

	if err = json.Unmarshal([]byte(wx_session_data.AdStr), user_ad_info); err != nil {
		log.Error("user json parse error, err = %v, json = %s", err, wx_session_data.AdStr)
		return err
	}

	return nil
}

// 上面的是上版本的获取session key的方式，下面是新版本的方式

type UserNickInfo struct {
	SessionKey string `json:"session_key"`
}

func QueryWxSessionKey(appid string, uni_id string) (string, error) {
	key := fmt.Sprintf("%s_%s_nick", appid, uni_id)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get data error, err = %v", key, err)
		return "", err
	}

	var user_data UserNickInfo
	info_str := fmt.Sprintf("{%s}", string(ckvval))
	log.Debug("Session Key appid: %v, user: %v, value: %v", appid, uni_id, info_str)
	if err := json.Unmarshal([]byte(info_str), &user_data); err != nil {
		log.Error("user json parse error, err = %v, json = %s", err, info_str)
		return "", err
	}

	return user_data.SessionKey, nil
}
