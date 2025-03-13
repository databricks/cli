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
	templateName := "default-python"
	if len(os.Args) > 1 {
		templateName = os.Args[1]
	}
	
	tmpl := template.GetTemplate(templateName)
	// Output template info as indented JSON
	tmplJSON, _ := json.MarshalIndent(tmpl, "", "  ")
	fmt.Printf("Template: %s\n%s\n", templateName, tmplJSON)

	// Create sample parameters
	params := map[string]string{"param1": "value1"}
	
	// Render the template with the given name and parameters
	out := Render(templateName, params)
	
	// Output rendered result as indented JSON
	outJSON, _ := json.MarshalIndent(out, "", "  ")
	fmt.Printf("Rendered output:\n%s\n", outJSON)
}
