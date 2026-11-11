// Command testmask reads Taskfile.yml to decide which CI jobs should run
// based on the set of files changed in a PR and on the tests its head commit
// message names.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <head-ref> <base-ref>\n", os.Args[0])
		os.Exit(1)
	}

	headRef := os.Args[1]
	baseRef := os.Args[2]

	mappings, err := LoadTargetMappings("../../Taskfile.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading target mappings: %v\n", err)
		os.Exit(1)
	}

	changedFiles, err := GetChangedFiles(headRef, baseRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	targets := GetTargets(changedFiles, mappings)

	// Tests named in the head commit message run on cloud, which only happens when the
	// "test" target is selected.
	message, err := GetCommitMessage(headRef)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading commit message: %v\n", err)
		os.Exit(1)
	}
	if NamesTests(message) && !slices.Contains(targets, "test") {
		targets = append(targets, "test")
		slices.Sort(targets)
	}
	err = json.NewEncoder(os.Stdout).Encode(targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding targets: %v\n", err)
		os.Exit(1)
	}
}
