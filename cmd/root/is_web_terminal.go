package root

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dbr"
)

// TODO: Split this into a separate PR and add a test.
func isWebTerminal(ctx context.Context) bool {
	if !dbr.RunsOnRuntime(ctx) {
		return false
	}

	cur := os.Getpid()

	// Max number of ancestors to check for trying to detect if the process is
	// running in a web terminal (i.e. launched by ttyd).
	maxHeight := 10

	for range maxHeight {
		// If the pid is a 0 or 1, we are at the root of the process tree.
		if cur == 0 || cur == 1 {
			return false
		}

		// Read the name of the current process
		b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", cur))
		if err != nil {
			return false
		}

		// If the name for any of the parent processes is ttyd, then the
		// CLI has been run from the web terminal.
		if strings.TrimSpace(string(b)) == "ttyd" {
			return true
		}

		// The 4th field in /proc/<pid>/stat is the parent pid.
		b, err = os.ReadFile(fmt.Sprintf("/proc/%d/stat", cur))
		if err != nil {
			return false
		}

		stat := strings.Split(string(b), " ")
		if len(stat) < 4 {
			return false
		}

		v, err := strconv.Atoi(stat[3])
		if err != nil {
			return false
		}

		cur = v
	}

	return false
}
