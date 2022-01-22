//

package util

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var (
	version bool
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
	flag.BoolVar(&version, "version", false, "get version for http_svr service")
}

// ShowVersion is the function called by others to print version information
func ShowVersion() {
	fmt.Printf("build name:\t%s\n", BuildName)
	fmt.Printf("build ver:\t%s\n", BuildVersion)
	fmt.Printf("build time:\t%s\n", BuildTime)
	fmt.Printf("Commit ID:\t%s\n", CommitID)
	fmt.Printf("Branch Name:\t%s\n", BranchName)
}

// HandleFlags handles flags.
func HandleFlags() {
	if version {
		ShowVersion()
		os.Exit(0)
	}
}
