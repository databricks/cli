//go:build !wasm

package main

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/libs/template"
)

func main() {
	tmpl := template.GetTemplate("default-python")
	// Output template info as indented JSON
	tmplJSON, _ := json.MarshalIndent(tmpl, "", "  ")
	fmt.Printf("%s\n", tmplJSON)
	
	out := Render("default-python", map[string]string{"param1": "value1"})
	// Output rendered result as indented JSON
	outJSON, _ := json.MarshalIndent(out, "", "  ")
	fmt.Printf("%s\n", outJSON)
}
