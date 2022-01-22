//
package fundmgr

import (
	"fmt"

	"git.code.oa.com/gongyi/gy_warn_msg_httpd/msgclient"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/client"
	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/ckvclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/mqclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/activity"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/match"
	"git.code.oa.com/gongyi/yqz/pkg/statisticmgr"
)

var (
	cdbClient dbclient.DBClient
)

// InitComponents init all fundmgr client
func InitComponents(path string) bool {
	// get config from file
	config.InitConfig(path)
	conf := config.GetConfig()

	// init logger
	if conf.PluginConf != nil {
		err := conf.PluginConf.Setup()
		if err != nil {
			return false
		}
	}

	// init alarm client
	msgclient.Msg_init(&msgclient.STMsgPara{
		Businame: conf.AlarmConf.Busi,
		Sender:   conf.AlarmConf.Sender,
		Recver:   conf.AlarmConf.Receiver,
		Septime:  conf.AlarmConf.Septime,
	})

	// init mysql
	var err error
	cdbClient, err = dbclient.GetDBClientByCfg(conf.DBConf)
	if err != nil {
		logger.Error("GetDBClientByCfg error: %v", err)
		return false
	}

	// init ckv
	if err = ckvclient.CkvInit(conf.CkvConf); err != nil {
		logger.Error("Get NewCkvClient_pool error: %v", err)
		return false
	}

	// init redis
	cacheclient.RedisInit(conf.RedisConf)

	// init statistic mgr
	if err = statisticmgr.InitStatisticDB(conf.StatisticDBConf); err != nil {
		logger.Error("InitStatisticDB error: %v", err)
		return false
	}

	// init activity component
	if err = activity.InitComponent(cdbClient); err != nil {
		logger.Error("Init activity component error: %v", err)
		return false
	}

	// init match component
	if err = match.InitComponent(cdbClient, conf.MatchConf); err != nil {
		logger.Error("Init match component error: %v", err)
		return false
	}

	// init monitor component
	err = InitMonitorComponent()
	if err != nil {
		logger.Error("InitMonitorComponent error: %v", err)
		return false
	}

	// init activity mgr client
	if err = client.InitClient(&conf.ProxyConf, &conf.ActivityConf); err != nil {
		logger.Error("Init match component error: %v", err)
		return false
	}

	if err = mqclient.KafkaSenderInit(); err != nil {
		logger.Error("KafkaSenderInit error: %v", err)
		return false
	}

	alarm.CallAlarmFunc(fmt.Sprintf("fundmgr: %v init success", util.GetLocalIP()))

	return true
}

// CloseComponents close all component
func CloseComponents() {
	CloseMonitorComponent()
	match.CloseComponent()
}
