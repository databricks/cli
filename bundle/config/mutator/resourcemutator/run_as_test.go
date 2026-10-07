package resourcemutator

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func allResourceTypes(t *testing.T) []string {
	// Compute supported resource types based on the `Resources{}` struct.
	var resourceTypes []string
	for f := range reflect.TypeFor[config.Resources]().Fields() {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		resourceTypes = append(resourceTypes, name)
	}
	slices.Sort(resourceTypes)

	// Assert the total list of resource supported, as a sanity check that using
	// the struct gives us the correct list of all resources supported. Please
	// also update this check when adding a new resource
	require.Equal(
		t, []string{
			"alerts",
			"apps",
			"catalogs",
			"cluster_policies",
			"clusters",
			"dashboards",
			"database_catalogs",
			"database_instances",
			"experiments",
			"external_locations",
			"genie_spaces",
			"instance_pools",
			"internal_immutable_snapshots",
			"job_runs",
			"jobs",
			"mcp_services",
			"model_provider_services",
			"model_services",
			"model_serving_endpoints",
			"models",
			"pipelines",
			"postgres_branches",
			"postgres_catalogs",
			"postgres_databases",
			"postgres_endpoints",
			"postgres_projects",
			"postgres_roles",
			"postgres_snapshot_schedules",
			"postgres_synced_tables",
			"quality_monitors",
			"registered_models",
			"schemas",
			"secret_scopes",
			"secrets",
			"sql_warehouses",
			"synced_database_tables",
			"vector_search_endpoints",
			"vector_search_indexes",
			"volumes",
		},
		resourceTypes,
	)

	return resourceTypes
}

func TestRunAsWorksForAllowedResources(t *testing.T) {
	config := config.Root{
		Workspace: config.Workspace{
			CurrentUser: &config.User{
				User: &iam.User{
					UserName: "alice",
				},
			},
		},
		RunAs: &jobs.JobRunAs{
			UserName: "bob",
		},
		Resources: config.Resources{
			Jobs: map[string]*resources.Job{
				"job_one": {
					JobSettings: jobs.JobSettings{
						Name: "foo",
					},
				},
				"job_two": {
					JobSettings: jobs.JobSettings{
						Name: "bar",
					},
				},
				"job_three": {
					JobSettings: jobs.JobSettings{
						Name: "baz",
					},
				},
			},
			Models: map[string]*resources.MlflowModel{
				"model_one": {},
			},
			RegisteredModels: map[string]*resources.RegisteredModel{
				"registered_model_one": {},
			},
			Experiments: map[string]*resources.MlflowExperiment{
				"experiment_one": {},
			},
			Pipelines: map[string]*resources.Pipeline{
				"pipeline_one": {},
			},
			Alerts: map[string]*resources.Alert{
				"alert_one": {
					AlertV2: sql.AlertV2{
						DisplayName: "alert",
					},
				},
			},
		},
	}

	b := &bundle.Bundle{
		Config: config,
	}

	diags := bundle.Apply(t.Context(), b, SetRunAs())
	assert.NoError(t, diags.Error())

	for _, job := range b.Config.Resources.Jobs {
		assert.Equal(t, "bob", job.RunAs.UserName)
	}

	for _, alert := range b.Config.Resources.Alerts {
		assert.Equal(t, "bob", alert.RunAs.UserName)
	}
}

