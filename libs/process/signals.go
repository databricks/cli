package process

import (
	"os"
	"os/signal"
	"syscall"
)

// hupIgnored is whether this process started with SIGHUP ignored, as under
// nohup. It is read when the package is initialized, before anything catches
// SIGHUP: catching a signal clears the record that it was ignored.
var hupIgnored = signal.Ignored(syscall.SIGHUP)

// InterruptSignals returns the signals that ask the CLI to stop: Ctrl-C, a
// request to terminate and the terminal closing. SIGHUP is left out when the
// process started with it ignored, so that catching it does not undo nohup.
func InterruptSignals() []os.Signal {
	if hupIgnored {
		return []os.Signal{os.Interrupt, syscall.SIGTERM}
	}
	return []os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}
}
