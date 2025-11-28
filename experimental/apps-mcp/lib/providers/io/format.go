package io

import (
	"fmt"
	"path/filepath"
)

// formatScaffoldResult formats a ScaffoldResult for display
func formatScaffoldResult(result *ScaffoldResult) string {
	return fmt.Sprintf(
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n"+
			"🚀 Databricks MCP: App scaffolded successfully\n"+
			"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"✅ Created %s application at %s\n\n"+
			"Files copied: %d\n\n"+
			"Template: %s\n\n"+
			"It is recomended to run the app in the background immediately after scaffolding using `npm install && npm run dev`. Then directly open http://localhost:8000 in the browser so the user can follow the progress.\n\n"+
			"IMPORTANT: Make sure to read %s before proceeding with the project!!!\n\n",
		result.TemplateName,
		result.WorkDir,
		result.FilesCopied,
		result.TemplateName,
		filepath.Join(result.WorkDir, "CLAUDE.md"),
	)
}

// formatValidateResult formats a ValidateResult for display
func formatValidateResult(result *ValidateResult) string {
	header := "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n" +
		"🔍 Databricks MCP: Validating your app\n" +
		"━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n"

	if result.Success {
		return header + "✅ " + result.Message
	}

	if result.Details == nil {
		return header + "❌ " + result.Message
	}

	return fmt.Sprintf(
		header+"❌ %s\n\nExit code: %d\n\nStdout:\n%s\n\nStderr:\n%s",
		result.Message,
		result.Details.ExitCode,
		result.Details.Stdout,
		result.Details.Stderr,
	)
}
