//go:build !wasm

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/template"
)

func main() {
	// Use command line arg for template name if provided, otherwise default
	templateName := os.Args[1]
	paramsString := os.Args[2]
	helpersString := os.Args[3]

	var params map[string]any
	err := json.Unmarshal([]byte(paramsString), &params)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing parameters: %v\n", err)
		os.Exit(1)
	}

	var helpers map[string]string
	err = json.Unmarshal([]byte(helpersString), &helpers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing helpers: %v\n", err)
		os.Exit(1)
	}

	out, err := template.Render(templateName, params, helpers)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	// Output rendered result as indented JSON
	outJSON, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	fmt.Printf("%s\n", outJSON)
}
