// Command sdkfieldcopies completes explicitly opted-in SDK field mappings.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/databricks/cli/tools/sdkfieldcopies/fix"
)

func main() {
	root := flag.String("root", "..", "CLI repository root")
	flag.Parse()
	if err := fix.Run(context.Background(), *root, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "sdkfieldcopies:", err)
		os.Exit(1)
	}
}
