package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <build-dir> <bin-dir>\n", os.Args[0])
		os.Exit(1)
	}

	buildDir := os.Args[1]
	binDir := os.Args[2]

	// Directories with the _ prefix are ignored by Go. That is important
	// since the go installation in _bin would include stdlib go modules which would
	// otherwise cause an error during a build of the CLI.
	err := createArchive(buildDir, binDir, "../..")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archive: %v\n", err)
		os.Exit(1)
	}
}
