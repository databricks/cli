package direct

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestIsIDReference(t *testing.T) {
	for ref, want := range map[string]bool{
		"${resources.jobs.foo.id}":                   true,
		"${resources.jobs.foo.name}":                 false,
		"${resources.jobs.foo.tasks[0].id}":          false,
		"${resources.postgres_projects.p.spec.id}":   true,
		"${resources.postgres_projects.p.id}":        true,
		"${resources.schemas.s.grants[0].principal}": false,
		"not a reference":                            false,
	} {
		assert.Equal(t, want, isIDReference(ref), ref)
	}
}

// A reference to the id of a resource planned for deletion is left to the check after
// --select, also when it is spelled the Terraform way.
func TestCheckReferenceTargetsLeavesIDReferences(t *testing.T) {
	for label, wantErr := range map[string]bool{
		"${resources.postgres_projects.p.id}":      false,
		"${resources.postgres_projects.p.spec.id}": false,
		"${resources.postgres_projects.p.name}":    true,
	} {
		plan := deployplan.NewPlanDirect()
		plan.Plan["resources.postgres_projects.p"] = &deployplan.PlanEntry{Action: deployplan.Delete}
		plan.Plan["resources.jobs.live"] = &deployplan.PlanEntry{
			DependsOn: []deployplan.DependsOnEntry{{Node: "resources.postgres_projects.p", Label: label}},
		}
		err := checkReferenceTargets(&config.Root{}, plan)
		if wantErr {
			assert.ErrorContains(t, err, label+" refers to resources.postgres_projects.p, which was removed", label)
		} else {
			assert.NoError(t, err, label)
		}
	}
}

// A Terraform-style reference has no resource node; it is left to makeGraph, which
// reports it, instead of being reported as a resource missing from the config.
func TestCheckReferenceTargetsLeavesTerraformReferences(t *testing.T) {
	plan := deployplan.NewPlanDirect()
	plan.Plan["resources.jobs.live"] = &deployplan.PlanEntry{
		DependsOn: []deployplan.DependsOnEntry{{Node: "", Label: "${databricks_pipeline.x.id}"}},
	}
	assert.NoError(t, checkReferenceTargets(&config.Root{}, plan))
}
