//

package wxclient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"git.code.oa.com/going/l5"
	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/pkg/dbclient"
)

type wxStepsReq struct {
	Appid string   `json:"appid"`
	Oids  []string `json:"openid_list"`
}

type steps struct {
	Timestamp int64 `json:"timestamp"`
	Step      int64 `json:"step"`
}

type wxStepsData struct {
	Openid   string  `json:"openid"`
	Errcode  int     `json:"errcode"`
	StepList []steps `json:"step_list"`
}

type wxStepsRes struct {
	Errcode      int           `json:"errcode"`
	Errmsg       string        `json:"errmsg"`
	UserStepList []wxStepsData `json:"user_step_list"`
}

type userInfo struct {
	Appid string
	Unin  string
}

func GetUserSteps(oid string) (steps map[string]int64, err error) {
	// get wx server rs first
	ser, err := l5.ApiGetRoute(common.Yqzconfig.Wxstepsl5config.ModId, common.Yqzconfig.Wxstepsl5config.CmdId)
	if err != nil {
		log.Error("get wx steps ip and port error, err = %s, modid = %d, cmdid = %d",
			err.Error(), common.Yqzconfig.Wxstepsl5config.ModId, common.Yqzconfig.Wxstepsl5config.CmdId)
		return
	}

	appid, uin, err := getUserWxInfo(oid)
	if err != nil {
		log.Error("getUserWxInfo error, err = %v", err)
		return
	}

	steps, rotten, err := fetchStepFromWx(ser, appid, uin, oid)
	if rotten {
		err = updateUserBackendUpdate(oid, 1)
		if err != nil {
			log.Error("UpdateUserBackendUpdate error, err = %v", err)
		}
	}
	return steps, err
}

func GetUsersSteps(oid []string) (steps map[string]map[string]int64, err error) {
	// get wx server rs first
	ser, err := l5.ApiGetRoute(common.Yqzconfig.Wxstepsl5config.ModId, common.Yqzconfig.Wxstepsl5config.CmdId)
	if err != nil {
		log.Error("get wx steps ip and port error, err = %s, modid = %d, cmdid = %d",
			err.Error(), common.Yqzconfig.Wxstepsl5config.ModId, common.Yqzconfig.Wxstepsl5config.CmdId)
		return
	}
	// get user appid & unin
	users, unins, err := getUsersWxInfo(oid)
	if err != nil {
		log.Error("getUsersWxInfo error, err = %v", err)
		return
	}
	// get steps & rotten user
	steps, rotten, err := fetchUsersStepFromWx(ser, users, unins)
	log.Info("rotten list: %v", rotten)
	return steps, err
}

func generateReqWxStepsData(appid string, oids []string) ([]byte, error) {
	var wx_steps_req wxStepsReq
	wx_steps_req.Appid = appid
	wx_steps_req.Oids = oids
	json_bytes, err := json.Marshal(wx_steps_req)
	if err != nil {
		log.Error("json marshal error, err = %s", err.Error())
		return []byte(""), err
	}
	return json_bytes, nil
}

func updateUserBackendUpdate(oid string, backend_update int) error {
	if len(oid) == 0 {
		return errors.New("oid empty")
	}

	setArgs := make(map[string]interface{})
	whereArgs := make(map[string]interface{})
	setArgs["f_backend_update"] = backend_update
	setArgs["f_modify_time"] = "now()"
	whereArgs["f_user_id"] = oid
	/*
		sql_str := "update t_user set f_backend_update = ?, f_modify_time = now() where f_user_id = ?"
		args := append(args, backend_update)
		args = append(args, oid)
	*/
	defaultClient, err := dbclient.GetDefaultMysqlClient("default")
	if err != nil {
		log.Error("get default client error, err = %v", err)
		return err
	}
	err = defaultClient.Update("t_user_set", setArgs, whereArgs)
	if err != nil {
		log.Error("ExecDB error, err = %v", err)
		return err
	}
	return nil
}

func fetchStepFromWx(ser *l5.Server, appid, uin, oid string) (map[string]int64, bool, error) {
	date_steps := make(map[string]int64)
	wx_req_data, err := generateReqWxStepsData(appid, []string{uin})
	if err != nil {
		return date_steps, false, err
	}

	url := fmt.Sprintf("http://%s:%d/innerapi/appmgrapi/wxaappauth/werun?action=getuserstep&appname=TXGY_Auto_Step",
		ser.Ip(), ser.Port())
	resp, err := client.client.Post(url, "application/json", strings.NewReader(string(wx_req_data)))
	if err != nil {
		log.Error("wx steps send request error, err = %s", err.Error())
		return date_steps, false, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("wx steps data read error, err = %s", err.Error())
		return date_steps, false, err
	}
	//log.Debug("req = %s, url = %s, body = %s", string(wx_req_data), url, string(body))
	var wx_steps_res wxStepsRes
	if err := json.Unmarshal(body, &wx_steps_res); err != nil {
		log.Error("user json parse error, err = %s, json = %s", err.Error(), string(body))
		return date_steps, false, err
	}

	for _, it := range wx_steps_res.UserStepList {
		//log.Debug("oid = %s, wx ret = %+v", oid, it)
		// 出现授权过期或者未授权
		// todo 应该推送一个授权过期的消息给用户, 此处只用list，后续优化
		if it.Errcode == 20070 || it.Errcode == 20071 {
			return date_steps, true, err
		}
		for _, itt := range it.StepList {
			tm := time.Unix(itt.Timestamp, 0)
			date := tm.Format("2006-01-02")
			date_steps[date] = itt.Step
		}
		//log.Debug("get user: %v, step: %v", oid, date_steps)
	}
	return date_steps, false, nil
}

