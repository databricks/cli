package io

import (
	"fmt"
	"path/filepath"
)

// formatScaffoldResult formats a ScaffoldResult for display
func formatScaffoldResult(result *ScaffoldResult) string {
	return fmt.Sprintf(
		"Successfully scaffolded %s template to %s\n\n"+
			"Files copied: %d\n\n"+
			"Template: %s\n\n"+
			"Make sure to read %s before proceeding with the project\n\n"+
			"It is recomended to run the app in the background immediately after scaffolding using `npm run dev` and open it in your browser so the user can follow the changes to the app.\n\n"+
			"%s",
		result.TemplateName,
		result.WorkDir,
		result.FilesCopied,
		result.TemplateName,
		filepath.Join(result.WorkDir, "CLAUDE.md"),
		result.TemplateDescription,
	)
}

// formatValidateResult formats a ValidateResult for display
func formatValidateResult(result *ValidateResult) string {
	if result.Success {
		return "✓ " + result.Message
	}

	if result.Details == nil {
		return "✗ " + result.Message
	}

	return fmt.Sprintf(
		"✗ %s\n\nExit code: %d\n\nStdout:\n%s\n\nStderr:\n%s",
		result.Message,
		result.Details.ExitCode,
		result.Details.Stdout,
		result.Details.Stderr,
	)
}
