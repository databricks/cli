package middlewares

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/experimental/aitools/lib/session"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go"
)

const (
	DatabricksClientKey = "databricks_client"
)

// GetAvailableProfiles returns all available profiles from ~/.databrickscfg.
func GetAvailableProfiles(ctx context.Context) profile.Profiles {
	profiles, err := profile.DefaultProfiler.LoadProfiles(ctx, profile.MatchAllProfiles)
	if err != nil {
		// If we can't load profiles, return empty list (config file might not exist)
		return profile.Profiles{}
	}
	return profiles
}

func GetDatabricksClient(ctx context.Context) (*databricks.WorkspaceClient, error) {
	sess, err := session.GetSession(ctx)
	if err != nil {
		return nil, err
	}
	w, ok := sess.Get(DatabricksClientKey)
	if !ok {
		return nil, newAuthError(ctx)
	}
	return w.(*databricks.WorkspaceClient), nil
}

func newAuthError(ctx context.Context) error {
	return errors.New(formatAuthError(GetAvailableProfiles(ctx)))
}

func formatAuthError(profiles profile.Profiles) string {
	var b strings.Builder
	b.WriteString("Not authenticated to Databricks\n\n")
	b.WriteString("I need to know either the Databricks workspace URL or the Databricks profile name.\n\n")
	b.WriteString("The available profiles are:\n\n")
	for _, p := range profiles {
		fmt.Fprintf(&b, "- %s (%s)\n", p.Name, p.Host)
	}
	b.WriteString("\n")
	b.WriteString("IMPORTANT: YOU MUST ASK the user which of the configured profiles or databricks workspace URL they want to use.\n")
	b.WriteString("Then set the DATABRICKS_HOST and DATABRICKS_TOKEN environment variables, or use a named profile via DATABRICKS_CONFIG_PROFILE.\n\n")
	b.WriteString("Do not run anything else before authenticating successfully.\n")
	return b.String()
}
