// Command testmask reads Taskfile.yml to decide which CI jobs should run
// based on the set of files changed in a PR.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <head-ref> <base-ref>\n", os.Args[0])
		os.Exit(1)
	}

	headRef := os.Args[1]
	baseRef := os.Args[2]

	repoRoot, err := GitRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding repo root: %v\n", err)
		os.Exit(1)
	}

	mappings, err := LoadTargetMappings(filepath.Join(repoRoot, "Taskfile.yml"))
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
	err = json.NewEncoder(os.Stdout).Encode(targets)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding targets: %v\n", err)
		os.Exit(1)
	}
}
