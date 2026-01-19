package configsync

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/libs/log"
)

// DetectChanges compares current remote state with the last deployed state
// and returns a map of resource changes.
func DetectChanges(ctx context.Context, b *bundle.Bundle) (map[string]deployplan.Changes, error) {
	changes := make(map[string]deployplan.Changes)

	deployBundle := &direct.DeploymentBundle{}
	// TODO: for Terraform engine we should read the state file, converted to direct state format, it should be created during deployment
	_, statePath := b.StateFilenameDirect(ctx)

	plan, err := deployBundle.CalculatePlan(ctx, b.WorkspaceClient(), &b.Config, statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate plan: %w", err)
	}

	for resourceKey, entry := range plan.Plan {
		resourceChanges := make(deployplan.Changes)

		if entry.Changes != nil {
			for path, changeDesc := range entry.Changes {
				if changeDesc.Remote == nil && changeDesc.Old == nil && changeDesc.New == nil {
					continue
				}

				shouldSkip := (changeDesc.Action == deployplan.Skip && changeDesc.Reason != deployplan.ReasonServerSideDefault) || shouldSkipField(path, changeDesc)
				if !shouldSkip {
					resourceChanges[path] = changeDesc
				}
			}
		}

		if len(resourceChanges) != 0 {
			changes[resourceKey] = resourceChanges
		}

		log.Debugf(ctx, "Resource %s has %d changes", resourceKey, len(resourceChanges))
	}

	return changes, nil
}

// fieldDefault represents a field with its default check function.
type fieldDefault struct {
	pattern   *regexp.Regexp
	isDefault func(*deployplan.ChangeDesc) bool
}

// serverSideDefaults contains all hardcoded server-side defaults.
// This is a temporary solution until the bundle plan issue is resolved.
var serverSideDefaults = []fieldDefault{
	// Job-level fields
	{
		pattern:   regexp.MustCompile(`^email_notifications$`),
		isDefault: isEmptyStruct,
	},
	{
		pattern:   regexp.MustCompile(`^webhook_notifications$`),
		isDefault: isEmptyStruct,
	},
	{
		pattern:   regexp.MustCompile(`^timeout_seconds$`),
		isDefault: isZero,
	},
	{
		pattern:   regexp.MustCompile(`^usage_policy_id$`),
		isDefault: alwaysDefault, // computed field
	},

	// Task-level fields (using regex to match any task_key)
	{
		pattern:   regexp.MustCompile(`^tasks\[task_key='[^']+'\]\.email_notifications$`),
		isDefault: isEmptyStruct,
	},
	{
		pattern:   regexp.MustCompile(`^tasks\[task_key='[^']+'\]\.run_if$`),
		isDefault: isStringEqual("ALL_SUCCESS"),
	},
	{
		pattern:   regexp.MustCompile(`^tasks\[task_key='[^']+'\]\.disabled$`),
		isDefault: isBoolEqual(false),
	},
	{
		pattern:   regexp.MustCompile(`^tasks\[task_key='[^']+'\]\.timeout_seconds$`),
		isDefault: isZero,
	},
	{
		pattern:   regexp.MustCompile(`^tasks\[task_key='[^']+'\]\.notebook_task\.source$`),
		isDefault: isStringEqual("WORKSPACE"),
	},
}

// shouldSkipField checks if a given field path should be skipped as a hardcoded server-side default.
func shouldSkipField(path string, changeDesc *deployplan.ChangeDesc) bool {
	// TODO: as for now in bundle plan all remote-side changes are considered as server-side defaults.
	// Once it is solved - stop skipping server-side defaults in these checks and remove hardcoded default.
	if changeDesc.Action == deployplan.Skip && changeDesc.Reason != deployplan.ReasonServerSideDefault {
		return true
	}

	for _, def := range serverSideDefaults {
		if def.pattern.MatchString(path) {
			return def.isDefault(changeDesc)
		}
	}
	return false
}

// alwaysDefault always returns true (for computed fields).
func alwaysDefault(*deployplan.ChangeDesc) bool {
	return true
}

// isEmptyStruct checks if the remote value is an empty struct or map.
func isEmptyStruct(changeDesc *deployplan.ChangeDesc) bool {
	if changeDesc.Remote == nil {
		return true
	}

	val := reflect.ValueOf(changeDesc.Remote)
	switch val.Kind() {
	case reflect.Map:
		return val.Len() == 0
	case reflect.Struct:
		return isStructAllZeroValues(val)
	case reflect.Ptr:
		if val.IsNil() {
			return true
		}
		return isEmptyStruct(&deployplan.ChangeDesc{Remote: val.Elem().Interface()})
	default:
		return false
	}
}

// isStructAllZeroValues checks if all fields in a struct are zero values.
func isStructAllZeroValues(val reflect.Value) bool {
	for i := range val.NumField() {
		field := val.Field(i)
		if !field.IsZero() {
			return false
		}
	}
	return true
}

// isStringEqual returns a function that checks if the remote value equals the given string.
func isStringEqual(expected string) func(*deployplan.ChangeDesc) bool {
	return func(changeDesc *deployplan.ChangeDesc) bool {
		if changeDesc.Remote == nil {
			return expected == ""
		}
		// Convert to string to handle SDK enum types
		actual := fmt.Sprintf("%v", changeDesc.Remote)
		return actual == expected
	}
}

// isBoolEqual returns a function that checks if the remote value equals the given bool.
func isBoolEqual(expected bool) func(*deployplan.ChangeDesc) bool {
	return func(changeDesc *deployplan.ChangeDesc) bool {
		if actual, ok := changeDesc.Remote.(bool); ok {
			return actual == expected
		}
		return false
	}
}

// isZero checks if the remote value is zero (0 or 0.0).
func isZero(changeDesc *deployplan.ChangeDesc) bool {
	if changeDesc.Remote == nil {
		return true
	}

	switch v := changeDesc.Remote.(type) {
	case int:
		return v == 0
	case int64:
		return v == 0
	case float64:
		return v == 0.0
	default:
		return false
	}
}
