//

// ./stepmgr -busi_conf=../conf/ -namespace=Development -service=yqz.stepmgr -grpc_port=9006 -sidecar_port=19006 -http_port=8006
// ./stepmgr -busi_conf=../conf/ -namespace=Development -service=yqz.stepmgr -grpc_port=9006 -sidecar_port=19006 -http_port=8006 -enable_printscreen -log_level=5
//
// ./sidecar -namespace=Development -service=yqz.stepmgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000
// ./sidecar -namespace=Development -service=yqz.stepmgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000 -enable_printscreen -log_level=5

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"

	gms "git.code.oa.com/gongyi/gomore/service"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr"
	"git.code.oa.com/gongyi/yqz/pkg/stepmgr/handler"
)

var busiConf string
var cpuprofile string

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
		http.ListenAndServe("localhost:9106", nil)
	}()

	svc := gms.NewService()
	svc.Handle("GetUsersSteps", handler.HandleGetUsersSteps)
	svc.Handle("SetUsersSteps", handler.HandleSetUsersSteps)
	gms.SimpleSvrMain(svc, checkParams, svrInit, svrFini)
	defer stepmgr.CloseComponents()
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
	return stepmgr.InitComponents(busiConf)
}

// 服务退出收尾
func svrFini() {
	if cpuprofile != "" {
		defer pprof.StopCPUProfile()
	}
	// useless ????
	//stepmgr.CloseComponents()
}
