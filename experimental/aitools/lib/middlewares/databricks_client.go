package middlewares

import (
	"context"
	"errors"

	"github.com/databricks/cli/experimental/aitools/lib/prompts"
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
	// Prepare template data
	data := map[string]any{
		"Profiles": GetAvailableProfiles(ctx),
	}
	return errors.New(prompts.MustExecuteTemplate("auth_error.tmpl", data))
}
