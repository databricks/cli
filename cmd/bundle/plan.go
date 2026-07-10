package bundle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	bundleconfig "github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

func newPlanCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "Show deployment plan",
		Long: `Show the deployment plan for the current bundle configuration.

This command builds the bundle and displays the actions which will be done on resources that would be deployed, without making any changes.
It is useful for previewing changes before running 'bundle deploy'.`,
		Args: root.NoArgs,
	}

	var force bool
	var clusterId string
	var selectResources []string
	cmd.Flags().BoolVar(&force, "force", false, "Force-override Git branch validation.")
	cmd.Flags().StringVar(&clusterId, "compute-id", "", "Override cluster in the deployment with the given compute ID.")
	cmd.Flags().StringVarP(&clusterId, "cluster-id", "c", "", "Override cluster in the deployment with the given cluster ID.")
	cmd.Flags().MarkDeprecated("compute-id", "use --cluster-id instead")
	cmd.Flags().StringSliceVar(&selectResources, "select", nil, "Plan only the specified resource (e.g. 'my_job' or 'jobs.my_job'). Can be repeated or comma-separated.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		opts := utils.ProcessOptions{
			AlwaysPull:      true,
			FastValidate:    true,
			Build:           true,
			PreDeployChecks: true,
			InitFunc: func(b *bundle.Bundle) {
				b.Config.Bundle.Force = force
				b.Select = selectResources

				if cmd.Flag("compute-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}

				if cmd.Flag("cluster-id").Changed {
					b.Config.Bundle.ClusterId = clusterId
				}
			},
		}

		b, stateDesc, err := utils.ProcessBundleRet(cmd, opts)
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		plan := phases.RunPlan(ctx, b, stateDesc.Engine)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Count actions by type and collect formatted actions
		createCount := 0
		updateCount := 0
		deleteCount := 0
		unchangedCount := 0

		for _, change := range plan.GetActions() {
			switch change.ActionType {
			case deployplan.Create:
				createCount++
			case deployplan.Update, deployplan.UpdateWithID, deployplan.Resize:
				updateCount++
			case deployplan.Delete:
				deleteCount++
			case deployplan.Recreate:
				// A recreate counts as both a delete and a create
				deleteCount++
				createCount++
			case deployplan.Skip, deployplan.Undefined:
				unchangedCount++
			}
		}

		out := cmd.OutOrStdout()

		switch root.OutputType(cmd) {
		case flags.OutputText:
			// Print summary line and actions to stdout
			totalChanges := createCount + updateCount + deleteCount
			if totalChanges > 0 {
				// Print all actions in the order they were processed
				for _, action := range plan.GetActions() {
					if action.ActionType == deployplan.Skip {
						continue
					}
					key := strings.TrimPrefix(action.ResourceKey, "resources.")
					fmt.Fprintf(out, "%s %s\n", action.ActionType.StringShort(), key)
				}
				fmt.Fprintln(out)
			}
			// Note, this string should not be changed, "bundle deployment migrate" depends on this format:
			fmt.Fprintf(out, "Plan: %d to add, %d to change, %d to delete, %d unchanged\n", createCount, updateCount, deleteCount, unchangedCount)
		case flags.OutputJSON:
			buf, err := marshalPlanRedacted(plan)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, string(buf))
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
			return nil
		}

		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		return nil
	}

	return cmd
}

// marshalPlanRedacted encodes plan as indented JSON with sensitive field values
// replaced by "********". It operates on a decoded copy of the plan and never
// mutates the live *deployplan.Plan used by the deployment pipeline.
func marshalPlanRedacted(plan *deployplan.Plan) ([]byte, error) {
	// Step 1: encode the plan to JSON.
	raw, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}

	// Step 2: decode into a generic map using UseNumber to preserve int64 IDs.
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}

	// Step 3: walk plan.plan entries and redact sensitive fields.
	planEntries, _ := m["plan"].(map[string]any)
	for resourceKey, entryAny := range planEntries {
		// Resource key format: "resources.<type>.<name>", e.g. "resources.secrets.my_secret".
		parts := strings.SplitN(resourceKey, ".", 3)
		if len(parts) < 2 || parts[0] != "resources" {
			continue
		}
		sensitiveNames := bundleconfig.SensitiveFieldsForResourceType(parts[1])
		if len(sensitiveNames) == 0 {
			continue
		}

		entry, ok := entryAny.(map[string]any)
		if !ok {
			continue
		}
		redactPlanEntry(entry, sensitiveNames)
	}

	// Step 4: re-encode the redacted copy.
	return json.MarshalIndent(m, "", "  ")
}

// redactPlanEntry masks sensitive field values inside a single decoded plan
// entry map in place. sensitiveNames is the set of JSON field names to mask.
func redactPlanEntry(entry map[string]any, sensitiveNames map[string]bool) {
	// Mask fields inside new_state.value (JSON object of the resource state).
	if ns, ok := entry["new_state"].(map[string]any); ok {
		if valRaw, ok := ns["value"].(json.RawMessage); ok {
			ns["value"] = redactJSONObject(valRaw, sensitiveNames)
		} else if valMap, ok := ns["value"].(map[string]any); ok {
			redactMapInPlace(valMap, sensitiveNames)
		}
	}

	// Mask fields inside changes[<fieldName>].{old, new, remote}.
	// Each change key is the field path (e.g. "value") and the payload is a
	// ChangeDesc object whose Old/New/Remote hold the raw field value.
	if changes, ok := entry["changes"].(map[string]any); ok {
		for fieldPath, changeAny := range changes {
			// fieldPath may be "value" or a nested path like "config.value".
			// Check the top-level segment only.
			topField := fieldPath
			if before, _, ok0 := strings.Cut(fieldPath, "."); ok0 {
				topField = before
			}
			if !sensitiveNames[topField] {
				continue
			}
			change, ok := changeAny.(map[string]any)
			if !ok {
				continue
			}
			for _, key := range []string{"old", "new", "remote"} {
				if s, ok := change[key].(string); ok && s != "" {
					change[key] = "********"
				}
			}
		}
	}

	// Mask fields inside remote_state (the full remote resource struct).
	// In practice sensitive fields like Secret.Value are write-only and not
	// returned by the API, so remote_state will already be empty here. We
	// mask defensively in case a future resource type differs.
	if rs, ok := entry["remote_state"].(map[string]any); ok {
		redactMapInPlace(rs, sensitiveNames)
	}
}

// redactJSONObject decodes a json.RawMessage as a map, masks sensitive fields,
// and returns the result as a map[string]any. Falls back to returning raw on
// decode error.
func redactJSONObject(raw json.RawMessage, sensitiveNames map[string]bool) any {
	var m map[string]any
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	if err := dec.Decode(&m); err != nil {
		return raw
	}
	redactMapInPlace(m, sensitiveNames)
	return m
}

// redactMapInPlace replaces non-empty string values for sensitive field names
// with "********", operating in place on a decoded JSON map.
func redactMapInPlace(m map[string]any, sensitiveNames map[string]bool) {
	for name := range sensitiveNames {
		if s, ok := m[name].(string); ok && s != "" {
			m[name] = "********"
		}
	}
}