func fetchUsersStepFromWx(ser *l5.Server, user map[string]*userInfo, unins map[string]string) (map[string]map[string]int64, []string, error) {
	date_steps := make(map[string]map[string]int64)
	var uninList []string
	var appid string
	for _, value := range user {
		appid = value.Appid
		uninList = append(uninList, value.Unin)
	}
	wx_req_data, err := generateReqWxStepsData(appid, uninList)
	if err != nil {
		log.Error("generateReqWxStepsData error: %v", err)
		return date_steps, uninList, err
	}

	url := fmt.Sprintf("http://%s:%d/innerapi/appmgrapi/wxaappauth/werun?action=getuserstep&appname=TXGY_Auto_Step",
		ser.Ip(), ser.Port())
	resp, err := client.client.Post(url, "application/json", strings.NewReader(string(wx_req_data)))
	if err != nil {
		log.Error("wx steps send request error, err = %s", err.Error())
		return date_steps, uninList, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("wx steps data read error, err = %s", err.Error())
		return date_steps, uninList, err
	}
	//log.Debug("req = %s, url = %s, body = %s", string(wx_req_data), url, string(body))
	var wx_steps_res wxStepsRes
	if err := json.Unmarshal(body, &wx_steps_res); err != nil {
		log.Error("user json parse error, err = %s, json = %s", err.Error(), string(body))
		return date_steps, uninList, err
	}
	var need_update_users []string
	for _, it := range wx_steps_res.UserStepList {
		oid := unins[it.Openid]
		// log.Debug("oid = %s, wx ret = %+v", oid, it)
		// 出现授权过期或者未授权
		// todo 应该推送一个授权过期的消息给用户, 此处只用list，后续优化
		if it.Errcode == 20070 || it.Errcode == 20071 {
			need_update_users = append(need_update_users, oid)
			log.Info("wx step rotten user: %v", oid)
			continue
		}
		steps := make(map[string]int64)
		for _, itt := range it.StepList {
			tm := time.Unix(itt.Timestamp, 0)
			date := tm.Format("2006-01-02")
			steps[date] = itt.Step
		}
		date_steps[oid] = steps
		//log.Debug("get user: %v, step: %v", oid, date_steps)
	}
	return date_steps, need_update_users, nil
}

func getUserWxInfo(oid string) (appid, uin string, err error) {
	sqlStr := "select f_unin_id, f_appid from t_user " +
		"where f_user_id = ? "
	args := []interface{}{oid}

	// 执行查询
	defaultClient, err := dbclient.GetDefaultMysqlClient("default")
	if err != nil {
		log.Error("get default client error, err = %v", err)
		return "", "", err
	}
	res, err := defaultClient.Query(sqlStr, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return "", "", err
	}
	if len(res) == 0 {
		log.Error("no user: %v appid&uin data now", oid)
		return "", "", fmt.Errorf("no user: %v appid&uin data now", oid)
	}
	return res[0]["f_appid"], res[0]["f_unin_id"], nil
}

func getUsersWxInfo(oid []string) (map[string]*userInfo, map[string]string, error) {
	// check length
	if len(oid) > 50 {
		return nil, nil, fmt.Errorf("oid total: %v, too much", len(oid))
	}
	// get all user wx info
	sqlStr := "select f_user_id, f_unin_id, f_appid from t_user " +
		"where f_user_id in ( "
	args := make([]interface{}, 0)
	for _, it := range oid {
		sqlStr += "?, "
		args = append(args, it)
	}
	sqlStr = strings.TrimRight(sqlStr, ", ")
	sqlStr += ")"

	defaultClient, err := dbclient.GetDefaultMysqlClient("default")
	if err != nil {
		log.Error("get default client error, err = %v", err)
		return nil, nil, err
	}
	res, err := defaultClient.Query(sqlStr, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return nil, nil, err
	}
	if len(res) == 0 {
		log.Error("no user: %v appid&uin data now", oid)
		return nil, nil, fmt.Errorf("no user: %v appid&uin data now", oid)
	}
	infoMap := make(map[string]*userInfo)
	openMap := make(map[string]string)
	for _, it := range res {
		infoMap[it["f_user_id"]] = &userInfo{
			Unin:  it["f_unin_id"],
			Appid: it["f_appid"],
		}
		openMap[it["f_unin_id"]] = it["f_user_id"]
	}
	//log.Debug("infoMap: %v,  openMap: %v", infoMap, openMap)
	return infoMap, openMap, nil
}
