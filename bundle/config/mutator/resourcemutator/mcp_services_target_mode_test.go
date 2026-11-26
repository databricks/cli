package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("mcp_services", func(r *config.Resources) {
		r.McpServices = map[string]*resources.McpService{
			"mcpservice1": {McpServiceConfig: resources.McpServiceConfig{Parent: "schemas/main.default", McpServiceId: "mcpservice1"}},
		}
	})
}
