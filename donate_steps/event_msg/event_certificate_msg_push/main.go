package main

import (
	systemlog "log"

	"git.code.oa.com/gongyi/agw/log"
	"git.code.oa.com/gongyi/donate_steps/internal/cache_access"
	"git.code.oa.com/gongyi/donate_steps/internal/common"
	"git.code.oa.com/gongyi/donate_steps/internal/data_access"
	"git.code.oa.com/gongyi/donate_steps/event_msg/event_certificate_msg_push/event_certificate_msg"
	"git.code.oa.com/gongyi/gongyi_base/component/gy_kafka"
)

func init() {
	log.SetFlags(systemlog.LstdFlags | systemlog.Llongfile)
}

func main() {
	log.InitGlobal("../log/event_certificate_msg", 500000000, 10)
	data_access.InitDBConn(common.Yqzconfig.YqzDbConfig)
	defer func() {
		data_access.DBHandler.Close()
	}()

	cache_access.RedisInit(common.Yqzconfig.RedisConfig)

	log.Debug("config = %+v", common.Yqzconfig)

	consumer := gy_kafka.NewKafkaConsumer()
	client := &event_certificate_msg.Client{}
	client.SyncCtx = consumer.Bind(client)
	consumer.Run("../conf/config.toml")
}
