package appkit

import "embed"

// DocsFS embeds the appkit template documentation.
//
//go:embed template/{{.project_name}}/docs/*.md
var DocsFS embed.FS
