//go:build !wasm

package main

import (
	"fmt"

	"github.com/databricks/cli/libs/template"
)

func main() {
	tmpl := template.GetTemplate("default-python")
	fmt.Printf("vim-go %v\n", tmpl)
	out := Render("default-python", map[string]string{"param1": "value1"})
	fmt.Printf("%v\n", out)
}
