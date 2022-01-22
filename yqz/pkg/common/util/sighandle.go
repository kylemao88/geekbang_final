//

package util

import (
	"os"
	"os/signal"
	"syscall"
)

// DefaultServiceCloseSIG defines default close signal.
var DefaultServiceCloseSIG = []os.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT}

// HandleSignal handles signal.
func HandleSignal(callback func(sig os.Signal)) {
	go func() {
		signalChannel := make(chan os.Signal, 10)
		signal.Notify(signalChannel, DefaultServiceCloseSIG...)

		s := <-signalChannel
		callback(s)

		os.Exit(-1)
	}()
}
