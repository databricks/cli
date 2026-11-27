// Command genschema compresses the bitmap schema text in INPUT (as printed by
// 'databricks bundle bitmap update-schema') and writes the same bytes to each
// OUTPUT: the copy the CLI embeds (bundle/bitmap/schema.deflate) and the one the
// databricks-bundles decoder reads (python/databricks/bundles/_bitmap).
package main

import (
	"fmt"
	"os"

	"github.com/databricks/cli/bundle/bitmap"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: genschema INPUT OUTPUT...")
		os.Exit(2)
	}
	if err := generate(os.Args[1], os.Args[2:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(input string, outputs []string) error {
	text, err := os.ReadFile(input)
	if err != nil {
		return err
	}
	compressed, err := bitmap.CompressSchema(text)
	if err != nil {
		return err
	}
	for _, path := range outputs {
		if err := os.WriteFile(path, compressed, 0o644); err != nil {
			return err
		}
	}
	return nil
}
