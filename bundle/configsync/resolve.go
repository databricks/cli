package configsync

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/structs/structpath"
)

type FieldChange struct {
	FilePath        string
	Change          *ConfigChangeDesc
	FieldCandidates []string
}

// resolveSelectors converts key-value selectors to numeric indices that match
// the YAML file positions. It also returns the location of the resolved leaf value.
// Example: "resources.jobs.foo.tasks[task_key='main'].name" -> "resources.jobs.foo.tasks[1].name"
// Returns a PatternNode because for Add operations, [*] may be used as a placeholder for new elements.
//
// The returned bool reports whether the resolved element was contributed by a
// target override block. Its numeric index is then local to that block, so only
// the targets.<t>.-prefixed candidate is valid (see ResolveChanges).
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (*structpath.PatternNode, dyn.Location, bool, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return nil, dyn.Location{}, false, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var result *structpath.PatternNode
	inOverrideBlock := false
	currentValue := b.Config.Value()

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			result = structpath.NewPatternStringKey(result, key)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			// A numeric index reaches here when the change path is positional
			// rather than keyed — i.e. the resource registered no KeyedSlices
			// entry for this sequence (e.g. pipeline `clusters`, which structdiff
			// compares by position). When such a sequence is split across a
			// top-level and a target override block, idx is the merged-sequence
			// position; convert it to the block-local index and flag target-block
			// elements, exactly as the keyed selector below does.
			if currentValue.IsValid() && currentValue.Kind() == dyn.KindSequence {
				seq, _ := currentValue.AsSequence()
				seqLocations := currentValue.Locations()
				if len(seqLocations) >= 2 && idx >= 0 && idx < len(seq) {
					result = structpath.NewPatternIndex(result, yamlFileIndex(seq, idx, seqLocations))
					inOverrideBlock = inOverrideBlock || elementInOverrideBlock(seq[idx].Location(), seqLocations)
					currentValue = seq[idx]
					continue
				}
			}
			result = structpath.NewPatternIndex(result, idx)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return nil, dyn.Location{}, false, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
			}

			seq, _ := currentValue.AsSequence()
			seqLocations := currentValue.Locations()
			foundIndex := -1

			for i, elem := range seq {
				keyValue, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(key)})
				if err != nil {
					continue
				}

				if keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					foundIndex = i
					break
				}
			}

			if foundIndex == -1 {
				if operation == OperationAdd {
					result = structpath.NewPatternBracketStar(result)
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return nil, dyn.Location{}, false, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			// Mutators may reorder sequence elements (e.g., tasks sorted by task_key).
			// Use location information to determine the original YAML file position.
			yamlIndex := yamlFileIndex(seq, foundIndex, seqLocations)
			result = structpath.NewPatternIndex(result, yamlIndex)
			// yamlIndex is local to the element's own block; latch so a nested
			// single-block selector further down doesn't clear a target match.
			inOverrideBlock = inOverrideBlock || elementInOverrideBlock(seq[foundIndex].Location(), seqLocations)
			currentValue = seq[foundIndex]
			continue
		}
	}

	return result, currentValue.Location(), inOverrideBlock, nil
}

// elementInOverrideBlock reports whether a merged sequence element came from a
// target override block rather than the top-level block. merge keeps the
// reference (top-level) locations first, so seqLocations[0] is the top-level
// anchor; an element anchored elsewhere came from a target block and its
// block-local index is only valid under the targets.<t>. prefix. A resource is
// declared once across top-level files (duplicate keys are rejected at load), so
// a second anchor can only be a target override, never a second top-level file.
func elementInOverrideBlock(elemLoc dyn.Location, seqLocations []dyn.Location) bool {
	if len(seqLocations) < 2 || elemLoc.File == "" {
		return false
	}
	topAnchor := seqLocations[0]
	return elemLoc.File != topAnchor.File || blockAnchor(elemLoc, seqLocations) != topAnchor.Line
}

