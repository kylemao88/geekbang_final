//
// Package main provides service entry functions
// 启动示例 1：
// ./fundmgr -busi_conf=../conf/ -namespace=Development -service=yqz.fundmgr -grpc_port=9001 -sidecar_port=19001 -http_port=8001
// 启动示例 2：
// ./fundmgr -busi_conf=../conf/ -namespace=Development -service=yqz.fundmgr -grpc_port=9001 -sidecar_port=19001 -http_port=8001 -enable_printscreen -log_level=5
//
// 对应的 sidecar 启动示例 1：
// ./sidecar -namespace=Development -service=yqz.fundmgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000
// 对应的 sidecar 启动示例 2：
// ./sidecar -namespace=Development -service=yqz.fundmgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000 -enable_printscreen -log_level=5

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"

	gms "git.code.oa.com/gongyi/gomore/service"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr"
	"git.code.oa.com/gongyi/yqz/pkg/fundmgr/handler"
	"git.code.oa.com/polaris/polaris-go/api"
)

type FundMgrService struct {
}

var busiConf string
var cpuprofile string

//命令行参数
func init() {
	// initialize global variable
	flag.StringVar(&busiConf, "busi_conf", "../conf/", "Business configure dir path.")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
}

func main() {
	flag.Parse()
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	go func() {
		http.ListenAndServe("localhost:9101", nil)
	}()

	api.SetLoggersLevel(api.NoneLog)
	svc := gms.NewService()
	svc.Handle("GetUserMatchRecordByOffset", handler.GetUserMatchRecordByOffset)
	svc.Handle("GetUserTodayMatch", handler.GetUserTodayMatch)
	svc.Handle("UserMatch", handler.UserMatch)
	svc.Handle("UpdateMatch", handler.UpdateMatch)
	svc.Handle("GetActivityMatchInfo", handler.GetActivityMatchInfo)
	svc.Handle("GetActivityMatchRank", handler.GetActivityMatchRank)
	gms.SimpleSvrMain(svc, checkParams, svrInit, svrFini)
}

// 参数校验
func checkParams() bool {
	if len(busiConf) == 0 {
		fmt.Println("Parameter[-busiConf] is not set")
		return false
	}
	return true
}

// 服务启动初始化
func svrInit() bool {
	return fundmgr.InitComponents(busiConf)
}

// 服务退出收尾
func svrFini() {
	if cpuprofile != "" {
		defer pprof.StopCPUProfile()
	}

}
