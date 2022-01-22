//
package config

import (
	"fmt"

	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/ckvclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/config"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/proxyclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/wxclient"
	fund_config "git.code.oa.com/gongyi/yqz/pkg/fundmgr/config"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"gopkg.in/yaml.v2"
)

// ConfigName is the config file's name
const ConfigName string = "busi_conf.yaml"

type BusinessConfig struct {
	CkvConf         ckvclient.CkvConfig       `yaml:"ckv_conf"`
	RedisConf       cacheclient.RedisConfig   `yaml:"redis_conf"`
	DBConf          dbclient.DBConfig         `yaml:"cdb_conf"`
	StatsDBConf     dbclient.DBConfig         `yaml:"stats_db_conf"`
	AlarmConf       alarm.AlarmMsgConfig      `yaml:"alarm_msg_conf"`
	LogConf         LogConfig                 `yaml:"log_conf"`
	WxL5Conf        wxclient.WxL5Config       `yaml:"wxl5_conf"`
	PushConf        EventPushConfig           `yaml:"push_conf"`
	PluginConf      plugin.Config             `yaml:"plugins"`
	FundmgrConf     proxyclient.ServiceConfig `yaml:"fundmgr_conf"`
	StepmgrConf     proxyclient.ServiceConfig `yaml:"stepmgr_conf"`
	ActivitymgrConf proxyclient.ServiceConfig `yaml:"activitymgr_conf"`
	ProxyConf       proxyclient.ProxyConfig   `yaml:"proxy_conf"`
	MatchConf       fund_config.MatchConfig   `yaml:"match_conf"`
	RemindPercent   int                       `yaml:"remind_percent"` // 剩余配捐金额提醒比例
}

// 推送配置
type EventPushConfig struct {
	State            string   `yaml:"state"`              // developer为开发版；trial为体验版；formal为正式版；默认为正式版
	Black            []string `yaml:"black"`              // oid黑名单
	BattleRemindDays int      `yaml:"battle_remind_days"` // 战报消息推送间隔, 天数, 0表示当天立即推送, 用于测试
	BattlePushHour   int      `yaml:"battle_push_hour"`   // 每天9点推送战报
	CertPushHour     int      `yaml:"cert_push_hour"`     // 每天10点推送证书
}

type LogConfig struct {
	FileName string `yaml:"file_name"`
	FileSize int    `yaml:"file_size"`
	FileNum  int    `yaml:"file_num"`
	LogLevel int8   `yaml:"log_level"`
	LogDays  int    `yaml:"log_days"`
	Compress bool   `yaml:"compress"`
}

var globalConfig = &BusinessConfig{}

// InitConfig : init monitor config
func InitConfig(path string) {
	//if err := config.ParseConfigWithPath(globalConfig, path); err != nil {
	if err := config.InitConfigFromFile(path, ConfigName, globalConfig); err != nil {
		panic(err.Error())
	}

	if !validateConfig(globalConfig) {
		panic("fundmgr get invalid config ... exit")
	}
	prettyConfig()
}

// GetConfig gets monitor global config.
func GetConfig() *BusinessConfig {
	return globalConfig
}

// validateConfig checks the validation of config.
func validateConfig(cfg *BusinessConfig) bool {
	if cfg == nil {
		return false
	}

	return true
}

// prettyConfig : show monitor's config
func prettyConfig() {
	d, _ := yaml.Marshal(globalConfig)
	fmt.Println("=============  backgroundmgr config   =============")
	fmt.Println(string(d))
	fmt.Println("==============================================")
}
