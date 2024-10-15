package permissions

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

func TryExtendTerraformPermissionError(ctx context.Context, b *bundle.Bundle, err error) diag.Diagnostics {
	_, assistance := analyzeBundlePermissions(b)

	// In a best-effort attempt to provide actionable error messages, we match
	// against a few specific error messages that come from the Jobs and Pipelines API.
	// For matching errors we provide a more specific error message that includes
	// details on how to resolve the issue.
	if !strings.Contains(err.Error(), "cannot update permissions") &&
		!strings.Contains(err.Error(), "permissions on pipeline") &&
		!strings.Contains(err.Error(), "cannot read permissions") &&
		!strings.Contains(err.Error(), "cannot set run_as to user") {
		return nil
	}

	log.Errorf(ctx, "Terraform error during deployment: %v", err.Error())

	// Best-effort attempt to extract the resource name from the error message.
	re := regexp.MustCompile(`databricks_(\w*)\.(\w*)`)
	match := re.FindStringSubmatch(err.Error())
	resource := "resource"
	if len(match) > 1 {
		resource = match[2]
	}

	return diag.Diagnostics{{
		Summary: fmt.Sprintf("permission denied creating or updating %s.\n"+
			"%s\n"+
			"They can redeploy the project to apply the latest set of permissions.\n"+
			"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.",
			resource, assistance),
		Severity: diag.Error,
		ID:       diag.ResourcePermissionDenied,
	}}
}
