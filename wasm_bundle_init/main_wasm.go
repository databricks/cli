//go:build wasm

package main

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

func jsonWrapper() js.Func {
	jsonFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 1 {
			return "Invalid no of arguments passed"
		}
		inputJSON := args[0].String()
		fmt.Printf("input %s\n", inputJSON)

		out := Render("default-python", map[string]string{"param1": "value1"})

		pretty, err := json.MarshalIndent(out, "", "  ")
		if err != nil {
			return err.Error()
		}
		return string(pretty)
	})
	return jsonFunc
}

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set("formatJSON", jsonWrapper())

	select {}
}
