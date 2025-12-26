package common

import "fmt"

const (
	headerLine = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

// FormatBrandedHeader creates a branded header with the given emoji and message.
func FormatBrandedHeader(emoji, message string) string {
	return fmt.Sprintf("%s\n%s Databricks MCP: %s\n%s\n\n",
		headerLine, emoji, message, headerLine)
}

// FormatValidationSuccess formats a success message for validation.
func FormatValidationSuccess(message string) string {
	header := FormatBrandedHeader("🔍", "Validating your app")
	return fmt.Sprintf("%s✅ %s\n", header, message)
}

// FormatValidationFailure formats a failure message for validation.
func FormatValidationFailure(message string, exitCode int, stdout, stderr string) string {
	header := FormatBrandedHeader("🔍", "Validating your app")
	return fmt.Sprintf("%s❌ %s\n\nExit code: %d\n\nStdout:\n%s\n\nStderr:\n%s\n",
		header, message, exitCode, stdout, stderr)
}

// FormatDeploymentSuccess formats a success message for deployment.
func FormatDeploymentSuccess(appName, appURL string) string {
	header := FormatBrandedHeader("🚢", "Deploying to production")
	return fmt.Sprintf("%s✅ App '%s' deployed successfully!\n\n🌐 URL: %s\n",
		header, appName, appURL)
}

// FormatDeploymentFailure formats a failure message for deployment.
func FormatDeploymentFailure(appName, message string) string {
	header := FormatBrandedHeader("🚢", "Deploying to production")
	return fmt.Sprintf("%s❌ Deployment failed for '%s'\n\n%s\n",
		header, appName, message)
}

// FormatProjectScaffoldSuccess formats a success message for project scaffolding.
func FormatProjectScaffoldSuccess(templateName, workDir string, filesCopied int) string {
	header := FormatBrandedHeader("📦", "Project scaffolded successfully")
	return fmt.Sprintf("%s✅ Created %s project at %s\n\nFiles copied: %d\n",
		header, templateName, workDir, filesCopied)
}
