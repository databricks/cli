//go:build wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

func renderTemplateWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 2 {
			return "Invalid number of arguments passed. Expected: template name and parameters JSON"
		}
		
		templateName := args[0].String()
		inputJSON := args[1].String()
		fmt.Printf("Template: %s, input: %s\n", templateName, inputJSON)

		// Parse the JSON input into a map
		var params map[string]string
		if err := json.Unmarshal([]byte(inputJSON), &params); err != nil {
			return fmt.Sprintf("Error parsing JSON: %s", err.Error())
		}

		out := Render(templateName, params)

		pretty, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err.Error()
		}
		return string(pretty)
	})
}

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set("RenderTemplate", renderTemplateWrapper())

	select {}
}
