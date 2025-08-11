package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> [args...]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  archive <output_path>     - Create tar.gz archive of git-tracked files + downloaded tools\n")
		fmt.Fprintf(os.Stderr, "  go <architecture>         - Download & extract Go for specified architecture (amd64/arm64)\n")
		fmt.Fprintf(os.Stderr, "  uv <architecture>         - Download & extract UV for specified architecture (amd64/arm64)\n")
		fmt.Fprintf(os.Stderr, "  jq <architecture>         - Download jq for specified architecture (amd64/arm64)\n")
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s archive ./repo-backup.tar.gz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s archive /tmp/my-repo.tar.gz\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s go arm64\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s go amd64\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s uv arm64\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s uv amd64\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s jq arm64\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s jq amd64\n", os.Args[0])
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "archive":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s archive <output_path>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Example: %s archive ./repo-backup.tar.gz\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Creates archive containing git-tracked files + downloaded tools (Go, UV, jq)\n")
			os.Exit(1)
		}

		outputPath := os.Args[2]
		if err := createGitArchive(outputPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating archive: %v\n", err)
			os.Exit(1)
		}

	case "go":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s go <architecture>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Architecture: amd64 or arm64\n")
			fmt.Fprintf(os.Stderr, "Downloads and extracts Go to ./bin/go/\n")
			os.Exit(1)
		}

		arch := strings.ToLower(os.Args[2])
		if err := downloadGoForLinux(arch); err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading Go: %v\n", err)
			os.Exit(1)
		}

	case "uv":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s uv <architecture>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Architecture: amd64 or arm64\n")
			fmt.Fprintf(os.Stderr, "Downloads and extracts UV to ./bin/\n")
			os.Exit(1)
		}

		arch := strings.ToLower(os.Args[2])
		if err := downloadUV(arch); err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading UV: %v\n", err)
			os.Exit(1)
		}

	case "jq":
		if len(os.Args) != 3 {
			fmt.Fprintf(os.Stderr, "Usage: %s jq <architecture>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Architecture: amd64 or arm64\n")
			fmt.Fprintf(os.Stderr, "Downloads jq to ./downloads/\n")
			os.Exit(1)
		}

		arch := strings.ToLower(os.Args[2])
		if err := downloadJq(arch); err != nil {
			fmt.Fprintf(os.Stderr, "Error downloading jq: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Use '%s' without arguments to see usage\n", os.Args[0])
		os.Exit(1)
	}
}
