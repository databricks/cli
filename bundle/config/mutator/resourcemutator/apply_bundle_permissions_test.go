package resourcemutator

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This list exists to ensure that this mutator is updated when new resource is added.
// These resources are there because they use grants, not permissions:
var unsupportedResources = []string{
	"catalogs",
	"external_locations",
	"volumes",
	"schemas",
	"features",
	"quality_monitors",
	"registered_models",
	"model_services",
	"mcp_services",
	"model_provider_services",
	"database_catalogs",
	"synced_database_tables",
	"postgres_branches",
	"postgres_databases",
	"postgres_endpoints",
	"postgres_catalogs",
	"postgres_roles",
	"postgres_snapshot_schedules",
	"postgres_synced_tables",
	"vector_search_indexes",
	"job_runs",
	"internal_immutable_snapshots",
	"secrets",
	"cluster_policies",
}

func TestApplyBundlePermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.test",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
				{Level: permissions.CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {},
					"pipeline_2": {},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {},
					"experiment_2": {},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {},
					"endpoint_2": {},
				},
				Dashboards: map[string]*resources.Dashboard{
					"dashboard_1": {},
					"dashboard_2": {},
				},
				Apps: map[string]*resources.App{
					"app_1": {},
					"app_2": {},
				},
				VectorSearchEndpoints: map[string]*resources.VectorSearchEndpoint{
					"vs_1": {},
					"vs_2": {},
				},
				InstancePools: map[string]*resources.InstancePool{
					"instance_pool_1": {},
					"instance_pool_2": {},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Len(t, b.Config.Resources.Jobs["job_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Jobs["job_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Models["model_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Models["model_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Apps["app_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_USE", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.VectorSearchEndpoints["vs_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.VectorSearchEndpoints["vs_1"].Permissions, resources.VectorSearchEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.VectorSearchEndpoints["vs_1"].Permissions, resources.VectorSearchEndpointPermission{Level: "CAN_USE", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.VectorSearchEndpoints["vs_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.VectorSearchEndpoints["vs_2"].Permissions, resources.VectorSearchEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.VectorSearchEndpoints["vs_2"].Permissions, resources.VectorSearchEndpointPermission{Level: "CAN_USE", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.InstancePools["instance_pool_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.InstancePools["instance_pool_1"].Permissions, resources.InstancePoolPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.InstancePools["instance_pool_1"].Permissions, resources.InstancePoolPermission{Level: "CAN_ATTACH_TO", GroupName: "TestGroup"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.test",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser"},
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser2"},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Len(t, diags, 1)
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "'jobs' already has permissions set for 'TestUser' user name", diags[0].Summary)

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
}

func TestTargetPermissionsOverrideWithoutWarning(t *testing.T) {
	for resourceType, lm := range levelsMap {
		for _, resourceLayer := range []string{"root", "target"} {
			for _, withRootPermissions := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/root_permissions=%t", resourceType, resourceLayer, withRootPermissions), func(t *testing.T) {
					resourceConfig := fmt.Sprintf(`
resources:
  %s:
    test:
      permissions:
        - {group_name: principal, level: %s}
        - {user_name: principal, level: %s}
        - {service_principal_name: principal, level: %s}
`, resourceType, lm[permissions.CAN_MANAGE], lm[permissions.CAN_MANAGE], lm[permissions.CAN_MANAGE])
					rootConfig := ""
					if withRootPermissions {
						rootConfig = "permissions:\n  - {group_name: principal, level: CAN_VIEW}\n"
					}
					targetConfig := `permissions:
  - {group_name: principal, level: CAN_VIEW}
  - {user_name: principal, level: CAN_VIEW}
  - {service_principal_name: principal, level: CAN_VIEW}
  - {group_name: inherited, level: CAN_VIEW}
`
					if resourceLayer == "root" {
						rootConfig += resourceConfig
					} else {
						targetConfig += resourceConfig
					}
					yaml := rootConfig + "\ntargets:\n  test:\n" + "    " + strings.ReplaceAll(strings.TrimSpace(targetConfig), "\n", "\n    ") + "\n"
					root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
					require.Empty(t, diags)
					b := &bundle.Bundle{Config: *root}
					diags = bundle.ApplySeq(t.Context(), b, mutator.SelectTarget("test"), ApplyBundlePermissions())
					if withRootPermissions {
						require.Equal(t, diag.Warningf("'%s' already has permissions set for 'principal' group", resourceType), diags)
					} else {
						require.Empty(t, diags)
					}
					actual := b.Config.Value().Get("resources").Get(resourceType).Get("test").Get("permissions").AsAny()
					assert.Equal(t, []any{
						map[string]any{"group_name": "principal", "level": lm[permissions.CAN_MANAGE]},
						map[string]any{"user_name": "principal", "level": lm[permissions.CAN_MANAGE]},
						map[string]any{"service_principal_name": "principal", "level": lm[permissions.CAN_MANAGE]},
						map[string]any{"group_name": "inherited", "level": lm[permissions.CAN_VIEW]},
					}, actual)
				})
			}
		}
	}
}

func TestTargetPermissionsVariableResolution(t *testing.T) {
	for _, tc := range []struct {
		name   string
		root   string
		target string
		warn   bool
	}{
		{name: "target literal", target: `[{group_name: "${var.group}", level: CAN_VIEW}]`},
		{name: "target reference", target: "${var.acl}"},
		{name: "null root", root: "null", target: "${var.acl}"},
		{name: "resource narrows target", target: `[{group_name: developers, level: CAN_MANAGE}]`},
		{name: "target entry reference", target: "[\"${var.acl_entry}\"]"},
		{name: "root reference", root: "${var.acl}", warn: true},
		{name: "root literal", root: `[{group_name: "${var.group}", level: CAN_VIEW}]`, warn: true},
		{name: "both literals", root: `[{group_name: "${var.group}", level: CAN_VIEW}]`, target: `[{group_name: "${var.group}", level: CAN_VIEW}]`, warn: true},
		{name: "reference overrides reference", root: "${var.unused_acl}", target: "${var.acl}"},
		{name: "null target", root: "${var.acl}", target: "null", warn: true},
		{name: "empty target", root: `[{group_name: "${var.group}", level: CAN_VIEW}]`, target: "[]", warn: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			yaml := `
variables:
  group:
    default: developers
  acl:
    type: complex
    default:
      - {group_name: "${var.group}", level: CAN_VIEW}
  acl_entry:
    type: complex
    default: {group_name: "${var.group}", level: CAN_VIEW}
  unused_acl:
    type: complex
    default:
      - {group_name: overridden, level: CAN_MANAGE}
resources:
  jobs:
    test:
      permissions:
        - {group_name: developers, level: CAN_MANAGE_RUN}
`
			if tc.root != "" {
				yaml += "permissions: " + tc.root + "\n"
			}
			yaml += "targets:\n  test:\n    default: true\n"
			if tc.target != "" {
				yaml += "    permissions: " + tc.target + "\n"
			}
			root, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.Empty(t, diags)
			b := &bundle.Bundle{Config: *root}
			diags = bundle.ApplySeq(t.Context(), b,
				mutator.SelectTarget("test"), mutator.SetVariables(),
				mutator.ResolveVariableReferencesWithoutResources(), ApplyBundlePermissions(),
			)
			if tc.warn {
				require.Equal(t, diag.Warningf("'jobs' already has permissions set for 'developers' group"), diags)
			} else {
				require.Empty(t, diags)
			}
			assert.Equal(t, []resources.JobPermission{{GroupName: "developers", Level: "CAN_MANAGE_RUN"}}, b.Config.Resources.Jobs["test"].Permissions)
		})
	}
}

func TestAllResourcesExplicitlyDefinedForPermissionsSupport(t *testing.T) {
	r := config.Resources{}

	for _, resource := range unsupportedResources {
		_, ok := levelsMap[resource]
		assert.False(t, ok, "Resource %s is defined in both levelsMap and unsupportedResources", resource)
	}

	for _, resource := range r.AllResources() {
		_, ok := levelsMap[resource.Description.PluralName]
		if !slices.Contains(unsupportedResources, resource.Description.PluralName) && !ok {
			assert.Fail(t, fmt.Sprintf("Resource %s is not explicitly defined in levelsMap or unsupportedResources", resource.Description.PluralName))
		}
	}
}

func TestApplyBundlePermissionsEmpty(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"empty": {
						JobSettings: jobs.JobSettings{Name: "empty"},
						Permissions: []resources.JobPermission{},
					},
					"unset": {
						JobSettings: jobs.JobSettings{Name: "unset"},
					},
				},
			},
		},
	}

	diags := bundle.Apply(t.Context(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	// An empty list of permissions is dropped, an unset one stays unset.
	for _, name := range []string{"empty", "unset"} {
		require.Nil(t, b.Config.Resources.Jobs[name].Permissions)
		v, err := dyn.GetByPath(b.Config.Value(), dyn.MustPathFromString("resources.jobs."+name))
		require.NoError(t, err)
		assert.False(t, v.Get("permissions").IsValid(), name)
	}
}

func TestDuplicatePermissionsInOneSection(t *testing.T) {
	for _, section := range []string{"root", "target", "resource"} {
		t.Run(section, func(t *testing.T) {
			b := &bundle.Bundle{Config: config.Root{
				Targets: map[string]*config.Target{"test": {}},
				Workspace: config.Workspace{CurrentUser: &config.User{
					User: &iam.User{UserName: "deployer@example.test"},
				}},
				Resources: config.Resources{Jobs: map[string]*resources.Job{"test": {}}},
			}}
			permissions := []resources.Permission{
				{UserName: "reader@example.test", Level: "CAN_VIEW"},
				{UserName: "reader@example.test", Level: "CAN_MANAGE"},
			}
			switch section {
			case "root":
				b.Config.Permissions = permissions
			case "target":
				b.Config.Targets["test"].Permissions = permissions
			case "resource":
				b.Config.Resources.Jobs["test"].Permissions = []resources.JobPermission{
					{UserName: "reader@example.test", Level: "CAN_VIEW"},
					{UserName: "reader@example.test", Level: "CAN_MANAGE"},
				}
			}

			diags := bundle.ApplySeq(t.Context(), b,
				mutator.SelectTarget("test"), ApplyBundlePermissions(), FixPermissions())
			require.Empty(t, diags)
			assert.Equal(t, []resources.JobPermission{
				{UserName: "reader@example.test", Level: "CAN_MANAGE"},
				{UserName: "deployer@example.test", Level: "IS_OWNER"},
			}, b.Config.Resources.Jobs["test"].Permissions)
		})
	}
}
