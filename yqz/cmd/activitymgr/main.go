//
// Package main provides service entry functions
// 启动示例 1：
// ./activitymgr -busi_conf=../conf/ -namespace=Development -service=yqz.activitymgr -grpc_port=9001 -sidecar_port=19001 -http_port=8001
// 启动示例 2：
// ./activitymgr -busi_conf=../conf/ -namespace=Development -service=yqz.activitymgr -grpc_port=9001 -sidecar_port=19001 -http_port=8001 -enable_printscreen -log_level=5
//
// 对应的 sidecar 启动示例 1：
// ./sidecar -namespace=Development -service=yqz.activitymgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000
// 对应的 sidecar 启动示例 2：
// ./sidecar -namespace=Development -service=yqz.activitymgr -http_port=18001 -grpc_port=19001 -svc_port=8001 -auto_register=false -skywalking=endpoint://127.0.0.1:11800 -sampling_rate=5000 -enable_printscreen -log_level=5

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"runtime/pprof"

	gms "git.code.oa.com/gongyi/gomore/service"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/comment"
	"git.code.oa.com/gongyi/yqz/pkg/activitymgr/handler"
)

var busiConf string
var cpuprofile string
var polltask bool

//命令行参数
func init() {
	// initialize global variable
	flag.StringVar(&busiConf, "busi_conf", "../conf/busi_conf.yaml", "Business configure dir path.")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	flag.BoolVar(&polltask, "polltask", false, "poll task for FreshRank or snapshot...")
}

func testHandle(svc *gms.GRPCService) {
	svc.Handle("LeaveActivity", handler.LeaveActivity)
}

func main() {
	flag.Parse()
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create cpu profile: ", err)
		}

		pprof.StartCPUProfile(f)
	}

	go func() {
		http.ListenAndServe("localhost:9105", nil)
	}()

	debug.SetGCPercent(1000)

	svc := gms.NewService()
	svc.Handle("CreateActivity", handler.CreateActivity)
	svc.Handle("DeleteActivity", handler.DeleteActivity)
	svc.Handle("DeleteActivityTeam", handler.DeleteActivityTeam)

	svc.Handle("QueryUserCreateActivity", handler.QueryUserCreateActivity)
	svc.Handle("QueryUserJoinActivityNew", handler.QueryUserJoinActivityNew)
	svc.Handle("QueryActivityUserStats", handler.QueryActivityUserStats)
	svc.Handle("QueryActivity", handler.QueryActivity)
	svc.Handle("ActivityTeamRank", handler.ActivityTeamRank)
	svc.Handle("ActivityUserRank", handler.ActivityUserRank)

	svc.Handle("JoinActivity", handler.JoinActivity)
	svc.Handle("JoinTeam", handler.JoinTeam)
	svc.Handle("QueryTeams", handler.QueryTeams)
	svc.Handle("QueryTeamStats", handler.QueryTeamStats)
	svc.Handle("QueryActivitySuccessNum", handler.QueryActivitySuccessNum)

	// test
	testHandle(svc)
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
	return activitymgr.InitComponents(busiConf, polltask)
}

// 服务退出收尾
func svrFini() {
	if cpuprofile != "" {
		defer pprof.StopCPUProfile()
	}

	comment.CloseStepComponent()
}