// Bundle "run_as" has two modes of operation, each with a different set of
// resources that are supported.
// Cases:
//  1. When the bundle "run_as" identity is same as the current deployment
//     identity. In this case all resources are supported.
//  2. When the bundle "run_as" identity is different from the current
//     deployment identity. In this case only a subset of resources are
//     supported. This subset of resources are defined in the allow list below.
//
// To be a part of the allow list, the resource must satisfy one of the following
// two conditions:
//  1. The resource supports setting a run_as identity to a different user
//     from the owner/creator of the resource. For example, jobs.
//  2. Run as semantics do not apply to the resource's current API. For example,
//     experiments or registered models.
//
// Any resource that is not on the allow list cannot be used when the bundle
// run_as is different from the current deployment user. "bundle validate" must
// return an error if such a resource has been defined, and the run_as identity
// is different from the current deployment identity.
//
// If a resource gains run_as support, review this list and the group support
// classification below. Model serving endpoints are excluded. Dashboards are
// allowed when embed_credentials is false and rejected otherwise.
var allowList = []string{
	"alerts",
	"catalogs",
	"clusters",
	"cluster_policies",
	"dashboards",
	"database_catalogs",
	"database_instances",
	"external_locations",
	"synced_database_tables",
	"jobs",
	"pipelines",
	"models",
	"model_services",
	"mcp_services",
	"model_provider_services",
	"postgres_branches",
	"postgres_catalogs",
	"postgres_databases",
	"postgres_endpoints",
	"postgres_projects",
	"postgres_roles",
	"postgres_snapshot_schedules",
	"postgres_synced_tables",
	"registered_models",
	"experiments",
	"genie_spaces",
	"instance_pools",
	"job_runs",
	"internal_immutable_snapshots",
	"schemas",
	"secret_scopes",
	"secrets",
	"sql_warehouses",
	"vector_search_endpoints",
	"vector_search_indexes",
	"volumes",
}

type groupRunAsSupport int

const (
	groupRunAsUnsupported groupRunAsSupport = iota
	groupRunAsSupported
)

// Classify every resource type that exposes a run_as field.
var groupRunAsSupportByResource = map[string]groupRunAsSupport{
	"alerts":    groupRunAsUnsupported,
	"jobs":      groupRunAsSupported,
	"pipelines": groupRunAsSupported,
}

