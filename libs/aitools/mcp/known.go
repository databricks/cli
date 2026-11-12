package mcp

// systemAILocation is the schema that ships the Databricks managed MCP services.
const systemAILocation = "system.ai"

// KnownService is the CLI's own description of a system.ai service, keyed by leaf name.
type KnownService struct {
	Label       string
	Summary     string
	Recommended bool
	Beta        bool
	// Note is printed after install or sign-in. It is empty when there is nothing to add.
	Note string
}

// knownSystemAI describes the system.ai services. Discovery from the API decides what is
// offered; this table only adds labels and notes, so an unlisted service still appears.
var knownSystemAI = map[string]KnownService{
	"genie_one_mcp": {
		Label: "Genie One", Summary: "Ask questions about your data",
		Recommended: true,
	},
	"dbsql": {
		Label: "Databricks SQL", Summary: "Run SQL on a SQL warehouse",
		Beta: true, Note: "Available with the account-level Unity Gateway.",
	},
	"web_search": {
		Label: "Web search", Summary: "Search the public web",
		Beta: true, Note: "Not for consumer-only users; off in HIPAA workspaces.",
	},
	"sandbox": {
		Label: "Sandbox", Summary: "Run Python, SQL or shell in a sandbox",
		Beta: true,
	},
	"github": {
		Label: "GitHub", Summary: "Repositories, issues, pull requests",
		Note: "Public repositories only until a repository owner installs https://github.com/apps/databricks-github-connector",
	},
	"slack": {
		Label: "Slack", Summary: "Channels and messages",
	},
	"atlassian": {
		Label: "Jira and Confluence",
	},
	"google_drive": {
		Label: "Google Drive", Summary: "Drive, Docs, Sheets, Slides",
	},
	"google_calendar": {
		Label: "Google Calendar",
	},
	"gmail": {
		Label: "Gmail",
	},
	"microsoft_365": {
		Label: "Microsoft 365", Summary: "SharePoint, Outlook, Teams",
		Note: "`.All` scopes usually need tenant admin consent.",
	},
}
