package templates

import (
	"embed"
)

//go:embed appkit/*
var appkitFS embed.FS

// GetAppKitTemplate returns the embedded AppKit template
func GetAppKitTemplate() Template {
	return NewEmbeddedTemplate(
		"AppKit",
		"Modern full-stack template with AppKit, TypeScript, and React",
		appkitFS,
		"appkit",
	)
}
