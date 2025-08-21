package main

import (
	"fmt"
	"os"
)

func main() {
	// Directories with the _ prefix are ignored by Go. That is important
	// since the go installation in _bin would include go modules which would
	// otherwise cause an error during a build of the CLI.
	err := createArchive("_build", "_bin", "../..")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archive: %v\n", err)
		os.Exit(1)
	}
}
