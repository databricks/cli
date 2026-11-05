package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <commit> [pathspec...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Prints a content fingerprint over the tracked files matching pathspec at commit.\n")
		os.Exit(1)
	}

	commit := os.Args[1]
	pathspecs := os.Args[2:]

	entries, err := TreeEntriesAt(commit, pathspecs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing tree: %v\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		// No file matched the pathspecs. Emitting a hash of the empty set would
		// let an unrelated later commit collide with it; a caller must decide
		// what "nothing to fingerprint" means, so make it an explicit error
		// rather than silently returning a reusable hash.
		fmt.Fprintf(os.Stderr, "Error: no tracked files matched at %s\n", commit)
		os.Exit(1)
	}

	fmt.Println(Fingerprint(entries))
}
