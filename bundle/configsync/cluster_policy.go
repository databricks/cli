package configsync

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

type clusterPolicyDefinitions map[string]dyn.Value

// get reads the effective definition, including policy-family overrides, once per sync.
func (definitions clusterPolicyDefinitions) get(ctx context.Context, b *bundle.Bundle, id string) (dyn.Value, error) {
	if definition, ok := definitions[id]; ok {
		return definition, nil
	}
	policy, err := b.WorkspaceClient(ctx).ClusterPolicies.GetByPolicyId(ctx, id)
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("reading cluster policy %s: %w", id, err)
	}
	definition, err := jsonloader.LoadJSON([]byte(policy.Definition), "cluster policy "+id)
	if err != nil {
		return dyn.InvalidValue, err
	}
	definitions[id] = definition
	return definition, nil
}

type clusterPolicyFilter struct {
	definitions clusterPolicyDefinitions
	resourceKey string
	config      any
	remote      any
	standalone  bool
}

func newClusterPolicyFilter(resourceKey string, entry *deployplan.PlanEntry, definitions clusterPolicyDefinitions) *clusterPolicyFilter {
	if !strings.HasPrefix(resourceKey, "resources.jobs.") && !strings.HasPrefix(resourceKey, "resources.clusters.") {
		return nil
	}
	if entry.RemoteState == nil {
		return nil
	}
	return &clusterPolicyFilter{
		definitions: definitions,
		resourceKey: resourceKey,
		remote:      entry.RemoteState,
		standalone:  strings.HasPrefix(resourceKey, "resources.clusters."),
	}
}

// filter removes only omitted values that match the attached policy. It also handles
// whole-object additions, where the plan does not expose individual policy-applied leaves.
// A nil result means the value should be omitted from YAML.
func (f *clusterPolicyFilter) filter(ctx context.Context, b *bundle.Bundle, path *structpath.PathNode, value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	applied, err := f.isPolicyApplied(ctx, b, path, value)
	if err != nil {
		return nil, err
	}
	if applied {
		return nil, nil
	}

	switch value := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(value))
		for key, child := range value {
			filtered, err := f.filter(ctx, b, structpath.NewStringKey(path, key), child)
			if err != nil {
				return nil, err
			}
			if filtered != nil {
				result[key] = filtered
			}
		}
		if len(value) > 0 && len(result) == 0 {
			return nil, nil
		}
		_, attribute, _ := clusterPolicyAttribute(path, f.standalone)
		if (attribute == "cluster_log_conf" || strings.HasPrefix(attribute, "cluster_log_conf.")) && !reflect.DeepEqual(value, result) {
			// Log destinations are required. Removing only policy-matched leaves
			// can leave an invalid object, such as an S3 region without a destination.
			log.Warnf(ctx, "Keeping policy-applied values in %s to preserve the complete log configuration; review this field before deploying with another policy", path)
			return value, nil
		}
		return result, nil
	case []any:
		result := make([]any, 0, len(value))
		for i, child := range value {
			childPath := structpath.NewIndex(path, i)
			// Match cluster owners by identity even when the UI reorders tasks.
			if object, ok := child.(map[string]any); ok {
				field, _ := path.StringKey()
				var key string
				switch field {
				case "tasks":
					key = "task_key"
				case "job_clusters":
					key = "job_cluster_key"
				}
				if name, ok := object[key].(string); ok && key != "" {
					childPath = structpath.NewKeyValue(path, key, name)
				}
			}
			filtered, err := f.filter(ctx, b, childPath, child)
			if err != nil {
				return nil, err
			}
			if filtered != nil {
				result = append(result, filtered)
			}
		}
		if len(value) > 0 && len(result) == 0 {
			return nil, nil
		}
		// Indexed policy rules depend on positions. A partial removal would
		// move a user-authored element into a different policy constraint.
		if len(result) != len(value) {
			log.Warnf(ctx, "Keeping policy-applied values in %s to preserve array positions; review this field before deploying with another policy", path)
			return value, nil
		}
		return result, nil
	default:
		return value, nil
	}
}

