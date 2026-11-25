package deploy

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type namedCheck struct {
	name   string
	isPlan bool
}

func (m *namedCheck) Name() string {
	return m.name
}

func (m *namedCheck) Apply(context.Context, *bundle.Bundle) diag.Diagnostics {
	return nil
}

func newNamedCheck(name string) func(isPlan bool) bundle.Mutator {
	return func(isPlan bool) bundle.Mutator {
		return &namedCheck{name: name, isPlan: isPlan}
	}
}

func TestOrderPreDeployChecksUsesTypeOrder(t *testing.T) {
	checks := orderPreDeployChecks([]preDeployCheckRegistration{
		{resourceKey: "pipelines", newCheck: newNamedCheck("pipelines")},
		{resourceKey: "jobs", newCheck: newNamedCheck("jobs")},
	}, []string{"jobs", "pipelines"})

	var names []string
	for _, newCheck := range checks {
		check := newCheck(true).(*namedCheck)
		assert.True(t, check.isPlan)
		names = append(names, check.Name())
	}
	assert.Equal(t, []string{"jobs", "pipelines"}, names)
}

func TestOrderPreDeployChecksRejectsUnknownType(t *testing.T) {
	assert.PanicsWithValue(t, `pre-deploy check registered for unknown resource type "nope"`, func() {
		orderPreDeployChecks([]preDeployCheckRegistration{{resourceKey: "nope", newCheck: newNamedCheck("a")}}, []string{"jobs"})
	})
}

func TestOrderPreDeployChecksRejectsDuplicate(t *testing.T) {
	assert.PanicsWithValue(t, `pre-deploy check registered more than once for resource type "jobs"`, func() {
		orderPreDeployChecks([]preDeployCheckRegistration{
			{resourceKey: "jobs", newCheck: newNamedCheck("a")},
			{resourceKey: "jobs", newCheck: newNamedCheck("b")},
		}, []string{"jobs"})
	})
}

func TestPreDeployResourceChecksRunsRegisteredChecks(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(&dashboards.Dashboard{
			DisplayName: "My Special Dashboard",
			Etag:        "1234",
		}, nil).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(ctx, b, PreDeployResourceChecks(true))
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `dashboard "dash1" has been modified remotely`, diags[0].Summary)
	}
}
