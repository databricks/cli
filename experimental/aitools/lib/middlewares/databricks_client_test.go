package middlewares

import (
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/stretchr/testify/assert"
)

func TestFormatAuthError(t *testing.T) {
	tests := []struct {
		name     string
		profiles profile.Profiles
		want     string
	}{
		{
			name: "with profiles",
			profiles: profile.Profiles{
				{Name: "DEFAULT", Host: "https://dbc-a.cloud.databricks.com"},
				{Name: "dev", Host: "https://dbc-b.cloud.databricks.com"},
			},
			want: "Not authenticated to Databricks\n\n" +
				"I need to know either the Databricks workspace URL or the Databricks profile name.\n\n" +
				"The available profiles are:\n\n" +
				"- DEFAULT (https://dbc-a.cloud.databricks.com)\n" +
				"- dev (https://dbc-b.cloud.databricks.com)\n\n" +
				"IMPORTANT: YOU MUST ASK the user which of the configured profiles or databricks workspace URL they want to use.\n" +
				"Then set the DATABRICKS_HOST and DATABRICKS_TOKEN environment variables, or use a named profile via DATABRICKS_CONFIG_PROFILE.\n\n" +
				"Do not run anything else before authenticating successfully.\n",
		},
		{
			name: "without profiles",
			want: "Not authenticated to Databricks\n\n" +
				"I need to know either the Databricks workspace URL or the Databricks profile name.\n\n" +
				"The available profiles are:\n\n\n" +
				"IMPORTANT: YOU MUST ASK the user which of the configured profiles or databricks workspace URL they want to use.\n" +
				"Then set the DATABRICKS_HOST and DATABRICKS_TOKEN environment variables, or use a named profile via DATABRICKS_CONFIG_PROFILE.\n\n" +
				"Do not run anything else before authenticating successfully.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatAuthError(tt.profiles))
		})
	}
}
