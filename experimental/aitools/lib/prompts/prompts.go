package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed *.tmpl
var promptTemplates embed.FS

// ExecuteTemplate loads and executes a template with the given name and data.
func ExecuteTemplate(name string, data any) (string, error) {
	tmplContent, err := promptTemplates.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(tmplContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// MustExecuteTemplate is like ExecuteTemplate but panics on error.
// Use this only when template execution errors are programming errors.
func MustExecuteTemplate(name string, data any) string {
	result, err := ExecuteTemplate(name, data)
	if err != nil {
		panic(err)
	}
	return result
}

// LoadTemplate loads a template without executing it.
// Returns the raw template content as a string.
func LoadTemplate(name string) (string, error) {
	tmplContent, err := promptTemplates.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("failed to read template %s: %w", name, err)
	}
	return string(tmplContent), nil
}

// MustLoadTemplate is like LoadTemplate but panics on error.
func MustLoadTemplate(name string) string {
	result, err := LoadTemplate(name)
	if err != nil {
		panic(err)
	}
	return result
}

// TemplateExists checks if a template with the given name exists.
func TemplateExists(name string) bool {
	_, err := promptTemplates.ReadFile(name)
	return err == nil
}
