// Command pindepversions fetches the compatibility manifest and writes
// the resolved entry for a given CLI version. Used at release build time
// to embed version-specific dep versions into the binary.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/depversions"
)

const defaultOutput = "internal/build/dep_versions.json"

func main() {
	version := flag.String("version", "", "CLI version to resolve (required)")
	output := flag.String("o", defaultOutput, "output file path")
	flag.Parse()

	if *version == "" {
		fmt.Fprintln(os.Stderr, "error: --version is required")
		os.Exit(1)
	}

	m, err := depversions.FetchManifest(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not fetch manifest: %v\n", err)
		return
	}

	entry, err := depversions.Resolve(m, *version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not resolve version %s: %v\n", *version, err)
		return
	}

	data, _ := json.Marshal(entry)
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "error: write %s: %v\n", *output, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Pinned for CLI %s: appkit=%s skills=%s\n", *version, entry.AppKit, entry.Skills)
}
