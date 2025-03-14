//go:build wasm

package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
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

func renderTemplateZipWrapper() js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) != 3 {
			return wrapError("Invalid number of arguments passed. Expected: template name, parameters JSON, and helpers JSON")
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

		// Render the template
		out, err := template.Render(templateName, params, helpers)
		if err != nil {
			return wrapError(err.Error())
		}

		// Create a buffer to write our archive to
		buf := new(bytes.Buffer)
		zipWriter := zip.NewWriter(buf)

		// Add each file to the zip archive
		for path, content := range out {
			// Create a file inside the zip archive
			zipFile, err := zipWriter.Create(path)
			if err != nil {
				return wrapError(fmt.Sprintf("Error creating zip file entry %s: %s", path, err.Error()))
			}

			// Write content to the file
			_, err = zipFile.Write([]byte(content))
			if err != nil {
				return wrapError(fmt.Sprintf("Error writing content to zip file %s: %s", path, err.Error()))
			}
		}

		// Close the zip writer
		if err := zipWriter.Close(); err != nil {
			return wrapError(fmt.Sprintf("Error closing zip writer: %s", err.Error()))
		}

		// Base64 encode the zip content to safely return it through JS
		base64Zip := base64.StdEncoding.EncodeToString(buf.Bytes())

		// Return the base64 encoded zip file
		result := map[string]string{
			"data": base64Zip,
			"type": "application/zip",
		}

		resultJSON, err := json.Marshal(result)
		if err != nil {
			return wrapError(fmt.Sprintf("Error marshaling result: %s", err.Error()))
		}

		return string(resultJSON)
	})
}

func main() {
	fmt.Println("Go Web Assembly")
	js.Global().Set("RenderTemplate", renderTemplateWrapper())
	js.Global().Set("RenderTemplateZip", renderTemplateZipWrapper())
	select {}
}
