package terraform_dabs_map

import "github.com/databricks/cli/libs/structs/structpath"

// DABsPathToTerraform translates a field path from DABs naming conventions
// to Terraform naming conventions for the given resource group.
//
// It is the inverse of TerraformPathToDABs. For groups whose TF schema wraps fields
// under a structural prefix (e.g. "spec"), that prefix is prepended to the result.
// Each field name segment is looked up in the DABsToTerraformRenameMap: when found the TF
// name is used and the tree descends for the remainder of the path. Array indices pass
// through unchanged without advancing the tree position. An unrecognised segment stops
// further renaming; remaining segments are kept as-is. Returns nil when path is nil.
//
// The path must be relative to the resource root (e.g. "tasks", not
// "resources.jobs.my_job.tasks").
func DABsPathToTerraform(group string, path *structpath.PathNode) *structpath.PathNode {
	if path == nil {
		return nil
	}

	// For groups with a TF wrapper (Unwrap inverse), prepend it as the first segment.
	var result *structpath.PathNode
	if wrapper, ok := DABsToTerraformWrappers[group]; ok {
		result = structpath.NewDotString(nil, wrapper)
	}

	tree := DABsToTerraformRenameMap[group]
	for _, n := range path.AsSlice() {
		if key, ok := n.StringKey(); ok {
			if rn, found := tree[key]; found {
				if rn.NewName != "" {
					key = rn.NewName
				}
				tree = rn.Children
			} else {
				tree = nil
			}
			if n.IsDotString() {
				result = structpath.NewDotString(result, key)
			} else {
				result = structpath.NewBracketString(result, key)
			}
		} else if idx, ok := n.Index(); ok {
			result = structpath.NewIndex(result, idx)
		} else if k, v, ok := n.KeyValue(); ok {
			result = structpath.NewKeyValue(result, k, v)
		}
	}
	return result
}

// TerraformPathToDABs translates a field path from Terraform naming conventions
// to DABs naming conventions for the given resource group.
//
// Each field name segment is looked up in the rename tree: when found the DABs
// name is used and the tree descends into the sub-tree for the remainder of the
// path. Array indices pass through unchanged without advancing the tree position.
// An unrecognised field name stops further renaming; remaining segments are kept
// as-is. Returns nil when path is nil.
//
// The path must be relative to the resource root (e.g. "task.library", not
// "resources.jobs.my_job.task.library").
func TerraformPathToDABs(group string, path *structpath.PathNode) *structpath.PathNode {
	if path == nil {
		return nil
	}
	tree := TerraformToDABsFieldMap[group]
	var result *structpath.PathNode
	for _, n := range path.AsSlice() {
		if key, ok := n.StringKey(); ok {
			if rn, found := tree[key]; found {
				if rn.Unwrap {
					// This TF segment is a structural wrapper with no DABs equivalent; skip it.
					tree = rn.Children
					continue
				}
				if rn.NewName != "" {
					key = rn.NewName
				}
				tree = rn.Children
			} else {
				tree = nil
			}
			if n.IsDotString() {
				result = structpath.NewDotString(result, key)
			} else {
				result = structpath.NewBracketString(result, key)
			}
		} else if idx, ok := n.Index(); ok {
			result = structpath.NewIndex(result, idx)
		} else if k, v, ok := n.KeyValue(); ok {
			result = structpath.NewKeyValue(result, k, v)
		}
	}
	return result
}