// yamlFileIndex returns a sequence element's index within its own config block.
// Tasks/clusters merged from a top-level block and a target override are one
// sorted in-memory sequence but patched per block in the YAML, so the merged
// index is wrong. seqLocations holds one anchor per block; counting only
// same-block, earlier-line elements yields the block-local index. A single-block
// sequence reduces to a plain same-file count.
func yamlFileIndex(seq []dyn.Value, sortedIndex int, seqLocations []dyn.Location) int {
	matchLocation := seq[sortedIndex].Location()
	if matchLocation.File == "" {
		return sortedIndex
	}

	matchAnchor := blockAnchor(matchLocation, seqLocations)

	yamlIndex := 0
	for i, elem := range seq {
		if i == sortedIndex {
			continue
		}
		loc := elem.Location()
		if loc.File == matchLocation.File && loc.Line < matchLocation.Line && blockAnchor(loc, seqLocations) == matchAnchor {
			yamlIndex++
		}
	}
	return yamlIndex
}

// blockAnchor returns the config block a sequence element belongs to, identified by the
// line of that block's sequence node. Blocks never interleave within a file, so the
// element's block is the anchor with the greatest line at or before the element in the
// same file. Returns 0 when no anchor matches (single unlocated block), which keeps all
// elements in one group.
func blockAnchor(loc dyn.Location, seqLocations []dyn.Location) int {
	anchor := 0
	for _, l := range seqLocations {
		if l.File == loc.File && l.Line <= loc.Line && l.Line > anchor {
			anchor = l.Line
		}
	}
	return anchor
}

func pathDepth(pathStr string) int {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return 0
	}
	return len(node.AsSlice())
}

// pathTouchesSplitSequence reports whether resolving path crosses a sequence
// assembled from multiple YAML blocks. The merged value does not retain enough
// per-element provenance to map every remote edit back to the correct block, so
// callers must not emit patches for resources containing such changes.
func pathTouchesSplitSequence(pathStr string, b *bundle.Bundle) (bool, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return false, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	currentValue := b.Config.Value()
	for _, n := range node.AsSlice() {
		if currentValue.IsValid() && currentValue.Kind() == dyn.KindSequence && len(currentValue.Locations()) > 1 {
			return true, nil
		}

		if key, ok := n.StringKey(); ok {
			currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			continue
		}
		if idx, ok := n.Index(); ok {
			currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			continue
		}
		if key, value, ok := n.KeyValue(); ok {
			seq, ok := currentValue.AsSequence()
			if !ok {
				return false, nil
			}
			currentValue = dyn.Value{}
			for _, elem := range seq {
				keyValue, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(key)})
				if err == nil && keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					currentValue = elem
					break
				}
			}
		}
	}

	return currentValue.IsValid() && currentValue.Kind() == dyn.KindSequence && len(currentValue.Locations()) > 1, nil
}

// blockScopedParent namespaces per-sequence index bookkeeping by block. A target
// override block and the top-level block share the same unprefixed parent path
// but keep block-local indices, so add/remove operations in one block must not
// shift indices resolved in the other. Prefixing the override block's key with
// targets.<t>. keeps the two buckets separate.
func blockScopedParent(parentPathStr, targetName string, inOverrideBlock bool) string {
	if inOverrideBlock {
		return "targets." + targetName + "." + parentPathStr
	}
	return parentPathStr
}

// reuseFreedIndex pops the first index freed by a prior removal and rewrites path
// to point at that slot, so a recreated (renamed) element reuses the removed
// element's position instead of appending. Returns the remaining indices.
func reuseFreedIndex(indices []int, path *structpath.PatternNode) ([]int, *structpath.PatternNode) {
	return indices[1:], structpath.NewPatternIndex(path.Parent(), indices[0])
}

// adjustArrayIndex adjusts the index in a PatternNode based on previous operations.
// When operations are applied sequentially, removals and additions shift array indices.
// This function adjusts the index to account for those shifts. parentKey selects the
// block-scoped bucket of operations that apply to this element's sequence block.
func adjustArrayIndex(path *structpath.PatternNode, parentKey string, operations map[string][]struct {
	index     int
	operation OperationType
},
) *structpath.PatternNode {
	originalIndex, ok := path.Index()
	if !ok {
		return path
	}

	parentPath := path.Parent()
	ops := operations[parentKey]

	adjustment := 0
	for _, op := range ops {
		if op.index < originalIndex {
			switch op.operation {
			case OperationRemove:
				adjustment--
			case OperationAdd:
				adjustment++
			default:
			}
		}
	}

	adjustedIndex := max(originalIndex+adjustment, 0)

	return structpath.NewPatternIndex(parentPath, adjustedIndex)
}

