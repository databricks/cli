package templates

import (
	"embed"
)

//go:embed trpc/*
var trpcFS embed.FS

//go:embed appkit/*
var appkitFS embed.FS

// GetTRPCTemplate returns the embedded TRPC template
func GetTRPCTemplate() Template {
	return NewEmbeddedTemplate(
		"TRPC",
		"Modern full-stack template with tRPC, TypeScript, and React",
		trpcFS,
		"trpc",
	)
}

// GetAppKitTemplate returns the embedded AppKit template
func GetAppKitTemplate() Template {
	return NewEmbeddedTemplate(
		"AppKit",
		"Modern full-stack template with AppKit, TypeScript, and React",
		appkitFS,
		"appkit",
	)
}
