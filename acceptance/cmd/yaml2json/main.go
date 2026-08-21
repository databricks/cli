// Command yaml2json prints a YAML file as JSON, parsed the way the bundle parses it.
//
// Acceptance helpers are stdlib-only Python and cannot parse YAML; this lives under
// acceptance/ rather than the product CLI so it stays test-only.
package main

import (
	"fmt"
	"os"

	"github.com/databricks/cli/libs/dyn/jsonsaver"
	"github.com/databricks/cli/libs/dyn/yamlloader"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE\n", os.Args[0])
		os.Exit(1)
	}

	if err := run(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	v, err := yamlloader.LoadYAML(path, f)
	if err != nil {
		return err
	}

	buf, err := jsonsaver.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(buf)
	return err
}