// ResolveChanges resolves selectors and computes field path candidates for each change.
func ResolveChanges(ctx context.Context, b *bundle.Bundle, configChanges Changes) ([]FieldChange, error) {
	var result []FieldChange
	targetName := b.Config.Bundle.Target

	resourceKeys := slices.Sorted(maps.Keys(configChanges))

	for _, resourceKey := range resourceKeys {
		resourceChanges := configChanges[resourceKey]

		// A split sequence can contain an unpaired remove/add rename and elements
		// whose target-block location was collapsed by the merge. Skip the whole
		// resource before resolving any fields so a compound edit cannot be applied
		// partially and corrupt the source YAML.
		skipResource := false
		for fieldPath := range resourceChanges {
			split, err := pathTouchesSplitSequence(resourceKey+"."+fieldPath, b)
			if err != nil {
				return nil, err
			}
			if split {
				skipResource = true
				break
			}
		}
		if skipResource {
			log.Debugf(ctx, "Skipping config sync for resource %s because a changed sequence is defined in multiple YAML blocks", resourceKey)
			continue
		}

		fieldPaths := make([]string, 0, len(resourceChanges))
		fieldPathsDepths := map[string]int{}
		for fieldPath := range resourceChanges {
			fieldPaths = append(fieldPaths, fieldPath)
			fieldPathsDepths[fieldPath] = pathDepth(fieldPath)
		}

		// Sort field paths by depth (deeper first), then operation type (removals before adds), then alphabetically
		slices.SortStableFunc(fieldPaths, func(a, b string) int {
			depthA := fieldPathsDepths[a]
			depthB := fieldPathsDepths[b]

			if depthA != depthB {
				return cmp.Compare(depthB, depthA)
			}

			opA := resourceChanges[a].Operation
			opB := resourceChanges[b].Operation

			if opA == OperationRemove && opB != OperationRemove {
				return -1
			}
			if opA != OperationRemove && opB == OperationRemove {
				return 1
			}

			return cmp.Compare(a, b)
		})

		// Create indices map for this resource, path -> indices, that we could use to replace with added elements
		indicesToReplaceMap := make(map[string][]int)

		indexOperations := make(map[string][]struct {
			index     int
			operation OperationType
		})

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, resolvedLocation, inOverrideBlock, err := resolveSelectors(fullPath, b, configChange.Operation)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}

			// If the element is removed, we can use the index to replace it with added element
			// That may improve the diff in cases when the task is recreated because of renaming
			if configChange.Operation == OperationRemove {
				freeIndex, ok := resolvedPath.Index()
				if ok {
					parentKey := blockScopedParent(resolvedPath.Parent().String(), targetName, inOverrideBlock)
					indicesToReplaceMap[parentKey] = append(indicesToReplaceMap[parentKey], freeIndex)
				}
			}

			if configChange.Operation == OperationAdd && resolvedPath.BracketStar() {
				// A newly added keyed element has no location, so inOverrideBlock is
				// false here. Reuse an index freed by a removal in the same sync (a
				// rename diffs as remove+add): try the top-level block first, then the
				// target override block. Reusing a target-block slot means this add
				// renames a target-only element, so it must stay in that block — latch
				// inOverrideBlock so it is not relocated to the top-level block.
				topKey := resolvedPath.Parent().String()
				targetKey := blockScopedParent(topKey, targetName, true)
				switch {
				case len(indicesToReplaceMap[topKey]) > 0:
					indicesToReplaceMap[topKey], resolvedPath = reuseFreedIndex(indicesToReplaceMap[topKey], resolvedPath)
				case targetName != "" && len(indicesToReplaceMap[targetKey]) > 0:
					indicesToReplaceMap[targetKey], resolvedPath = reuseFreedIndex(indicesToReplaceMap[targetKey], resolvedPath)
					inOverrideBlock = true
				}
			}

			parentKey := blockScopedParent(resolvedPath.Parent().String(), targetName, inOverrideBlock)
			resolvedPath = adjustArrayIndex(resolvedPath, parentKey, indexOperations)

			// Track this operation for future index adjustments (only for array element operations)
			if originalIndex, ok := resolvedPath.Index(); ok {
				indexOperations[parentKey] = append(indexOperations[parentKey], struct {
					index     int
					operation OperationType
				}{originalIndex, configChange.Operation})
			}

			resolvedPathStr := resolvedPath.String()
			var candidates []string
			targetPrefixedPath := "targets." + targetName + "." + resolvedPathStr
			switch {
			case inOverrideBlock:
				// inOverrideBlock is only ever set when a target override block
				// contributed to the sequence, which requires a selected target, so
				// targetName is non-empty here and targetPrefixedPath is well-formed.
				// The index is local to that override block, so only the target-prefixed
				// candidate points at the right element. Emitting the unprefixed
				// candidate too would let applyChange write the same index into the
				// top-level block and silently patch the wrong element.
				candidates = []string{targetPrefixedPath}
			case targetName != "":
				candidates = []string{resolvedPathStr, targetPrefixedPath}
			default:
				candidates = []string{resolvedPathStr}
			}

			filePath := resolvedLocation.File

			isDefinedInConfig := filePath != ""
			if !isDefinedInConfig {
				if configChange.Operation == OperationRemove {
					// If the field is not defined in the config and the operation is remove, it is more likely a CLI default
					// in this case we skip the change
					continue
				}

				if configChange.Operation == OperationReplace {
					// If the field is not defined in the config and the operation is replace, it is more likely a CLI default
					// in this case we add it explicitly to the resource location
					configChange.Operation = OperationAdd
				}

				resourceLocation := b.Config.GetLocation(resourceKey)
				filePath = resourceLocation.File
				if filePath == "" {
					return nil, fmt.Errorf("failed to find location for resource %s for a field %s", resourceKey, fieldPath)
				}

				log.Debugf(ctx, "Field %s has no location, using resource location: %s", fullPath, filePath)
			}

			if (configChange.Operation == OperationAdd || configChange.Operation == OperationReplace) && b.SyncRootPath != "" {
				configChange = &ConfigChangeDesc{
					Operation: configChange.Operation,
					Value:     translateWorkspacePaths(configChange.Value, b.SyncRootPath, b.SyncRoot, filepath.Dir(filePath)),
				}
			}

			result = append(result, FieldChange{
				FilePath:        filePath,
				Change:          configChange,
				FieldCandidates: candidates,
			})
		}
	}

	return result, nil
}

