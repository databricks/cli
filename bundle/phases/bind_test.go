package phases

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
)

func TestConfirmBindPlan(t *testing.T) {
	for _, tc := range []struct {
		name        string
		action      deployplan.ActionType
		child       deployplan.ActionType
		immediate   bool
		autoApprove bool
		want        bool
	}{
		{"unchanged", deployplan.Bind, deployplan.Skip, true, false, true},
		{"update requires approval", deployplan.BindAndUpdate, deployplan.Skip, true, false, false},
		{"permissions require approval", deployplan.Bind, deployplan.Create, true, false, false},
		{"approved permissions", deployplan.Bind, deployplan.Create, true, true, true},
		{"deferred permissions", deployplan.Skip, deployplan.Create, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			ctx := logdiag.InitContext(t.Context())
			ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, strings.NewReader(""), io.Discard, stderr, "", ""))
			plan := deployplan.NewPlanDirect()
			plan.Plan = map[string]*deployplan.PlanEntry{
				"resources.jobs.foo":             {Action: tc.action},
				"resources.jobs.foo.permissions": {Action: tc.child},
			}
			assert.Equal(t, tc.want, confirmBindPlan(ctx, "resources.jobs.foo", plan, tc.autoApprove, tc.immediate))
			assert.Equal(t, !tc.want, logdiag.HasError(ctx))
			if !tc.want {
				assert.Contains(t, stderr.String(), "requires user confirmation")
			}
		})
	}
}
