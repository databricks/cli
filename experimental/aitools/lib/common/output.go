package common

import "fmt"

const (
	headerLine = "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
)

// FormatBrandedHeader creates a branded header with the given emoji and message.
func FormatBrandedHeader(emoji, message string) string {
	return fmt.Sprintf("%s\n%s Databricks AI Tools MCP server: %s\n%s\n\n",
		headerLine, emoji, message, headerLine)
}
