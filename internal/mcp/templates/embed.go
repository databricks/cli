package templates

import (
	"embed"

	"github.com/databricks/cli/experimental/mcp/lib/templates"
)

//go:embed trpc/*
var trpcFS embed.FS

// GetTRPCTemplate returns the embedded TRPC template
func GetTRPCTemplate() templates.Template {
	return templates.NewEmbeddedTemplate(
		"TRPC",
		"Modern full-stack template with tRPC, TypeScript, and React",
		trpcFS,
		"trpc",
	)
}
