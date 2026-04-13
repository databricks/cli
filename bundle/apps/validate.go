package apps

import (
	"context"
	"fmt"
	"regexp"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/apps"
)

// resourceReferencePattern matches ${resources.<type>.<key>.<field>} variable references.
var resourceReferencePattern = regexp.MustCompile(`\$\{resources\.(\w+)\.([^.]+)\.\w+\}`)

type validate struct{}

func (v *validate) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	usedSourceCodePaths := make(map[string]string)

	for key, app := range b.Config.Resources.Apps {
		if app.SourceCodePath == "" && app.GitSource == nil {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Missing app source code path or git source",
				Detail:    fmt.Sprintf("app resource '%s' should have either source_code_path or git_source field", key),
				Locations: b.Config.GetLocations("resources.apps." + key),
			})
			continue
		}

		if app.SourceCodePath != "" && app.GitSource != nil {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Both source_code_path and git_source fields are set",
				Detail:    fmt.Sprintf("app resource '%s' should have either source_code_path or git_source field, not both", key),
				Locations: b.Config.GetLocations("resources.apps." + key),
			})
			continue
		}

		if _, ok := usedSourceCodePaths[app.SourceCodePath]; ok {
			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   "Duplicate app source code path",
				Detail:    fmt.Sprintf("app resource '%s' has the same source code path as app resource '%s', this will lead to the app configuration being overriden by each other", key, usedSourceCodePaths[app.SourceCodePath]),
				Locations: b.Config.GetLocations(fmt.Sprintf("resources.apps.%s.source_code_path", key)),
			})
		}
		usedSourceCodePaths[app.SourceCodePath] = key

		diags = diags.Extend(warnForAppResourcePermissions(b, key, app))
	}

	return diags
}

// appResourceRef extracts resource references from an app resource entry.
// When resourceType is empty the reference matches any bundle resource type
// extracted from the variable reference pattern.
func appResourceRef(r apps.AppResource) (appResourceReference, bool) {
	switch {
	case r.Job != nil:
		return appResourceReference{"jobs", r.Job.Id, string(r.Job.Permission)}, true
	case r.SqlWarehouse != nil:
		return appResourceReference{"sql_warehouses", r.SqlWarehouse.Id, string(r.SqlWarehouse.Permission)}, true
	case r.ServingEndpoint != nil:
		return appResourceReference{"model_serving_endpoints", r.ServingEndpoint.Name, string(r.ServingEndpoint.Permission)}, true
	case r.Experiment != nil:
		return appResourceReference{"experiments", r.Experiment.ExperimentId, string(r.Experiment.Permission)}, true
	case r.Postgres != nil:
		if r.Postgres.Branch != "" {
			return appResourceReference{"postgres_projects", r.Postgres.Branch, string(r.Postgres.Permission)}, true
		}
		return appResourceReference{
			"database_instances", r.Postgres.Database, string(r.Postgres.Permission),
		}, true
	case r.UcSecurable != nil:
		return appResourceReference{string(r.UcSecurable.SecurableType), r.UcSecurable.SecurableFullName, string(r.UcSecurable.Permission)}, true
	default:
		return appResourceReference{}, false
	}
}

type appResourceReference struct {
	resourceType string
	refValue     string
	permission   string
}

// hasPermissions checks if a bundle resource at the given dyn path has a non-empty permissions list.
func hasPermissions(b *bundle.Bundle, resourcePath string) bool {
	pv, err := dyn.Get(b.Config.Value(), resourcePath+".permissions")
	if err != nil {
		return false
	}
	s, ok := pv.AsSequence()
	return ok && len(s) > 0
}

// hasAppSPInPermissions checks if any permission entry for the given resource
// references the app's service principal via variable interpolation.
func hasAppSPInPermissions(b *bundle.Bundle, resourcePath, appKey string) bool {
	appSPRef := fmt.Sprintf("${resources.apps.%s.service_principal_client_id}", appKey)
	pv, err := dyn.Get(b.Config.Value(), resourcePath+".permissions")
	if err != nil {
		return false
	}
	s, ok := pv.AsSequence()
	if !ok {
		return false
	}
	for _, entry := range s {
		spn, err := dyn.Get(entry, "service_principal_name")
		if err != nil {
			continue
		}
		if str, ok := spn.AsString(); ok && str == appSPRef {
			return true
		}
	}
	return false
}

// warnForAppResourcePermissions warns when an app references a bundle resource
// that has explicit permissions but doesn't include the app's service principal.
// Without the SP in the permission list, the second deploy will overwrite the
// app-granted permission on the resource.
// See https://github.com/databricks/cli/issues/4309
func warnForAppResourcePermissions(b *bundle.Bundle, appKey string, app *resources.App) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, ar := range app.Resources {
		ref, ok := appResourceRef(ar)
		if !ok {
			continue
		}

		matches := resourceReferencePattern.FindStringSubmatch(ref.refValue)
		if len(matches) < 3 {
			continue
		}
		refType, resourceKey := matches[1], matches[2]

		resourcePath := fmt.Sprintf("resources.%s.%s", refType, resourceKey)
		if !hasPermissions(b, resourcePath) {
			continue
		}

		if hasAppSPInPermissions(b, resourcePath, appKey) {
			continue
		}

		appPath := "resources.apps." + appKey
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  fmt.Sprintf("app %q references %s %q which has permissions set. To prevent permission override after deploying the app, please add the app service principal to the %s permissions", appKey, refType, resourceKey, refType),
			Detail: fmt.Sprintf(
				"Add the following section to the %s permissions:\n\n"+
					"  resources:\n"+
					"    %s:\n"+
					"      %s:\n"+
					"        permissions:\n"+
					"          - level: %s\n"+
					"            service_principal_name: ${resources.apps.%s.service_principal_client_id}\n",
				refType,
				refType,
				resourceKey,
				ref.permission,
				appKey,
			),
			Paths:     []dyn.Path{dyn.MustPathFromString(appPath)},
			Locations: b.Config.GetLocations(appPath),
		})
	}

	return diags
}

func (v *validate) Name() string {
	return "apps.Validate"
}

func Validate() bundle.Mutator {
	return &validate{}
}
