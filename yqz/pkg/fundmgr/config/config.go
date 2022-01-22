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
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"gopkg.in/yaml.v2"
)

// ConfigName is the config file's name
const ConfigName string = "busi_conf.yaml"

type BusinessConfig struct {
	CkvConf         ckvclient.CkvConfig       `yaml:"ckv_conf"`
	RedisConf       cacheclient.RedisConfig   `yaml:"redis_conf"`
	DBConf          dbclient.DBConfig         `yaml:"cdb_conf"`
	StatisticDBConf dbclient.DBConfig         `yaml:"statistic_cdb_conf"`
	AlarmConf       alarm.AlarmMsgConfig      `yaml:"alarm_msg_conf"`
	LogConf         LogConfig                 `yaml:"log_conf"`
	MatchConf       MatchConfig               `yaml:"match_conf"`
	ActivityConf    proxyclient.ServiceConfig `yaml:"activitymgr_conf"`
	ProxyConf       proxyclient.ProxyConfig   `yaml:"proxy_conf"`
	PluginConf      plugin.Config             `yaml:"plugins"`
	CouponsConf     CouponsConfig             `yaml:"coupons_conf"`
}

// 运营平台配捐活动列表key
type MatchConfig struct {
	ListKey            string `yaml:"list_key"`              // 配捐活动列表key
	RemainMatchRuleKey string `yaml:"remain_match_rule_key"` // 剩余配捐规则key
	RecoverCount       int    `yaml:"recover_count"`
	RecoverIntervalMs  int    `yaml:"recover_interval_ms"`
	UpdateRankInterval int    `yaml:"update_rank_interval"`
	CouponsCount       int    `yaml:"coupons_count"`
	CouponsIntervalMs  int    `yaml:"coupons_interval_ms"`
}

// RPC请求客户端配置
type ClientConfig struct {
	ServiceAddr string `yaml:"service_addr"`
	ServicePort int    `yaml:"service_port"`
}

type LogConfig struct {
	FileName string `yaml:"file_name"`
	FileSize int    `yaml:"file_size"`
	FileNum  int    `yaml:"file_num"`
	LogLevel int8   `yaml:"log_level"`
	LogDays  int    `yaml:"log_days"`
	Compress bool   `yaml:"compress"`
}

type CouponsConfig struct {
	Url string `yaml:"url"`
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
	if cfg.MatchConf.RecoverCount == 0 {
		cfg.MatchConf.RecoverCount = 2
	}
	if cfg.MatchConf.RecoverIntervalMs == 0 {
		cfg.MatchConf.RecoverIntervalMs = 100
	}
	if cfg.MatchConf.UpdateRankInterval == 0 {
		cfg.MatchConf.UpdateRankInterval = 3600
	}
	if cfg.MatchConf.CouponsCount == 0 {
		cfg.MatchConf.CouponsCount = 2
	}
	if cfg.MatchConf.CouponsIntervalMs == 0 {
		cfg.MatchConf.CouponsIntervalMs = 100
	}
	if len(cfg.CouponsConf.Url) == 0 {
		cfg.CouponsConf.Url = "https://coupon-api.gongyi.tencent-cloud.net"
	}
	return true
}

// prettyConfig : show monitor's config
func prettyConfig() {
	d, _ := yaml.Marshal(globalConfig)
	fmt.Println("=============  fundmgr config   =============")
	fmt.Println(string(d))
	fmt.Println("==============================================")
}
