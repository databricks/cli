//go:build wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"

	"github.com/databricks/cli/libs/template"
)

func wrapError(msg string) string {
	x := map[string]string{"error": msg}
	out, err := json.Marshal(x)
	if err != nil {
		return msg + "\n\n" + err.Error()
	}
	return string(out)
}

func renderTemplateWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 3 {
			return wrapError("Invalid number of arguments passed. Expected: template name and parameters JSON")
		}

		templateName := args[0].String()
		paramsJSON := args[1].String()
		helpersJSON := args[2].String()

		var params map[string]any
		if err := json.Unmarshal([]byte(paramsJSON), &params); err != nil {
			return wrapError(fmt.Sprintf("Error parsing params: %s", err.Error()))
		}

		var helpers map[string]string
		if err := json.Unmarshal([]byte(helpersJSON), &helpers); err != nil {
			return wrapError(fmt.Sprintf("Error parsing helpers: %s", err.Error()))
		}

		out, err := template.Render(templateName, params, helpers)
		if err != nil {
			return wrapError(err.Error())
		}

		pretty, err := json.Marshal(out)
		if err != nil {
			return wrapError(err.Error())
		}
		return string(pretty)
	})
}

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set("RenderTemplate", renderTemplateWrapper())

	select {}
}
