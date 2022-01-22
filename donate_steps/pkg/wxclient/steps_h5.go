//
package wxclient

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"git.code.oa.com/gongyi/agw/json"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"github.com/go-redis/redis"
)

const REDIS_TOKEN_KEY = "yqz:step:h5"
const TOKEN_EXPIRE_TIME = 3600 * 24

type AccessTokenInfo struct {
	AccessToken  string `json:"access"`
	RefreshToken string `json:"refresh"`
	ExpiresIn    int64  `json:"expire"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
}

type getAccessResponse struct {
	Errcode      int    `json:"errcode"`
	Errmsg       string `json:"errmsg"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ExpiresIn    int64  `json:"expires_in"`
}

type wxStepsH5Data struct {
	Errcode  int       `json:"errcode"`
	StepList []stepsH5 `json:"list"`
}

type stepsH5 struct {
	Timestamp int64 `json:"time"`
	Step      int64 `json:"step"`
	RealTime  int64 `json:"real_time"`
}

func (info *AccessTokenInfo) String() string {
	return fmt.Sprintf("{access_token: %v refresh_token: %v, expire_in: %v, openid: %v, scope: %v}",
		info.AccessToken, info.RefreshToken, info.ExpiresIn, info.OpenID, info.Scope)
}

/*
 * 调用URL：
 * https://api.weixin.qq.com/sns/oauth2/access_token?appid=APPID&secret=SECRET&code=CODE&grant_type=authorization_code
 * 出错时返回：{"errcode":40029,"errmsg":"invalid code"}
 * 正常时返回：
{
   "access_token":"ACCESS_TOKEN",
   "expires_in":7200,
   "refresh_token":"REFRESH_TOKEN",
   "openid":"OPENID",
   "scope":"SCOPE"
}
*/
// 如果得到code？在微信中点击运行下面这个URL即可：
// https://ssl.gongyi.qq.com/m/weixin/test22_test.htm

