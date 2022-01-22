//
package stepmgr

import (
	"fmt"

	"git.code.oa.com/gongyi/gy_warn_msg_httpd/msgclient"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/client"
	"git.code.oa.com/gongyi/yqz/pkg/common/alarm"
	"git.code.oa.com/gongyi/yqz/pkg/common/cacheclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/dbclient"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
	"git.code.oa.com/gongyi/yqz/pkg/common/util"
	"git.code.oa.com/gongyi/yqz/pkg/common/wxclient"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/auto"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/config"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/pk"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/steps"
)

var (
	cdbClient dbclient.DBClient
)

// InitComponents init all stepmgr component
func InitComponents(path string) bool {
	// get config from file
	config.InitConfig(path)
	conf := config.GetConfig()

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

	// init redis
	cacheclient.RedisInit(conf.RedisConf)

	// init wx client
	wxclient.InitWXClient(wxclient.WithL5(conf.WXStepConf.ModId, conf.WXStepConf.CmdId))

	// init monitor component
	err = InitMonitorComponent()
	if err != nil {
		logger.Error("InitMonitorComponent error: %v", err)
		return false
	}

	// init step component
	steps.InitStepComponent(cdbClient, cacheclient.RedisClient, conf.StepConf)
	// init pk component
	pk.InitPkComponent(cdbClient, cacheclient.RedisClient, conf.StepConf)

	// init auto component
	auto.InitAutoUpdateComponent(conf.StepConf, cdbClient, cacheclient.RedisClient)

	// init activity mgr client
	if err = client.InitClient(&conf.ProxyConf, &conf.ActivityConf); err != nil {
		logger.Error("init activity client error: %v", err)
		return false
	}

	// init alarm
	alarm.CallAlarmFunc(fmt.Sprintf("stepmgr(%v) init success", util.GetLocalIP()))

	return true
}

// CloseComponents close all component
func CloseComponents() {
	steps.CloseStepComponent()
	auto.CloseAutoUpdateComponent()
	CloseMonitorComponent()
}
