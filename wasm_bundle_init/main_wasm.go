//go:build wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

func renderTemplateWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 3 {
			return "Invalid number of arguments passed. Expected: template name and parameters JSON"
		}

		templateName := args[0].String()
		paramsJSON := args[1].String()
		helpersJSON := args[2].String()

		var params map[string]any
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return fmt.Sprintf("Error parsing JSON: %s", err.Error())
		}

		var helpers map[string]string
		if err := json.Unmarshal([]byte(helpersJSON), &helpers); err != nil {
			return fmt.Sprintf("Error parsing JSON: %s", err.Error())
		}

		out := Render(templateName, params, helpers)

		pretty, err := json.Marshal(out)
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
