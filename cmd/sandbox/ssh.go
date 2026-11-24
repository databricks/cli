package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdk "github.com/databricks/databricks-sdk-go/service/sandbox"
)

// ResolveSSHName resolves the sandbox selection shared by SSH commands.
func ResolveSSHName(ctx context.Context, w *databricks.WorkspaceClient, requested string) (string, error) {
	profile := w.Config.Profile
	if profile == "" {
		profile = w.Config.Host
	}
	savedDefault := requested == ""
	if savedDefault {
		requested = getDefault(ctx, profile)
	}
	if requested == "" {
		return "", errors.New("no default sandbox configured — specify a sandbox ID or run 'databricks sandbox default <id>'")
	}
	id, err := resolveLocalID(ctx, profile, strings.TrimPrefix(requested, "sandboxes/"))
	if err != nil {
		return "", err
	}
	if err := validateSandboxID(id); err != nil {
		return "", err
	}
	if id == "" || strings.ContainsAny(id, "?#") || strings.ContainsFunc(id, unicode.IsSpace) {
		return "", fmt.Errorf("invalid sandbox ID %q", id)
	}
	name := "sandboxes/" + id
	if _, err := w.Sandbox.GetSandbox(ctx, sdk.GetSandboxRequest{Name: name}); err != nil {
		if savedDefault && errors.Is(err, apierr.ErrNotFound) {
			return "", fmt.Errorf("saved default %q is unavailable through the Sandbox API; verify the profile or choose another sandbox: %w", id, err)
		}
		return "", err
	}
	return name, nil
}
