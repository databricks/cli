package resourcemutator

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rejectsAllRunAs reports whether every resource of the declaring type is
// rejected when the run_as identity differs from the deploying user.
func rejectsAllRunAs(declarer resources.RunAsDeclarer) bool {
	_, conditional := declarer.(resources.RunAsConditionalRejecter)
	return declarer.RunAsSupport() == resources.RunAsRejected && !conditional
}

func TestRunAsSupportIsDeclaredForEveryResourceType(t *testing.T) {
	config.AssertResourceTypesImplement[resources.RunAsDeclarer](t)
}

func TestRunAsDeclarationsAreConsistent(t *testing.T) {
	for key, declarer := range config.ResourceTypesImplementing[resources.RunAsDeclarer]() {
		support := declarer.RunAsSupport()
		assert.Contains(t, []resources.RunAsSupport{resources.RunAsNotApplicable, resources.RunAsApplied, resources.RunAsRejected}, support, key)

		_, receiver := declarer.(resources.RunAsReceiver)
		assert.Equal(t, support == resources.RunAsApplied, receiver, "%s must implement RunAsReceiver if and only if it declares RunAsApplied", key)

		if _, ok := declarer.(resources.RunAsValidator); ok {
			assert.True(t, receiver, "%s implements RunAsValidator but not RunAsReceiver", key)
		}
		if _, ok := declarer.(resources.RunAsLegacyReceiver); ok {
			assert.True(t, receiver, "%s implements RunAsLegacyReceiver but not RunAsReceiver", key)
		}
		if _, ok := declarer.(resources.RunAsConditionalRejecter); ok {
			assert.Equal(t, resources.RunAsRejected, support, "%s implements RunAsConditionalRejecter but does not declare RunAsRejected", key)
		}
	}
}

// runAsField returns the RunAs field of the resource type under key.
func runAsField(t *testing.T, key string) (reflect.StructField, bool) {
	for _, rt := range config.AllResourceTypes() {
		if rt.Key == key {
			return rt.Type.FieldByName("RunAs")
		}
	}
	require.Failf(t, "not a resource type", "%s", key)
	return reflect.StructField{}, false
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

// A resource type with a run_as field must receive the bundle run_as identity.
func TestRunAsAppliedToEveryTypeWithRunAsField(t *testing.T) {
	for key, declarer := range config.ResourceTypesImplementing[resources.RunAsDeclarer]() {
		_, hasField := runAsField(t, key)
		assert.Equal(t, hasField, declarer.RunAsSupport() == resources.RunAsApplied, "%s has a RunAs field: %v", key, hasField)
	}
}

// A resource type whose run_as has a group_name receives a group identity; any
// other type that receives run_as rejects a group identity.
func TestRunAsGroupSupportBehavior(t *testing.T) {
	for key := range config.ResourceTypesImplementing[resources.RunAsReceiver]() {
		t.Run(key, func(t *testing.T) {
			field, ok := runAsField(t, key)
			require.True(t, ok)
			require.Equal(t, reflect.Pointer, field.Type.Kind())
			require.Equal(t, reflect.Struct, field.Type.Elem().Kind())
			_, supportsGroup := field.Type.Elem().FieldByName("GroupName")

			yaml := fmt.Sprintf("run_as: {group_name: group}\nworkspace: {current_user: {userName: deployer}}\nresources:\n  %s: {test: {}}\n", key)
			r, diags := config.LoadFromBytes("databricks.yml", []byte(yaml))
			require.NoError(t, diags.Error())
			b := &bundle.Bundle{Config: *r}
			diags = bundle.Apply(t.Context(), b, SetRunAs())

			if !supportsGroup {
				require.ErrorContains(t, diags.Error(), "run_as.group_name")
				return
			}
			require.NoError(t, diags.Error())
			res, _, err := b.Config.Resources.ResolveKey(key + ".test")
			require.NoError(t, err)
			runAs := reflect.ValueOf(res).Elem().FieldByName("RunAs")
			require.False(t, runAs.IsNil())
			assert.Equal(t, "group", runAs.Elem().FieldByName("GroupName").String())
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

	v, err := convert.FromTyped(base, dyn.NilValue)
	require.NoError(t, err)

	// Define top level resources key in the bundle configuration.
	// This is not part of the typed configuration, so we need to add it manually.
	v, err = dyn.Set(v, "resources", dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)

	for rt, declarer := range config.ResourceTypesImplementing[resources.RunAsDeclarer]() {
		if !rejectsAllRunAs(declarer) {
			continue
		}

		// Add an instance of the rejected resource type to the bundle configuration.
		nv, err := dyn.SetByPath(v, dyn.NewPath(dyn.Key("resources"), dyn.Key(rt)), dyn.V(map[string]dyn.Value{
			"foo": dyn.V(map[string]dyn.Value{
				"path": dyn.V("bar"),
			}),
		}))
		require.NoError(t, err)

		// Get back typed configuration from the newly created invalid bundle configuration.
		r := &config.Root{}
		err = convert.ToTyped(r, nv)
		require.NoError(t, err)

		// Assert this invalid bundle configuration fails validation.
		b := &bundle.Bundle{
			Config: *r,
		}
		diags := bundle.Apply(t.Context(), b, SetRunAs())
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

	v, err := convert.FromTyped(base, dyn.NilValue)
	require.NoError(t, err)

	// Define top level resources key in the bundle configuration.
	// This is not part of the typed configuration, so we need to add it manually.
	v, err = dyn.Set(v, "resources", dyn.V(map[string]dyn.Value{}))
	require.NoError(t, err)

	for rt, declarer := range config.ResourceTypesImplementing[resources.RunAsDeclarer]() {
		// Resource types whose rejection is conditional are not rejected for
		// this configuration.
		if rejectsAllRunAs(declarer) {
			continue
		}

		// Add an instance of the resource type to the bundle configuration.
		nv, err := dyn.SetByPath(v, dyn.NewPath(dyn.Key("resources"), dyn.Key(rt)), dyn.V(map[string]dyn.Value{
			"foo": dyn.V(map[string]dyn.Value{
				"name": dyn.V("bar"),
			}),
		}))
		require.NoError(t, err)

		// Get back typed configuration from the newly created invalid bundle configuration.
		r := &config.Root{}
		err = convert.ToTyped(r, nv)
		require.NoError(t, err)

		// Assert this configuration passes validation.
		b := &bundle.Bundle{
			Config: *r,
		}
		diags := bundle.Apply(t.Context(), b, SetRunAs())
		require.NoError(t, diags.Error())
	}
}

func TestRunAsIdentities(t *testing.T) {
	for _, tc := range []struct {
		runAs     string
		wantError bool
	}{
		{`null`, true},
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
			before := b.Config.Value().Get("resources")
			diags = bundle.Apply(t.Context(), b, SetRunAs())
			if tc.wantError != "" {
				require.Error(t, diags.Error())
				assert.Contains(t, diags.Error().Error(), tc.wantError)
			} else {
				require.NoError(t, diags.Error())
				assert.Equal(t, before, b.Config.Value().Get("resources"))
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
