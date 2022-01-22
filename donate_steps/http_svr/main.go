package main

import (
	"flag"
	"fmt"
	systemlog "log"
	"net/http"

	"git.code.oa.com/gongyi/agw"
	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	_ "git.code.oa.com/gongyi/donate_steps/http_svr/router"
	"git.code.oa.com/gongyi/donate_steps/pkg/activitymgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/dbclient"
	"git.code.oa.com/gongyi/donate_steps/pkg/fundmgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/polarisclient"
	"git.code.oa.com/gongyi/donate_steps/pkg/statistic"
	"git.code.oa.com/gongyi/donate_steps/pkg/stepmgr"
	"git.code.oa.com/gongyi/donate_steps/pkg/util"
	"git.code.oa.com/gongyi/donate_steps/pkg/wxclient"
	"git.code.oa.com/gongyi/gy_warn_msg_httpd/msgclient"
)

func init() {
	log.SetFlags(systemlog.LstdFlags | systemlog.Llongfile)
}

func main() {
	flag.Parse()
	util.HandleFlags()
	go func() {
		http.ListenAndServe("localhost:9110", nil)
	}()
	// log.InitGlobal("../log/http_svr", 300000000, 5)
	// log.SetLevel(log.LogLevelInfo)
	data_access.InitDBConn(common.Yqzconfig.YqzDbConfig)
	cfg := dbclient.DBConfig{
		DBType: "mysql",
		DBHost: common.Yqzconfig.YqzDbConfig.DbHost,
		DBPort: common.Yqzconfig.YqzDbConfig.DbPort,
		DBPass: common.Yqzconfig.YqzDbConfig.DbPass,
		DBUser: common.Yqzconfig.YqzDbConfig.DbUser,
		DBName: common.Yqzconfig.YqzDbConfig.DbName,
	}
	err := wxclient.InitWXClient(
		wxclient.WithL5(common.Yqzconfig.Wxstepsl5config.ModId, common.Yqzconfig.Wxstepsl5config.CmdId),
		wxclient.WithH5Step(common.Yqzconfig.WxStepH5.ModId, common.Yqzconfig.WxStepH5.CmdId, common.Yqzconfig.WxStepH5.Appid, common.Yqzconfig.WxStepH5.AppSecret))
	if err != nil {
		log.Error("init wx client err: %v", err)
		panic(fmt.Sprintf("init wx client err: %v", err))
	}
	// default client
	err = dbclient.InitDefaultMysqlClient("default", cfg)
	if err != nil {
		log.Error("init mysql default client err: %v", err)
		panic(fmt.Sprintf("init mysql default client err: %v", err))
	}
	// statistic client
	err = statistic.InitialStatistic(common.Yqzconfig.YqzStatisticsDbConfig)
	if err != nil {
		log.Error("init statistic client err: %v", err)
		panic(fmt.Sprintf("init statistic client err: %v", err))
	}
	// edproxy
	err = fundmgr.InitEdProxy(common.Yqzconfig.EdProxy)
	if err != nil {
		log.Error("init polaris client err:%v", err)
		panic(fmt.Sprintf("init polaris client err:%v", err))
	}
	// init fundmgr polaris client
	polarisConf := &polarisclient.PolarisInfo{PolarisEnv: common.Yqzconfig.FundMgr.PolarisEnv}
	if common.Yqzconfig.EdProxy.Flag {
		polarisConf.PolarisServiceName = common.Yqzconfig.FundMgr.PolarisAddr
	} else {
		polarisConf.PolarisServiceName = common.Yqzconfig.FundMgr.FullPolarisAddr
	}
	err = fundmgr.InitPolarisClient(polarisConf)
	if err != nil {
		log.Error("init fundmgr polaris client err:%v", err)
		panic(fmt.Sprintf("init fundmgr polaris client err:%v", err))
	}
	// init stepmgr polaris client
	stepPolarisConf := &polarisclient.PolarisInfo{PolarisEnv: common.Yqzconfig.StepMgr.PolarisEnv}
	if common.Yqzconfig.EdProxy.Flag {
		stepPolarisConf.PolarisServiceName = common.Yqzconfig.StepMgr.PolarisAddr
	} else {
		stepPolarisConf.PolarisServiceName = common.Yqzconfig.StepMgr.FullPolarisAddr
	}
	err = stepmgr.InitPolarisClient(stepPolarisConf)
	if err != nil {
		log.Error("init step polaris client err:%v", err)
		panic(fmt.Sprintf("init step polaris client err:%v", err))
	}
	// init activitymgr polaris client
	activityPolarisConf := &polarisclient.PolarisInfo{PolarisEnv: common.Yqzconfig.ActivityMgr.PolarisEnv}
	if common.Yqzconfig.EdProxy.Flag {
		activityPolarisConf.PolarisServiceName = common.Yqzconfig.ActivityMgr.PolarisAddr
	} else {
		activityPolarisConf.PolarisServiceName = common.Yqzconfig.ActivityMgr.FullPolarisAddr
	}
	err = activitymgr.InitPolarisClient(activityPolarisConf)
	if err != nil {
		log.Error("init activity polaris client err:%v", err)
		panic(fmt.Sprintf("init activity polaris client err:%v", err))
	}

	defer func() {
		wxclient.CloseWXClient()
		dbclient.CloseDefaultMysqlClient()
		statistic.Close()
		data_access.DBHandler.Close()
		fundmgr.ClosePolarisClient()
		activitymgr.ClosePolarisClient()
	}()

	cache_access.RedisInit(common.Yqzconfig.RedisConfig)
	common.KafkaSenderInit()

	msgclient.Msg_init(&msgclient.STMsgPara{
		Businame: "yqz_services",
		Sender:   "gongyi_warn",
		Recver:   "winterfeng,v_vypliu",
		Septime:  10 * 60,
	})

	agw.Run()
}
