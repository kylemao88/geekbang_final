//
package config

import (
	"fmt"

	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/config"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/proxyclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/wxclient"
	"git.code.oa.com/trpc-go/trpc-go/plugin"
	"gopkg.in/yaml.v2"
)

// ConfigName is the config file's name
const ConfigName string = "busi_conf.yaml"

type BusinessConfig struct {
	RedisConf    cacheclient.RedisConfig   `yaml:"redis_conf"`
	DBConf       dbclient.DBConfig         `yaml:"cdb_conf"`
	AlarmConf    alarm.AlarmMsgConfig      `yaml:"alarm_msg_conf"`
	ProxyConf    proxyclient.ProxyConfig   `yaml:"proxy_conf"`
	StepConf     StepConfig                `yaml:"step_conf"`
	PluginConf   plugin.Config             `yaml:"plugins"`
	WXStepConf   wxclient.WxL5Config       `yaml:"wx_step_conf"`
	ActivityConf proxyclient.ServiceConfig `yaml:"activitymgr_conf"`
}

type StepConfig struct {
	RecoverCount      int  `yaml:"recover_count"`
	RecoverIntervalMs int  `yaml:"recover_interval_ms"`
	Flag              bool `yaml:"flag"`
	UpdateCount       int  `yaml:"update_count"`
	UpdateIntervalS   int  `yaml:"update_interval_s"`
	WxUpdateBatch     int  `yaml:"wx_update_batch"`
	DBUpdateBatch     int  `yaml:"db_update_batch"`
	DBFetchBatch      int  `yaml:"db_fetch_batch"`
	AutoLimit         int  `yaml:"auto_limit"`
}

var globalConfig = &BusinessConfig{}

// InitConfig init service config
func InitConfig(path string) {
	if err := config.InitConfigFromFile(path, ConfigName, globalConfig); err != nil {
		panic(err.Error())
	}

	if !validateConfig(globalConfig) {
		panic("fundmgr get invalid config ... exit")
	}
	prettyConfig()
}

// GetConfig gets service global config.
func GetConfig() *BusinessConfig {
	return globalConfig
}

// validateConfig checks the validation of config.
func validateConfig(cfg *BusinessConfig) bool {
	if cfg == nil {
		return false
	}
	// recover config
	if cfg.StepConf.RecoverCount == 0 {
		cfg.StepConf.RecoverCount = 2
	}
	if cfg.StepConf.RecoverIntervalMs == 0 {
		cfg.StepConf.RecoverIntervalMs = 50
	}
	// auto update config
	if cfg.StepConf.UpdateCount == 0 {
		cfg.StepConf.UpdateCount = 4
	}
	if cfg.StepConf.UpdateIntervalS == 0 {
		cfg.StepConf.UpdateIntervalS = 600
	}
	if cfg.StepConf.WxUpdateBatch == 0 {
		cfg.StepConf.WxUpdateBatch = 50
	}
	if cfg.StepConf.DBUpdateBatch == 0 {
		cfg.StepConf.DBUpdateBatch = 1000
	}
	if cfg.StepConf.DBFetchBatch == 0 {
		cfg.StepConf.DBFetchBatch = 5000
	}
	// control max burst update speed for db, speed = autoLimit * dbUpdateBatch
	if cfg.StepConf.AutoLimit == 0 {
		cfg.StepConf.AutoLimit = 10
	}

	return true
}

// prettyConfig show service config
func prettyConfig() {
	d, _ := yaml.Marshal(globalConfig)
	fmt.Println("=============  service config   =============")
	fmt.Println(string(d))
	fmt.Println("==============================================")
}
