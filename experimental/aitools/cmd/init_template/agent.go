package init_template

import (
	"errors"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/spf13/cobra"
)

const (
	agentTemplateRepo = "https://github.com/databricks/cli"
	agentTemplateDir  = "experimental/aitools/templates/agent-openai-agents-sdk"
	agentBranch       = "main"
	agentPathEnvVar   = "DATABRICKS_AGENT_TEMPLATE_PATH"
)

// newAgentCmd creates the agent subcommand for init-template.
func newAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Initialize an OpenAI Agents SDK project",
		Args:  cobra.NoArgs,
		Long: `Initialize an OpenAI Agents SDK project for building conversational agents.

This creates a project with:
- OpenAI Agents SDK integration for building conversational agents
- Built-in chat UI and API endpoint for invoking the agent
- MLflow tracing and evaluation setup
- Access to Databricks built-in tools (code interpreter, etc.)
- Example agent implementation with MCP server support

Examples:
  experimental aitools tools init-template agent --name my-agent
  experimental aitools tools init-template agent --name my-agent --output-dir ./projects

Environment variables:
  DATABRICKS_AGENT_TEMPLATE_PATH  Override template source with local path (for development)

After initialization:
  cd <project-name>
  ./scripts/quickstart.sh  # Set up environment and start server
  uv run start-app         # Start agent server at http://localhost:8000
`,
	}

	var name string
	var outputDir string

	cmd.Flags().StringVar(&name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&outputDir, "output-dir", "", "Directory to write the initialized template to")

	cmd.PreRunE = root.MustWorkspaceClient
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		if name == "" {
			return errors.New("--name is required. Example: init-template agent --name my-agent")
		}

		configMap := map[string]any{
			"project_name": name,
		}

		// Resolve template source: env var override or default remote
		templatePathOrUrl := os.Getenv(agentPathEnvVar)
		templateDir := ""
		branch := ""

		if templatePathOrUrl == "" {
			templatePathOrUrl = agentTemplateRepo
			templateDir = agentTemplateDir
			branch = agentBranch
		}

		return MaterializeTemplate(ctx, TemplateConfig{
			TemplatePath: templatePathOrUrl,
			TemplateName: "agent-openai-agents-sdk",
			TemplateDir:  templateDir,
			Branch:       branch,
		}, configMap, name, outputDir)
	}
	return cmd
}
