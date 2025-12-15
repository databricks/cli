package common

import "os"

// GetCLIPath returns the path to the current CLI executable.
// This supports development testing with ./cli.
func GetCLIPath() string {
	return os.Args[0]
}
