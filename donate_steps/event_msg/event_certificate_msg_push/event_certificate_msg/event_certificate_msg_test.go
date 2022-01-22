package event_certificate_msg

import (
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"github.com/agiledragon/gomonkey"
	"testing"
	"time"
)

func init() {
	log.InitGlobal("../log/event_weekly_msg", 10000000, 10)
	data_access.InitDBConn(common.Yqzconfig.YqzDbConfig)

	config := cache_access.RedisConfig{
		IpPorts: "127.0.0.1:6379",
		ModId:   0,
		CmdId:   0,
	}
	cache_access.RedisInit(config)
	gomonkey.ApplyFunc(common.GetGlobalWeekInfo, func() (common.GlobalWeekInfo, error) {
		return common.GlobalWeekInfo{
			Name: "大一块走",
		}, nil
	})
	gomonkey.ApplyFunc(data_access.GetUninid, func(oid string) (string, error) {
		return oid, nil
	})
	gomonkey.ApplyFunc(common.GetWeekMap, func() (map[string]common.WeekMap, error) {
		var ret = make(map[string]common.WeekMap)
		t, _ := common.WeekStartEnd(time.Now())
		ret[t.Format("20060102")] = common.WeekMap{
			Name:      "西湖之行",
			MapId:     1617,
			Week:      t.Format("20060102"),
			CertTitle: "http://xxx.png",
		}
		return ret, nil
	})
	gomonkey.ApplyFunc(common.GetGlobalWeekLevel, func() (map[int]common.GlobalWeekLevel, error) {
		var ret = make(map[int]common.GlobalWeekLevel)
		ret[1616] = common.GlobalWeekLevel{
			Name:     "南京玄武湖",
			Distance: 21000,
			Abbr:     "玄武湖",
		}
		ret[1617] = common.GlobalWeekLevel{
			Name:     "西湖之行",
			Distance: 20000,
			Abbr:     "西湖",
		}
		return ret, nil
	})
	gomonkey.ApplyFunc(common.MGetNickHead, func(openids []string) (map[string]common.MNickHead, error) {
		ret := make(map[string]common.MNickHead)
		for _, v := range openids {
			ret[v] = common.MNickHead{
				Nick: v,
				Head: v,
			}
		}
		return ret, nil
	})
	gomonkey.ApplyFunc(common.SendWxMsg, func(appid string, msg string) error {
		log.Debug("msg = %v", msg)
		return nil
	})
}

func TestEventCertificateMsgPush(t *testing.T) {
	t.Log(EventCertificateMsgPush(common.WeekChallenge{
		Oid:  "oproJj0AsmIBLEn5uSD8YL7fE7lU",
		Step: 35686,
		Rank: 1,
		Num:  3,
		Time: "2021-04-12 14:41:00",
	}))
}