func (f *clusterPolicyFilter) isPolicyApplied(ctx context.Context, b *bundle.Bundle, path *structpath.PathNode, value any) (bool, error) {
	clusterPath, attribute, ok := clusterPolicyAttribute(path, f.standalone)
	if !ok {
		return false, nil
	}
	remotePolicyID, _ := structaccess.Get(f.remote, structpath.NewStringKey(clusterPath, "policy_id"))
	policyID, _ := remotePolicyID.(string)
	if policyID == "" {
		return false, nil
	}
	if f.config == nil {
		// Skip-only plans omit NewState. Load bundle config only when a
		// policy-backed cluster field needs comparison, not for unrelated edits.
		config, err := b.Config.GetResourceConfig(f.resourceKey)
		if err != nil {
			return false, fmt.Errorf("reading config for %s: %w", f.resourceKey, err)
		}
		if config == nil {
			// A deleted resource has no source configuration to filter against.
			return false, nil
		}
		f.config = config
	}
	// Missing fields and newly added keyed elements have no authored value.
	if configValue, _ := structaccess.Get(f.config, path); configValue != nil {
		return false, nil
	}
	definition, err := f.definitions.get(ctx, b, policyID)
	if err != nil {
		return false, err
	}
	// Jobs GET omits this input-only flag; use the authored setting.
	applyDefaults, _ := structaccess.Get(f.config, structpath.NewStringKey(clusterPath, "apply_policy_default_values"))
	return isPolicyAppliedValue(definition, attribute, value, applyDefaults == true), nil
}

// clusterPolicyAttribute returns the enclosing cluster path and policy attribute name,
// preserving map keys such as spark_conf["a.b"] as the attribute spark_conf.a.b.
func clusterPolicyAttribute(path *structpath.PathNode, standalone bool) (*structpath.PathNode, string, bool) {
	var clusterPath *structpath.PathNode
	var parts []string
	insideCluster := standalone
	for _, node := range path.AsSlice() {
		if key, ok := node.StringKey(); ok {
			if !insideCluster && key == "new_cluster" {
				clusterPath = node
				insideCluster = true
				continue
			}
			if insideCluster {
				parts = append(parts, key)
			}
		} else if index, ok := node.Index(); ok && insideCluster {
			parts = append(parts, strconv.Itoa(index))
		}
	}
	return clusterPath, strings.Join(parts, "."), insideCluster && len(parts) > 0
}

func isPolicyAppliedValue(definition dyn.Value, attribute string, remoteValue any, applyDefaults bool) bool {
	// Policies describe log delivery with type/path, while the API nests the
	// destination under its storage type (e.g. cluster_log_conf.dbfs.destination).
	// https://docs.databricks.com/aws/en/admin/clusters/policy-definition
	if attribute == "cluster_log_conf.s3.region" {
		attribute = "cluster_log_conf.region"
	}
	if storage, ok := strings.CutPrefix(attribute, "cluster_log_conf."); ok {
		if storage, ok := strings.CutSuffix(storage, ".destination"); ok {
			return isPolicyAppliedValue(definition, "cluster_log_conf.type", strings.ToUpper(storage), applyDefaults) &&
				isPolicyAppliedValue(definition, "cluster_log_conf.path", remoteValue, applyDefaults)
		}
	}
	rule, err := dyn.GetByPath(definition, dyn.NewPath(dyn.Key(attribute)))
	if err != nil {
		return false
	}
	typeValue, _ := dyn.Get(rule, "type")
	var policyValue dyn.Value
	switch {
	case typeValue.AsAny() == "fixed":
		policyValue, _ = dyn.Get(rule, "value")
	case applyDefaults && typeValue.AsAny() != "forbidden":
		policyValue, _ = dyn.Get(rule, "defaultValue")
	default:
		return false
	}
	return policyValue.IsValid() && policyValue.AsAny() != nil && reflect.DeepEqual(policyValue.AsAny(), remoteValue)
}
