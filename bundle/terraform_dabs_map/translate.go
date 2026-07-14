package terraform_dabs_map

import (
	"fmt"

	"github.com/databricks/cli/libs/structs/structpath"
)

// DABsPathToTerraform translates a field path from DABs naming conventions
// to Terraform naming conventions for the given resource group.
//
// It is the inverse of TerraformPathToDABs. For groups whose TF schema wraps config fields
// under a structural prefix (e.g. "spec"), that prefix is prepended when the path's first
// segment is listed in DABsToTerraformWrapperFields. Root-level fields and unrecognised
// segments pass through without the wrapper. Each field name segment is looked up in the
// DABsToTerraformRenameMap: when found the TF name is used and the tree descends for the
// remainder of the path. Array indices pass through unchanged without advancing the tree
// position. An unrecognised segment stops further renaming; remaining segments are kept
// as-is. Returns nil when path is nil.
// Returns an error when path is a known DABs-only field with no Terraform equivalent.
//
// The path must be relative to the resource root (e.g. "tasks", not
// "resources.jobs.my_job.tasks").
func DABsPathToTerraform(group string, path *structpath.PathNode) (*structpath.PathNode, error) {
	if path == nil {
		return nil, nil
	}

	if DABsOnlyFields[group].Contains(path) {
		return nil, fmt.Errorf("%s: %q is a DABs-only field with no Terraform equivalent", group, path)
	}

	// For groups with a TF wrapper, prepend it only when the first segment is a known
	// spec field. Unknown paths (root-level outputs, unrecognised segments) pass through unchanged.
	var result *structpath.PathNode
	if wrapper, ok := DABsToTerraformWrappers[group]; ok {
		segs := path.AsSlice()
		if len(segs) > 0 {
			if firstKey, ok := segs[0].StringKey(); ok {
				if _, isWrapped := DABsToTerraformWrapperFields[group][firstKey]; isWrapped {
					result = structpath.NewDotString(nil, wrapper)
				}
			}
		}
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
	return result, nil
}

// TerraformPathToDABs translates a field path from Terraform naming conventions
// to DABs naming conventions for the given resource group.
//
// Each field name segment is looked up in the rename tree: when found the DABs
// name is used and the tree descends into the sub-tree for the remainder of the
// path. Array indices pass through unchanged without advancing the tree position.
// An unrecognised field name stops further renaming; remaining segments are kept
// as-is. Returns nil when path is nil.
// Returns an error when path is a known Terraform-only field with no DABs equivalent.
//
// The path must be relative to the resource root (e.g. "task.library", not
// "resources.jobs.my_job.task.library").
func TerraformPathToDABs(group string, path *structpath.PathNode) (*structpath.PathNode, error) {
	if path == nil {
		return nil, nil
	}

	if TerraformOnlyFields[group].Contains(path) {
		return nil, fmt.Errorf("%s: %q is a Terraform-only field with no DABs equivalent", group, path)
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
	return result, nil
}