// translateWorkspacePaths recursively converts absolute workspace paths to relative
// paths when they fall within the bundle's sync root. Paths are made relative to
// targetDir (the directory of the YAML file being patched). For notebook paths
// where the extension was stripped by translate_paths, it restores the extension
// by checking the local filesystem.
func translateWorkspacePaths(value any, syncRootPath string, syncRoot fs.FS, targetDir string) any {
	switch v := value.(type) {
	case string:
		after, ok := strings.CutPrefix(v, syncRootPath+"/")
		if !ok {
			return v
		}
		after = resolveNotebookExtension(syncRoot, after)
		fullPath := filepath.Join(syncRootPath, after)
		relPath, err := filepath.Rel(targetDir, fullPath)
		if err != nil {
			return "./" + after
		}
		relPathSlash := filepath.ToSlash(relPath)
		if !strings.HasPrefix(relPathSlash, "..") {
			relPathSlash = "./" + relPathSlash
		}
		return relPathSlash
	case map[string]any:
		for key, val := range v {
			v[key] = translateWorkspacePaths(val, syncRootPath, syncRoot, targetDir)
		}
		return v
	case []any:
		for i, val := range v {
			v[i] = translateWorkspacePaths(val, syncRootPath, syncRoot, targetDir)
		}
		return v
	default:
		return value
	}
}

// resolveNotebookExtension checks if a relative path refers to a notebook whose
// extension was stripped by translate_paths. If the file doesn't exist as-is but
// exists with a notebook extension, the extension is appended.
func resolveNotebookExtension(syncRoot fs.FS, relPath string) string {
	if syncRoot == nil {
		return relPath
	}
	if _, err := fs.Stat(syncRoot, relPath); err == nil {
		return relPath
	}
	for _, ext := range notebook.Extensions {
		if _, err := fs.Stat(syncRoot, relPath+ext); err == nil {
			return relPath + ext
		}
	}
	return relPath
}
