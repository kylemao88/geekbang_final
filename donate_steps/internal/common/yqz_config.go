//
//

package common

import (
	"fmt"

	"git.code.oa.com/gongyi/agw/config"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/pkg/dbclient"
)

type WxStepsL5Config struct {
	ModId int32 `toml:"mod_id"`
	CmdId int32 `toml:"cmd_id"`
}

type WxStepsH5Config struct {
	ModId     int32  `toml:"mod_id"`
	CmdId     int32  `toml:"cmd_id"`
	Appid     string `toml:"app_id"`
	AppSecret string `toml:"app_secret"`
}

type DirtyConfig struct {
	DirtyModid int32  `toml:"dirty_modid"`
	DirtyCmdid int32  `toml:"dirty_cmdid"`
	DirtyIp    string `toml:"dirty_ip"`
	DirtyPort  int    `toml:"dirty_port"`
}

type ProfitShareConfig struct {
	ModId     int32  `toml:"mod_id"`
	CmdId     int32  `toml:"cmd_id"`
	NotifyUrl string `toml:"notify_url"`
}

type SendRedpacketConfig struct {
	ModId      int32 `toml:"mod_id"`
	CmdId      int32 `toml:"cmd_id"`
	ExpireDays int32 `toml:"expire_days"` // 红包过期天数, 超过x天没领自动捐给机构
}

type MsgTempleConfig struct {
	MsgIds []string `toml:"msg_ids"`
}

type CkvKeyConfig struct {
	RouteKey          string `toml:"route_key"`
	RouteKeyV2        string `toml:"route_key_v2"`
	WhiteKey          string `toml:"white_key_v2"`
	BigMapKey         string `toml:"big_map_key"`
	GlobalWeekKey     string `toml:"global_week_key"`
	GlobalWeekHistory string `toml:"global_week_history"`
}

type GlobalWeekInterval struct {
	Interval int `toml:"interval"` // km
}

type EventPushConfig struct {
	State string   `toml:"state"` // developer为开发版；trial为体验版；formal为正式版；默认为正式版
	Black []string `toml:"black"` // oid黑名单
}

type ServiceConfig struct {
	Addr            string `toml:"addr"`
	Port            int    `toml:"port"`
	PolarisEnv      string `toml:"polaris_env"`
	FullPolarisAddr string `toml:"full_polaris_addr"`
	PolarisAddr     string `toml:"polaris_addr"`
}

type EdProxyConfig struct {
	Flag     bool   `toml:"flag"`
	Addr     string `toml:"addr"`
	PollInit int    `toml:"poll_init"`
	PollIdle int    `toml:"poll_idle"`
	PollPeak int    `toml:"poll_peak"`
}

type AccessConfig struct {
	TeamMemberLimit  int    `default:"100" toml:"team_member_limit"`
	SuppFundRankDate string `default:"2022-01-06 00:00:00" toml:"supp_fund_rank_date"`
}

type YqzConfig struct {
	YqzDbConfig           data_access.DBConfig     `toml:"yqz_db_config"`
	RechargeDBConfig      data_access.DBConfig     `toml:"recharge_db_config"`
	RedisConfig           cache_access.RedisConfig `toml:"redis_conf"`
	Wxstepsl5config       WxStepsL5Config          `toml:"wxstepsl_5_config"`
	Dirtyconfig           DirtyConfig              `toml:"dirty_config"`
	ProfitShareconfig     ProfitShareConfig        `toml:"profit_share_config"`
	SendRedpacketconfig   SendRedpacketConfig      `toml:"send_redpacket_config"`
	MsgTemple             MsgTempleConfig          `toml:"msg_temple_config"`
	CkvKey                CkvKeyConfig             `toml:"ckv_key_config"`
	GlobalWeekInterval    GlobalWeekInterval       `toml:"global_week_interval"`
	EventPushConfig       EventPushConfig          `toml:"event_push_config"`
	YqzStatisticsDbConfig dbclient.DBConfig        `toml:"yqz_statistics_db_config"`
	FundMgr               ServiceConfig            `toml:"fundmgr"`
	StepMgr               ServiceConfig            `toml:"stepmgr"`
	ActivityMgr           ServiceConfig            `toml:"activitymgr"`
	EdProxy               EdProxyConfig            `toml:"edproxy"`
	WxStepH5              WxStepsH5Config          `toml:"wxstep_h5_config"`
	AccessConf            AccessConfig             `toml:"access"`
}

var Yqzconfig YqzConfig

func init() {
	if err := config.Parse(&Yqzconfig); err != nil {
		panic("yqz parse conf error!")
	}
	fmt.Printf("yqz read conf succ, config = %v", Yqzconfig)
}
