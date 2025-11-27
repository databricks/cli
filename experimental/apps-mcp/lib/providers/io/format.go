package io

import (
	"fmt"
	"path/filepath"
)

// formatScaffoldResult formats a ScaffoldResult for display
func formatScaffoldResult(result *ScaffoldResult) string {
	return fmt.Sprintf(
		"Successfully scaffolded %s template to %s\n\n"+
			"Template: %s\n\n"+
			"App name: %s\n\n"+
			"IMPORTANT: run `cd %s && npm install > /dev/null 2>&1` to install dependencies.\n\n"+
			"It is recomended to run the app in the background immediately after scaffolding using `npm install && npm run dev`. Then directly open http://localhost:8000 in the browser so the user can follow the progress.\n\n"+
			"IMPORTANT: Make sure to read %s before proceeding with the project!!!\n\n",
		result.TemplateName,
		result.WorkDir,
		result.TemplateName,
		result.AppName,
		result.AppName,
		filepath.Join(result.WorkDir, result.AppName, "CLAUDE.md"),
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
