//go:build !wasm

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

func main() {
	// Use command line arg for template name if provided, otherwise default
	templateName := os.Args[1]
	paramsString := os.Args[2]

	var params map[string]string
	// AI TODO: parse paramsString json into params

	out := Render(templateName, params)

	// Output rendered result as indented JSON
	outJSON, _ := json.MarshalIndent(out, "", "  ")
	fmt.Printf("%s\n", outJSON)
}
