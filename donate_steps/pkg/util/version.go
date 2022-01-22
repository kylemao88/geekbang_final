//

package util

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	// configPath string
	version  bool
	TestMode string
	// BuildVersion contains the current version.
	BuildVersion string
	// BuildTime contains the current build time.
	BuildTime string
	// BuildName contains the build name.
	BuildName string
	// CommitID contains the current git id.
	CommitID string
	// BranchName contains the git branch name.
	BranchName string
	// StartTime contains the start time
	StartTime int64
)

func init() {
	StartTime = time.Now().Unix()
	// check version flag
	flag.BoolVar(&version, "version", false, "get version for http_svr service")
	// flag.BoolVar(&testMode, "t", false, "test mode")
	// flag.StringVar(&configPath, "c", "../conf/config.toml", "configPath")
	// parse here is really bad, but agw init before main func
	// flag.Parse()
	// HandleFlags(&version)
	// agwConf.ConfPath = configPath
}

// HandleFlags handles flags.
func CheckTestMode() bool {
	fmt.Printf("test mode: %v\n", TestMode)
	return TestMode == "true"
}

// HandleFlags handles flags.
func HandleFlags() {
	if version {
		ShowVersion()
		os.Exit(0)
	}
}

// DefaultServiceCloseSIG defines default close signal.
var DefaultServiceCloseSIG = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

// HandleSignal handles signal.
func HandleSignal(callback func(sig os.Signal)) {
	go func() {
		signalChannel := make(chan os.Signal)
		signal.Notify(signalChannel, DefaultServiceCloseSIG...)

		s := <-signalChannel
		callback(s)

		os.Exit(-1)
	}()
}

// ShowVersion is the function called by others to print version information
func ShowVersion() {
	fmt.Printf("build name:\t%s\n", BuildName)
	fmt.Printf("build ver:\t%s\n", BuildVersion)
	fmt.Printf("build time:\t%s\n", BuildTime)
	fmt.Printf("Commit ID:\t%s\n", CommitID)
	fmt.Printf("Branch Name:\t%s\n", BranchName)
}
