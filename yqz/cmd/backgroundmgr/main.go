package main

import (
	"flag"
	"net/http"

	"git.code.oa.com/gongyi/yqz/pkg/backgroundmgr/msgmgr"
	"git.code.oa.com/gongyi/yqz/pkg/common/logger"
)

var (
	busiConf  = flag.String("busi_conf", "../conf/", "-busi_conf=../conf/")
	kafkaConf = flag.String("kafka_conf", "../conf/kafka_conf.toml", "-kafka_conf=../conf/kafka_conf.toml")
)

func main() {
	flag.Parse()

	go func() {
		http.ListenAndServe("localhost:9107", nil)
	}()

	m, err := msgmgr.NewRemind(*busiConf)
	if err != nil {
		logger.Error("new match remind error: %v", err)
		return
	}

	if err = m.Remind(*kafkaConf); err != nil {
		logger.Error("consumer remind err = %v", err)
		return
	}

}
