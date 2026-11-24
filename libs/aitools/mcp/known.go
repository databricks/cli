package mcp

import (
	"maps"
	"slices"
	"strings"
)

// SystemAILocation is the schema that ships the Databricks managed MCP services.
const SystemAILocation = "system.ai"

// KnownService is the CLI's own description of a system.ai service, keyed by leaf name.
type KnownService struct {
	Label       string
	Summary     string
	Recommended bool
	Beta        bool
	// Note is printed after install or sign-in. It is empty when there is nothing to add.
	Note string
}

// knownSystemAI describes the system.ai services listed at
// https://docs.databricks.com/aws/en/agents/mcp-tools/built-in-mcp-services. Discovery from
// the API decides what is offered; this table only adds labels and notes, so an unlisted
// service still appears.
var knownSystemAI = map[string]KnownService{
	"genie_one_mcp": {
		Label: "Genie One", Summary: "Ask questions about your data",
		Recommended: true,
	},
	"dbsql": {
		Label: "Databricks SQL", Summary: "Run SQL on a SQL warehouse",
		Beta: true, Note: "Needs an account admin to turn on the Enhanced Unity Gateway preview.",
	},
	"web_search": {
		Label: "Web search", Summary: "Search the public web",
		Beta: true, Note: "Needs the Enhanced Unity Gateway preview. Not for consumer-only users; off in HIPAA workspaces.",
	},
	"sandbox": {
		Label: "Sandbox", Summary: "Run Python or shell in a sandbox",
		Beta: true, Note: "Needs the Enhanced Unity Gateway preview. Not for consumer-only users; network egress is not guaranteed.",
	},
	"github": {
		Label: "GitHub", Summary: "Repositories, issues, pull requests",
		Note: "Public repositories only until a repository owner installs https://github.com/apps/databricks-github-connector (on AWS GovCloud, https://github.com/apps/databricks-gh-connector-gov-cloud).",
	},
	"slack": {
		Label: "Slack", Summary: "Channels and messages",
	},
	"atlassian": {
		Label: "Jira and Confluence", Summary: "Issues and pages",
	},
	"google_drive": {
		Label: "Google Drive", Summary: "Drive, Docs, Sheets, Slides",
	},
	"google_calendar": {
		Label: "Google Calendar", Summary: "Calendar events",
	},
	"gmail": {
		Label: "Gmail", Summary: "Email messages",
	},
	"microsoft_365": {
		Label: "Microsoft 365", Summary: "SharePoint, Outlook, Teams",
		Note: "Scopes ending in .All usually need tenant admin consent.",
	},
}

// KnownLeaves returns the leaf names of the system.ai services in the CLI's table, sorted.
func KnownLeaves() []string {
	return slices.Sorted(maps.Keys(knownSystemAI))
}

// KnownOf returns the CLI's description of a service, or nil when it is not a system.ai service in the table. The leaf
// is compared without case, as Unity Catalog names are.
func KnownOf(fullName string) *KnownService {
	known, ok := knownSystemAI[strings.ToLower(LeafOf(fullName))]
	if !ok || !inSystemAI(fullName) {
		return nil
	}
	return &known
}

// ProviderLabel names the provider of a service in messages: the known label for a system.ai
// service, else the full name.
func ProviderLabel(fullName string) string {
	if known := KnownOf(fullName); known != nil {
		return known.Label
	}
	return fullName
}
