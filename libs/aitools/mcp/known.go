package mcp

// systemAILocation is the schema that ships the Databricks managed MCP services.
const systemAILocation = "system.ai"

// builtinServicesDocs is the public documentation page for the system.ai MCP services.
const builtinServicesDocs = "https://docs.databricks.com/aws/en/agents/mcp-tools/built-in-mcp-services"

// KnownService is the CLI's own description of a system.ai service, keyed by leaf name.
type KnownService struct {
	Leaf        string
	Label       string
	Summary     string
	Recommended bool
	Beta        bool
	// Note is printed after install or sign-in. It is empty when there is nothing to add.
	Note string
	Docs string
}

// knownSystemAI describes the system.ai services. Discovery from the API decides what is
// offered; this table only adds labels and notes, so an unlisted service still appears.
var knownSystemAI = map[string]KnownService{
	"genie_one_mcp": {
		Leaf: "genie_one_mcp", Label: "Genie One", Summary: "Ask questions about your data",
		Recommended: true, Docs: builtinServicesDocs,
	},
	"dbsql": {
		Leaf: "dbsql", Label: "Databricks SQL", Summary: "Run SQL on a SQL warehouse",
		Beta: true, Note: "Available with the account-level Unity Gateway.", Docs: builtinServicesDocs,
	},
	"web_search": {
		Leaf: "web_search", Label: "Web search", Summary: "Search the public web",
		Beta: true, Note: "Not for consumer-only users; off in HIPAA workspaces.", Docs: builtinServicesDocs,
	},
	"sandbox": {
		Leaf: "sandbox", Label: "Sandbox", Summary: "Run Python, SQL or shell in a sandbox",
		Beta: true, Docs: builtinServicesDocs,
	},
	"github": {
		Leaf: "github", Label: "GitHub", Summary: "Repositories, issues, pull requests",
		Note: "Public repositories only until a repository owner installs https://github.com/apps/databricks-github-connector",
		Docs: builtinServicesDocs,
	},
	"slack": {
		Leaf: "slack", Label: "Slack", Summary: "Channels and messages", Docs: builtinServicesDocs,
	},
	"atlassian": {
		Leaf: "atlassian", Label: "Jira and Confluence", Docs: builtinServicesDocs,
	},
	"google_drive": {
		Leaf: "google_drive", Label: "Google Drive", Summary: "Drive, Docs, Sheets, Slides", Docs: builtinServicesDocs,
	},
	"google_calendar": {
		Leaf: "google_calendar", Label: "Google Calendar", Docs: builtinServicesDocs,
	},
	"gmail": {
		Leaf: "gmail", Label: "Gmail", Docs: builtinServicesDocs,
	},
	"microsoft_365": {
		Leaf: "microsoft_365", Label: "Microsoft 365", Summary: "SharePoint, Outlook, Teams",
		Note: "`.All` scopes usually need tenant admin consent.", Docs: builtinServicesDocs,
	},
}
