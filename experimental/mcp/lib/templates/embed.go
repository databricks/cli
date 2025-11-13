package templates

import (
	"embed"
)

//go:embed trpc/*
var trpcFS embed.FS

// GetTRPCTemplate returns the embedded TRPC template
func GetTRPCTemplate() Template {
	return NewEmbeddedTemplate(
		"TRPC",
		"Modern full-stack template with tRPC, TypeScript, and React",
		trpcFS,
		"trpc",
	)
}
