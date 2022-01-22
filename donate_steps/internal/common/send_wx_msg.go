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
	"time"
)

type WxSendResp struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func SendWxMsg(appid string, msg string) error {
	ckvval, err := util.GetCkv(context.Background(), "users", appid)
	if err != nil {
		log.Error("ckv key = %s, get token error = %v", appid, err)
		return err
	}

	tokens := strings.Split(string(ckvval), "|")
	if len(tokens) != 6 {
		log.Error("get token error, appid = %s", appid)
		return errors.New("token error")
	}

	ser, err := l5.ApiGetRoute(Yqzconfig.Wxstepsl5config.ModId, Yqzconfig.Wxstepsl5config.CmdId)
	if err != nil {
		log.Error("get wx server ip and port error, err = %s, modid = %d, cmdid = %d",
			err.Error(), Yqzconfig.Wxstepsl5config.ModId, Yqzconfig.Wxstepsl5config.CmdId)
		return err
	}

	url := fmt.Sprintf("http://%s:%d/cgi-bin/message/subscribe/send?access_token=%s",
		ser.Ip(), ser.Port(), tokens[3])

	log.Debug("url = %s, msg = %s", url, msg)

	c := http.Client{Timeout: 5 * time.Second}
	resp, err := c.Post(url, "application/json", strings.NewReader(msg))
	if err != nil {
		log.Error("post wx msg error, err = %v, url = %s, data= %s", err, url, msg)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("wx steps data read error, err = %v", err)
		return err
	}

	var wx_send_resp WxSendResp
	if err := json.Unmarshal(body, &wx_send_resp); err != nil {
		log.Error("wx resp json parse error, err = %v, json = %s", err, string(body))
		return err
	}

	if wx_send_resp.ErrCode != 0 {
		err := errors.New(fmt.Sprintf("wx send error, msg = %s, code = %d", wx_send_resp.ErrMsg, wx_send_resp.ErrCode))
		log.Error(err.Error())
		return err
	}

	return nil
}