// GetAccessToken get access/refresh token by user code
func GetAccessToken(code string) (result *AccessTokenInfo, err error) {
	// avoid panic because call third party
	defer func() {
		if r := recover(); r != nil {
			log.Error("GetAccessToken Recovered in %v", r)
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()
	result = &AccessTokenInfo{}
	ser, err := getWxL5(client.config.ModIDH5, client.config.CmdIDH5)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s:%d/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code&simple_get_token=1",
		ser.Ip(), ser.Port(), client.config.AppID, client.config.AppSecret, code)
	log.Info("GetAccessToken req url: %v", url)
	resp, err := client.client.Get(url)
	if err != nil {
		log.Error("GetAccessToken url: %v rpc err: %s", url, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("GetAccessToken url: %v read err: %s, rsp: %v", url, err.Error(), resp.Body)
		return nil, err
	}

	rspResult := &getAccessResponse{}
	if err := json.Unmarshal(body, &rspResult); err != nil {
		log.Error("GetAccessToken json parse error, err = %s, json = %s", err.Error(), string(body))
		return nil, err
	}

	if rspResult.Errcode != 0 {
		log.Error("GetAccessToken error code: %v, rsp: %v", rspResult.Errcode, rspResult)
		if rspResult.Errcode == 40029 || rspResult.Errcode == 40163 {
			return nil, fmt.Errorf("GetAccessToken error code: %v", WX_NO_PERM)
		}
		return nil, fmt.Errorf("GetAccessToken error code: %v", WX_FAILED)
	}

	if len(rspResult.AccessToken) == 0 {
		log.Error("GetAccessToken AccessToken is empty, rsp: %v", rspResult)
		return nil, errors.New("GetAccessToken AccessToken is empty")
	}
	result.AccessToken = rspResult.AccessToken
	result.RefreshToken = rspResult.RefreshToken
	result.ExpiresIn = rspResult.ExpiresIn
	result.OpenID = rspResult.OpenID
	result.Scope = rspResult.Scope
	log.Info("GetAccessToken success, get token: %v by code: %v", result, code)
	err = saveToken(result)
	if err != nil {
		log.Error("GetAccessToken save oid: %v token info error: %v", result.OpenID, err)
	}

	return
}

// RefreshAccessToken refresh access token by refresh token
func RefreshAccessToken(oid string) (result *AccessTokenInfo, err error) {
	// avoid panic because call third party
	defer func() {
		if r := recover(); r != nil {
			log.Error("RefreshAccessToken Recovered in %v", r)
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()
	info, err := getToken(oid)
	if err != nil {
		log.Error(fmt.Sprintf("RefreshAccessToken error: %v", err.Error()))
		if err == redis.Nil {
			return nil, fmt.Errorf("RefreshAccessToken nil token info error, code: %v, error: %v", WX_NO_PERM, err)
		}
		return nil, fmt.Errorf("RefreshAccessToken get token info error, code: %v, error: %v", WX_FAILED, err)
	}
	refreshToken := info.RefreshToken
	result = &AccessTokenInfo{}
	ser, err := getWxL5(client.config.ModIDH5, client.config.CmdIDH5)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("http://%s:%d/sns/oauth2/refresh_token?appid=%s&grant_type=refresh_token&refresh_token=%s&simple_get_token=1",
		ser.Ip(), ser.Port(), client.config.AppID, refreshToken)
	log.Info("RefreshAccessToken req url: %v", url)
	resp, err := client.client.Get(url)
	if err != nil {
		log.Error("RefreshAccessToken url: %v rpc err: %s", url, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("RefreshAccessToken url: %v read err: %s, rsp: %v", url, err.Error(), resp.Body)
		return nil, err
	}

	rspResult := &getAccessResponse{}
	if err := json.Unmarshal(body, &rspResult); err != nil {
		log.Error("RefreshAccessToken json parse error, err = %s, json = %s", err.Error(), string(body))
		return nil, err
	}

	if rspResult.Errcode != 0 {
		log.Error("RefreshAccessToken error code: %v, rsp: %v", rspResult.Errcode, rspResult)
		if rspResult.Errcode == 42007 || rspResult.Errcode == 41003 || rspResult.Errcode == 40030 || rspResult.Errcode == 42002 {
			return nil, fmt.Errorf("RefreshAccessToken refresh error code: %v", WX_NO_PERM)
		}
		return nil, fmt.Errorf("RefreshAccessToken refresh error code: %v", WX_FAILED)
	}

	if len(rspResult.AccessToken) == 0 {
		log.Error("RefreshAccessToken AccessToken is empty, rsp: %v", rspResult)
		return nil, errors.New("RefreshAccessToken AccessToken is empty")
	}
	result.AccessToken = rspResult.AccessToken
	result.RefreshToken = rspResult.RefreshToken
	result.ExpiresIn = rspResult.ExpiresIn
	result.OpenID = rspResult.OpenID
	result.Scope = rspResult.Scope
	log.Info("RefreshAccessToken success, refresh %v by refresh_token: %v", result, refreshToken)
	err = saveToken(result)
	if err != nil {
		log.Error("RefreshAccessToken save oid: %v token info error: %v", result.OpenID, err)
	}

	return
}

// FetchRecentStepsByToken get user step by token
func FetchStepsByToken(oid string, limit int8) (result map[string]int64, err error) {
	// avoid panic because call third party
	defer func() {
		if r := recover(); r != nil {
			log.Error("FetchStepsByToken recovered in %v", r)
			switch x := r.(type) {
			case string:
				err = errors.New(x)
			case error:
				err = x
			default:
				err = errors.New("unknown panic")
			}
		}
	}()
	info, err := getToken(oid)
	if err != nil {
		log.Error(fmt.Sprintf("FetchStepsByToken error: %v", err.Error()))
		if err == redis.Nil {
			return nil, fmt.Errorf("FetchStepsByToken nil token info, code: %v, error: %v", WX_NO_PERM, err)
		}
		return nil, fmt.Errorf("FetchStepsByToken get token info code: %v, error: %v", WX_FAILED, err)
	}
	ser, err := getWxL5(client.config.ModIDH5, client.config.CmdIDH5)
	if err != nil {
		return nil, err
	}
	// get wx step
	url := formatUrl(ser.Ip(), ser.Port(), info.AccessToken, getTodayBeginTime(), limit)
	// response_body：
	// {
	//   "errcode":0,
	//   "list":[
	//     {
	//       "step":6011,
	//       "time":1438704000
	//     },
	//     {
	//       "step":2801,
	//       "time":1438790400
	//     }
	//   ],
	//   "msg":""
	// }
	resp, err := client.client.Get(url)
	if err != nil {
		log.Error("FetchStepsByToken url: %v rpc err: %s", url, err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("FetchStepsByToken url: %v read err: %s, rsp: %v", url, err.Error(), resp.Body)
		return nil, err
	}
	var wxSteps wxStepsH5Data
	if err := json.Unmarshal(body, &wxSteps); err != nil {
		log.Error("FetchStepsByToken user json parse error, err = %s, json = %s", err.Error(), string(body))
		return nil, err
	}
	log.Debug("FetchStepsByToken get rsp: %v", string(body))
	// check if no auth
	if wxSteps.Errcode != 0 {
		log.Error("FetchStepsByToken error code: %v, json: %v", wxSteps.Errcode, string(body))
		if wxSteps.Errcode == 40001 || wxSteps.Errcode == 42001 || wxSteps.Errcode == 48001 {
			return nil, fmt.Errorf("FetchStepsByToken error: %v, code: %v, json: %v", WX_NO_PERM.String(), wxSteps.Errcode, string(body))
		}
		return nil, fmt.Errorf("FetchStepsByToken error: %v, code: %v, json: %v", WX_FAILED, wxSteps.Errcode, string(body))
	}

	dateSteps := make(map[string]int64)
	for _, value := range wxSteps.StepList {
		tm := time.Unix(value.Timestamp, 0)
		date := tm.Format("2006-01-02")
		dateSteps[date] = value.Step
	}
	log.Info("FetchStepsByToken get oid: %v, steps: %v", oid, dateSteps)
	return dateSteps, nil
}

func getTodayBeginTime() time.Time {
	currentTime := time.Now().In(Loc)
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	return startTime
}

func formatUrl(ip string, port int, accessToken string, timestamp time.Time, limit int8) string {
	url := fmt.Sprintf("http://%s:%d/hardware/snstransfer/bracelet/snsgetstep?access_token=%s&timestamp=%v", ip, port, accessToken, timestamp.Unix())
	var i int8
	if limit > 0 {
		for i = 1; i < limit; i++ {
			url += fmt.Sprintf("|%v", timestamp.Unix()+(int64(i)*86400))
		}
	} else if limit < 0 {
		//	for i = 1; i < limit; i++ {
		for i = 1; i < -limit; i++ {
			url += fmt.Sprintf("|%v", timestamp.Unix()-(int64(i)*86400))
		}
	}
	log.Info("format url: %v", url)
	return url
}

func saveToken(result *AccessTokenInfo) error {
	// use redis
	key := generateTokenKey(result.OpenID)
	value, err := json.Marshal(result)
	if err != nil {
		log.Error("saveToken json marshal error: %v", err)
		return err
	}
	res, err := cache_access.RedisClient.Set(key, value, TOKEN_EXPIRE_TIME*time.Second).Result()
	if err != nil {
		log.Error("saveToken error: %v", err)
		return err
	}
	/*
		// use mem
		userToken.Store(result.OpenID, result)
	*/
	log.Debug("saveToken success: %v", res)
	return nil
}

func getToken(oid string) (*AccessTokenInfo, error) {
	accessToken := &AccessTokenInfo{}
	// use redis
	key := generateTokenKey(oid)
	res, err := cache_access.RedisClient.Get(key).Result()
	if err != nil {
		return nil, err
	}
	// need auth
	if len(res) == 0 {
		return nil, redis.Nil
	}
	// unmarshal result
	err = json.Unmarshal([]byte(res), accessToken)
	if err != nil {
		// need auth to refresh redis
		return nil, redis.Nil
	}
	/*
		// use memory
		info, ok := userToken.Load(oid)
		if !ok {
			return nil, fmt.Errorf("getToken oid: %v, code: %v", oid, WX_NO_PERM)
		}
		accessToken = info.(*AccessTokenInfo)
	*/
	log.Debug("getToken success: %v", accessToken)
	return accessToken, nil
}

func generateTokenKey(oid string) string {
	return fmt.Sprintf("%s:%s", REDIS_TOKEN_KEY, oid)
}
