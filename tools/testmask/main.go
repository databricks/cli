package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	baseRef := os.Getenv("GITHUB_BASE_REF")
	if baseRef == "" {
		baseRef = "HEAD"
	}

	headRef := os.Getenv("GITHUB_HEAD_REF")
	if headRef == "" {
		headRef = "HEAD"
	}

	// Accept CLI arguments for testing
	if len(os.Args) == 3 {
		headRef = os.Args[1]
		baseRef = os.Args[2]
	}

	changedFiles, err := GetChangedFiles(headRef, baseRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	targets := GetTargets(changedFiles)
	fmt.Println(strings.Join(targets, " "))
}
