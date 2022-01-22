//

package personal

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/pkg/dbclient"
	"git.code.oa.com/gongyi/go_common_v2/util"
	"git.code.oa.com/gongyi/gongyi_base/utils/gy_encrypt"
)

// 获取用户nick和head
type NickHeadInfo struct {
	Nick          string `json:"nickname"`
	HeadImgUrl    string `json:"headimgurl"`
	TfsHeadImgUrl string `json:"tfs_headimgurl"`
	MOpenId       string `json:"mopenid"`
	OpenId        string `json:"openid"`
}

const GY_WX_APPID = "wxc0db45f411664b2e"
const WX_APPID = "wxff244f6b82a094d2"

func queryUserInfo(app_id string, uid string, nick_head *NickHeadInfo) error {
	key := fmt.Sprintf("%s_%s_nick", app_id, uid)
	ckvval, err := util.GetCkv(context.Background(), "users", key)
	if err != nil {
		log.Error("ckv key = %s, get user info error = %s", key, err.Error())
		return errors.New("ckv get user info error")
	}

	user_info_str := fmt.Sprintf("{%s}", string(ckvval))
	if err := json.Unmarshal([]byte(user_info_str), nick_head); err != nil {
		log.Error("user json parse error, json = %s", user_info_str)
		return errors.New("parse user info json error")
	}
	return nil
}

func GetNickHead(openid string, nick *string, head *string) error {
	app_id := WX_APPID
	uid := ""
	var err error
	uid, err = GetUninid(openid)
	if err != nil {
		log.Error("GetUninid error, err = %v, oid = %s", err, openid)
		// 因为需要通过小程序id拿用户头像，但是有部分用户没有存小程序id，所以通过这种方式处理一下，下同
		app_id = GY_WX_APPID
		uid = openid
	}

	var nick_head_info NickHeadInfo
	err = queryUserInfo(app_id, uid, &nick_head_info)
	if err != nil {
		log.Error("get nick head error, err = %s", err.Error())
		return errors.New("get nick head error")
	}

	(*nick), err = gy_encrypt.DecodeUrl(nick_head_info.Nick, gy_encrypt.ModeForce)
	if err != nil {
		log.Error("decode url error, err = %v, nick = %s", err, nick_head_info.Nick)
		return err
	}

	if len(nick_head_info.HeadImgUrl) != 0 {
		// 实时查ckv，可以优先使用这个
		(*head) = nick_head_info.HeadImgUrl
	} else {
		(*head) = nick_head_info.TfsHeadImgUrl
	}

	(*head), err = gy_encrypt.DecodeUrl(*head, gy_encrypt.ModeForce)
	if err != nil {
		log.Error("decode url error, err = %v, head = %s", err, *head)
		return err
	}

	return nil
}

type MNickHead struct {
	Nick string `json:"nick"`
	Head string `json:"head"`
}

func MGetNickHead(openids []string) (map[string]MNickHead, error) {
	if len(openids) == 0 {
		return nil, errors.New("openids empty")
	}

	id_map, err := GetUninids(openids)
	if err != nil {
		log.Error("GetUninids error, err = %v, oids = %+v", err, openids)
	}

	var mckv_info util.CkvInfo
	mckv_info.Section = "users"
	mckv_info.Keys = make([]string, 0)
	for _, it := range openids {
		if _, ok := id_map[it]; !ok {
			key := fmt.Sprintf("%s_%s_nick", GY_WX_APPID, it)
			mckv_info.Keys = append(mckv_info.Keys, key)
		} else {
			key := fmt.Sprintf("%s_%s_nick", WX_APPID, id_map[it])
			mckv_info.Keys = append(mckv_info.Keys, key)
		}
	}

	res, _ := util.GetMCkvVals(&mckv_info)
	if mckv_info.Err != nil {
		log.Error("GetMCkvVals error, err = %v", mckv_info.Err)
		return nil, mckv_info.Err
	}

	ret := make(map[string]MNickHead)
	for k, v := range res {
		var nick_head NickHeadInfo
		user_info_str := fmt.Sprintf("{%s}", v.(string))
		err := json.Unmarshal([]byte(user_info_str), &nick_head)
		if err != nil {
			log.Error("user json parse error, err = %v, k = %s, json = %s", err, k, user_info_str)
			return nil, err
		}

		var mnick_head MNickHead
		//mnick_head.Nick, err = gy_encrypt.DecodeUrl(nick_head.Nick, gy_encrypt.ModeForce)
		//if err != nil {
		//	log.Error("decode url error, err = %v, nick = %s", err, mnick_head.Nick)
		//	return nil, err
		//}
		mnick_head.Nick = nick_head.Nick

		if len(nick_head.HeadImgUrl) != 0 {
			mnick_head.Head = nick_head.HeadImgUrl
		} else {
			mnick_head.Head = nick_head.TfsHeadImgUrl
		}

		//mnick_head.Head, err = gy_encrypt.DecodeUrl(mnick_head.Head, gy_encrypt.ModeForce)
		//if err != nil {
		//	log.Error("decode url error, err = %v, head = %s", err, mnick_head.Head)
		//	return nil, err
		//}

		if nick_head.MOpenId != "" {
			ret[nick_head.MOpenId] = mnick_head
		} else {
			ret[nick_head.OpenId] = mnick_head
		}
	}
	return ret, nil
}

func GetUninid(oid string) (string, error) {
	if len(oid) == 0 {
		return "", errors.New("oid empty")
	}
	sql_str := "select f_unin_id from t_user where f_user_id = ? limit 2"
	args := make([]interface{}, 0)
	args = append(args, oid)

	dbClient, err := dbclient.GetDefaultMysqlClient("default")
	if err != nil {
		log.Error("get default client error, err = %v", err)
		return "", err
	}
	// 执行查询
	res, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return "", err
	}
	if len(res) != 1 {
		log.Error("res size error, oid = %s", oid)
		return "", errors.New("res size error")
	}

	return res[0]["f_unin_id"], nil
}

func GetUninids(oids []string) (map[string]string, error) {
	if len(oids) == 0 {
		return nil, errors.New("oid empty")
	}
	sql_str := "select f_user_id, f_unin_id from t_user where f_user_id in ("
	args := make([]interface{}, 0)
	for _, it := range oids {
		sql_str += "?, "
		args = append(args, it)
	}
	sql_str = strings.TrimRight(sql_str, ", ")
	sql_str += ")"

	dbClient, err := dbclient.GetDefaultMysqlClient("default")
	if err != nil {
		log.Error("get default client error, err = %v", err)
		return nil, err
	}
	// 执行查询
	res, err := dbClient.Query(sql_str, args)
	if err != nil {
		log.Error("QueryDB error, err = %v", err)
		return nil, err
	}

	ret := make(map[string]string, len(res))
	for _, it := range res {
		ret[it["f_user_id"]] = it["f_unin_id"]
	}
	return ret, nil
}
