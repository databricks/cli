package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <output_path>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Creates tar.gz archive containing git-tracked files + downloaded tools (Go, UV, jq)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s ./repo-backup.tar.gz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s /tmp/my-repo.tar.gz\n", os.Args[0])
		os.Exit(1)
	}

	outputPath := os.Args[1]
	if err := createGitArchive(outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating archive: %v\n", err)
		os.Exit(1)
	}
}