func TestRunAsGroupSupportClassification(t *testing.T) {
	actual := make(map[string]groupRunAsSupport)
	resourceTypes := reflect.TypeFor[config.Resources]()
	for field := range resourceTypes.Fields() {
		require.Equal(t, reflect.Map, field.Type.Kind(), field.Name)
		require.Equal(t, reflect.Pointer, field.Type.Elem().Kind(), field.Name)
		resourceType := field.Type.Elem().Elem()
		runAsField, ok := resourceType.FieldByName("RunAs")
		if !ok {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		require.Equal(t, reflect.Pointer, runAsField.Type.Kind(), name)
		runAsType := runAsField.Type.Elem()
		require.Equal(t, reflect.Struct, runAsType.Kind(), name)
		support := groupRunAsUnsupported
		if _, ok := runAsType.FieldByName("GroupName"); ok {
			support = groupRunAsSupported
		}
		actual[name] = support
	}

	require.Equal(t, groupRunAsSupportByResource, actual)
}

func TestRunAsGroupSupportBehavior(t *testing.T) {
	for resourceType, support := range groupRunAsSupportByResource {
		t.Run(resourceType, func(t *testing.T) {
			yaml := fmt.Sprintf("run_as: {group_name: group}\nworkspace: {current_user: {userName: deployer}}\nresources:\n  %s: {test: {}}\n", resourceType)
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			b := &bundle.Bundle{Config: *r}
			diags = bundle.Apply(t.Context(), b, SetRunAs())

			switch support {
			case groupRunAsUnsupported:
				require.ErrorContains(t, diags.Error(), "run_as.group_name")
			case groupRunAsSupported:
				require.NoError(t, diags.Error())
				switch resourceType {
				case "jobs":
					assert.Equal(t, "group", b.Config.Resources.Jobs["test"].RunAs.GroupName)
				case "pipelines":
					assert.Equal(t, "group", b.Config.Resources.Pipelines["test"].RunAs.GroupName)
				default:
					t.Fatalf("add a group propagation assertion for %s", resourceType)
				}
			default:
				t.Fatalf("unknown group run_as support level for %s", resourceType)
			}
		})
	}
}

func TestRunAsErrorForUnsupportedResources(t *testing.T) {
	base := config.Root{
		Workspace: config.Workspace{
			CurrentUser: &config.User{
				User: &iam.User{
					UserName: "alice",
				},
			},
		},
		RunAs: &jobs.JobRunAs{
			UserName: "bob",
		},
	}

	for _, rt := range allResourceTypes(t) {
		// Skip allowed resources
		if slices.Contains(allowList, rt) {
			continue
		}

		// Add an instance of the resource type that is not on the allow list to
		// the bundle configuration.
		r, diags := config.LoadFromBytes("databricks.yml", fmt.Appendf(nil, "resources:\n  %s:\n    foo:\n      path: bar\n", rt))
		require.NoError(t, diags.Error())
		r.Workspace = base.Workspace
		r.RunAs = base.RunAs

		// Assert this invalid bundle configuration fails validation.
		b := &bundle.Bundle{
			Config: *r,
		}
		diags = bundle.Apply(t.Context(), b, SetRunAs())
		require.Error(t, diags.Error())
		assert.Contains(t, diags.Error().Error(), "do not support a setting a run_as user that is different from the owner.\n"+
			"Current identity: alice. Run as identity: bob.\n"+
			"See https://docs.databricks.com/dev-tools/bundles/run-as.html to learn more about the run_as property.", rt)
	}
}

func TestRunAsNoErrorForSupportedResources(t *testing.T) {
	base := config.Root{
		Workspace: config.Workspace{
			CurrentUser: &config.User{
				User: &iam.User{
					UserName: "alice",
				},
			},
		},
		RunAs: &jobs.JobRunAs{
			UserName: "bob",
		},
	}

	for _, rt := range allResourceTypes(t) {
		// Skip unsupported resources
		if !slices.Contains(allowList, rt) {
			continue
		}

		// Add an instance of the resource type that is not on the allow list to
		// the bundle configuration.
		r, diags := config.LoadFromBytes("databricks.yml", fmt.Appendf(nil, "resources:\n  %s:\n    foo:\n      name: bar\n", rt))
		require.NoError(t, diags.Error())
		r.Workspace = base.Workspace
		r.RunAs = base.RunAs

		// Assert this configuration passes validation.
		b := &bundle.Bundle{
			Config: *r,
		}
		diags = bundle.Apply(t.Context(), b, SetRunAs())
		require.NoError(t, diags.Error())
	}
}

func TestRunAsIdentities(t *testing.T) {
	for _, tc := range []struct {
		runAs     string
		wantError bool
	}{
		// A null run_as is the same as not specifying it.
		{`null`, false},
		{`{}`, true},
		{`{user_name: ""}`, true},
		{`{service_principal_name: ""}`, true},
		{`{group_name: ""}`, true},
		{`{user_name: user}`, false},
		{`{service_principal_name: sp}`, false},
		{`{group_name: group}`, false},
		{`{user_name: "", service_principal_name: "", group_name: ""}`, true},
		{`{user_name: "", service_principal_name: "", group_name: group}`, false},
		{`{user_name: user, service_principal_name: sp}`, true},
		{`{user_name: user, group_name: group}`, true},
		{`{service_principal_name: sp, group_name: group}`, true},
		{`{user_name: user, service_principal_name: sp, group_name: group}`, true},
	} {
		t.Run(tc.runAs, func(t *testing.T) {
			yaml := "workspace: {current_user: {userName: deployer}}\nrun_as: " + tc.runAs
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			b := &bundle.Bundle{Config: *r}
			diags = bundle.Apply(t.Context(), b, SetRunAs())
			if tc.wantError {
				require.ErrorContains(t, diags.Error(), "run_as section must specify exactly one non-empty identity: user_name, service_principal_name, or group_name")
				assert.Equal(t, []diag.Location{r.GetLocation("run_as")}, diags[0].Locations)
			} else {
				require.NoError(t, diags.Error())
			}
		})
	}
}

func TestRunAsLegacyGroup(t *testing.T) {
	for _, runAs := range []string{`{group_name: group}`, `{group_name: ""}`} {
		t.Run(runAs, func(t *testing.T) {
			yaml := "workspace: {current_user: {userName: deployer}}\nexperimental: {use_legacy_run_as: true}\nrun_as: " + runAs
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			b := &bundle.Bundle{Config: *r}
			diags = bundle.Apply(t.Context(), b, SetRunAs())
			require.ErrorContains(t, diags.Error(), "run_as.group_name is not supported with experimental.use_legacy_run_as")
		})
	}
}

func TestRunAsGroupResources(t *testing.T) {
	for _, tc := range []struct {
		name      string
		resource  string
		wantError string
	}{
		{name: "alert", resource: `alerts: {test: {}}`, wantError: "alerts do not support run_as.group_name"},
		{name: "model serving", resource: `model_serving_endpoints: {test: {}}`, wantError: "Run as identity: group \"group\""},
		{name: "alert sp override", resource: `alerts: {test: {run_as: {service_principal_name: sp}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := "run_as: {group_name: group}\nworkspace: {current_user: {userName: group}}\nresources:\n  " + tc.resource
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			b := &bundle.Bundle{Config: *r}
			before := b.Config.View().Get("resources").AsAny()
			diags = bundle.Apply(t.Context(), b, SetRunAs())
			if tc.wantError != "" {
				require.Error(t, diags.Error())
				assert.Contains(t, diags.Error().Error(), tc.wantError)
			} else {
				require.NoError(t, diags.Error())
				assert.Equal(t, before, b.Config.View().Get("resources").AsAny())
			}
		})
	}
}

func TestRunAsGroupInheritance(t *testing.T) {
	for _, tc := range []struct {
		name   string
		root   string
		target string
		want   jobs.JobRunAs
	}{
		{name: "root group", root: `{group_name: group}`, target: `{}`, want: jobs.JobRunAs{GroupName: "group"}},
		{name: "target group replaces user", root: `{user_name: user}`, target: `{run_as: {group_name: group}}`, want: jobs.JobRunAs{GroupName: "group"}},
		{name: "target user replaces group", root: `{group_name: group}`, target: `{run_as: {user_name: user}}`, want: jobs.JobRunAs{UserName: "user"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := fmt.Sprintf(`
workspace: {current_user: {userName: deployer}}
run_as: %s
targets:
  test: %s
resources:
  jobs:
    inherited: {}
    user: {run_as: {user_name: other_user}}
    sp: {run_as: {service_principal_name: other_sp}}
    group: {run_as: {group_name: other_group}}
  pipelines:
    inherited: {}
    user: {run_as: {user_name: other_user}}
    sp: {run_as: {service_principal_name: other_sp}}
    group: {run_as: {group_name: other_group}}
`, tc.root, tc.target)
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			require.NoError(t, r.MergeTargetOverrides("test"))
			b := &bundle.Bundle{Config: *r}
			diags = bundle.Apply(t.Context(), b, SetRunAs())
			require.NoError(t, diags.Error())
			assert.Equal(t, &tc.want, b.Config.RunAs)
			assert.Equal(t, &tc.want, b.Config.Resources.Jobs["inherited"].RunAs)
			assert.Equal(t, &jobs.JobRunAs{UserName: "other_user"}, b.Config.Resources.Jobs["user"].RunAs)
			assert.Equal(t, &jobs.JobRunAs{ServicePrincipalName: "other_sp"}, b.Config.Resources.Jobs["sp"].RunAs)
			assert.Equal(t, &jobs.JobRunAs{GroupName: "other_group"}, b.Config.Resources.Jobs["group"].RunAs)
			assert.Equal(t, &pipelines.RunAs{
				GroupName:            tc.want.GroupName,
				ServicePrincipalName: tc.want.ServicePrincipalName,
				UserName:             tc.want.UserName,
			}, b.Config.Resources.Pipelines["inherited"].RunAs)
			assert.Equal(t, &pipelines.RunAs{UserName: "other_user"}, b.Config.Resources.Pipelines["user"].RunAs)
			assert.Equal(t, &pipelines.RunAs{ServicePrincipalName: "other_sp"}, b.Config.Resources.Pipelines["sp"].RunAs)
			assert.Equal(t, &pipelines.RunAs{GroupName: "other_group"}, b.Config.Resources.Pipelines["group"].RunAs)
		})
	}
}
