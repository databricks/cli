package configsync

import (
	"cmp"
	"context"
	"errors"
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
	FilePath string
	Change   *ConfigChangeDesc
	// WritePath addresses the field in the file at FilePath, so for an override block
	// it carries the targets.<target> prefix and its sequence indices are block-local.
	WritePath string
	// AltWritePath is tried when WritePath does not exist in the file. It is only set
	// when the change could not be routed to a block, i.e. when the scope had to be
	// guessed; a routed change has exactly one correct path.
	AltWritePath string
	// originalPath addresses the field in the original configuration, before deployment
	// replaced any ${var.X} reference with its value, so restoration can read the
	// reference back. That configuration is merged, so unlike WritePath this path is in
	// merged index space and has no prefix (see sequences.go). Deriving one from the
	// other reads a different element and restores a sibling's reference.
	originalPath string
}

// resolvedChange locates a change in the source YAML.
type resolvedChange struct {
	// path is relative to the block, with merged indices still in place.
	path  *structpath.PatternNode
	steps []sequenceStep
	// leaf is the merged value the change addresses, invalid for a new field.
	leaf dyn.Value
	// operation decides how many destinations the change needs: a removal has to
	// reach every definition, an edit only the one that wins the merge.
	operation OperationType
}

// resolveSelectors converts key-value selectors to the indices of the merged
// configuration and records the sequence elements traversed on the way, so the
// caller can map those positions onto the physical block that owns them.
// Example: "resources.jobs.foo.tasks[task_key='main'].name" -> "resources.jobs.foo.tasks[1].name"
// Returns a PatternNode because for Add operations, [*] may be used as a placeholder for new elements.
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (resolvedChange, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return resolvedChange{}, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var result *structpath.PatternNode
	var steps []sequenceStep
	var currentPath dyn.Path
	currentValue := b.Config.Value()

	for component, n := range nodes {
		if key, ok := n.StringKey(); ok {
			result = structpath.NewPatternStringKey(result, key)
			currentPath = append(currentPath, dyn.Key(key))
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			sequencePath := slices.Clone(currentPath)
			result = structpath.NewPatternIndex(result, idx)
			currentPath = append(currentPath, dyn.Index(idx))
			var element dyn.Value
			if currentValue.IsValid() {
				element, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
				currentValue = element
			}
			steps = append(steps, sequenceStep{
				component:    component,
				sequencePath: sequencePath,
				element:      element,
				// A positional path can address one past the end of the sequence,
				// which is how an add to a positionally-diffed list arrives. There is
				// no element to route by, so it is placed like any other new element.
				newElement: !element.IsValid(),
			})
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return resolvedChange{}, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
			}

			seq, _ := currentValue.AsSequence()
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

			sequencePath := slices.Clone(currentPath)

			if foundIndex == -1 {
				if operation == OperationAdd {
					result = structpath.NewPatternBracketStar(result)
					steps = append(steps, sequenceStep{
						component:    component,
						sequencePath: sequencePath,
						newElement:   true,
					})
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return resolvedChange{}, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			result = structpath.NewPatternIndex(result, foundIndex)
			currentPath = append(currentPath, dyn.Index(foundIndex))
			steps = append(steps, sequenceStep{
				component:    component,
				sequencePath: sequencePath,
				element:      seq[foundIndex],
			})
			currentValue = seq[foundIndex]
			continue
		}
	}

	return resolvedChange{path: result, steps: steps, leaf: currentValue, operation: operation}, nil
}

func pathDepth(pathStr string) int {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return 0
	}
	return len(node.AsSlice())
}

// adjustArrayIndex adjusts the index in a PatternNode based on previous operations.
// When operations are applied sequentially, removals and additions shift array indices.
// This function adjusts the index to account for those shifts.
func adjustArrayIndex(path *structpath.PatternNode, scope string, operations map[string][]struct {
	index     int
	operation OperationType
},
) *structpath.PatternNode {
	originalIndex, ok := path.Index()
	if !ok {
		return path
	}

	parentPath := path.Parent()
	ops := operations[scopedParent(scope, path)]

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
//
// A change that cannot be attributed to a single source location, or whose parent
// in the config is written as a variable reference, is left unapplied rather than
// written to a guessed or non-existent location. The count of those is returned so
// the caller can record it: the command runs unattended, so a change that silently
// disappears looks identical to one that was written. preResolved is the merged
// config with ${...} references still literal, used to detect variable-reference
// parents.
func ResolveChanges(ctx context.Context, b *bundle.Bundle, configChanges Changes, preResolved dyn.Value) ([]FieldChange, int, error) {
	var result []FieldChange
	skipped := 0
	targetName := b.Config.Bundle.Target
	blocks := newBlockResolver(ctx, b)

	resourceKeys := slices.Sorted(maps.Keys(configChanges))

	for _, resourceKey := range resourceKeys {
		resourceChanges := configChanges[resourceKey]

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

		// A key change on a keyed element arrives as a remove of the old key plus an
		// add of the new one, with nothing linking them. Pairing them back up lets a
		// rename be written as a key rewrite in every block that defines the element,
		// which keeps a split element split instead of collapsing it into one scope.
		renames := matchRenamingPairs(b, blocks, resourceKey, resourceChanges)

		indices := newIndexTracker()

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			// The add half of a rename carries no source location of its own; the
			// pair is written from the remove half, which does.
			if _, ok := renames.addPaths[fieldPath]; ok {
				continue
			}
			if reason, ok := renames.unpairedPaths[fieldPath]; ok {
				log.Debugf(ctx, "config-remote-sync: skipping %s: %s", fullPath, reason)
				skipped++
				continue
			}
			rename, isRename := renames.byRemovePath[fieldPath]

			resolved, err := resolveSelectors(fullPath, b, configChange.Operation)
			if err != nil {
				return nil, 0, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}
			resolvedPath := resolved.path

			// Captured before routing rewrites indices to be block-local.
			mergedIndexPath := resolvedPath

			// The field is nested inside a value written as a variable reference (e.g.
			// spark_conf: ${var.spark_conf}), which is a scalar in the file with no node
			// to place the key or index under. Leave it unapplied, and record the skip on
			// the shared change so the output reports "skip" rather than a write that never
			// happens (matching the reclassification below).
			if parentIsVariableReference(preResolved, mergedIndexPath.String()) {
				log.Debugf(ctx, "config-remote-sync: skipping %s: parent is a variable reference", fullPath)
				configChange.Operation = OperationSkip
				skipped++
				continue
			}

			destinations, err := routeChange(blocks, resolved, isRename)
			if err != nil {
				if !errors.Is(err, errAmbiguousBlock) {
					return nil, 0, err
				}
				// Applying this change would mean guessing a location. Leave it
				// unapplied; a later run can pick it up.
				log.Debugf(ctx, "config-remote-sync: skipping %s: %v", fullPath, err)
				skipped++
				continue
			}
			if len(destinations) == 0 {
				// A rename whose element could not be attributed to a block: the whole
				// pair is left for a later run rather than half-applied.
				log.Debugf(ctx, "config-remote-sync: skipping %s: the renamed element has no source location", fullPath)
				skipped++
				continue
			}

			for _, destination := range destinations {
				block := destination.block
				resolvedPath := destination.path
				// Each destination gets its own copy: the operation and the value
				// are rewritten below per destination, and one block's rewrite must
				// not leak into the next.
				destChange := &ConfigChangeDesc{
					Operation: configChange.Operation,
					Value:     configChange.Value,
					LocalEdit: configChange.LocalEdit,
				}

				// Index bookkeeping is scoped to a block so that shifts caused by
				// operations on one block cannot move indices in another.
				scope := destination.scopeKey()

				if isRename {
					result = append(result, renameKeyChange(blocks, *block, scope, indices,
						resolvedPath, mergedIndexPath, rename, destChange))
					continue
				}

				resolvedPath = indices.place(scope, resolvedPath, destChange.Operation)

				writePath, altWritePath := writeAddress(blocks, block, targetName, resolvedPath)

				// A change routed to a block has a known destination file even when the leaf
				// itself is new, but "defined in the config" must still be decided by
				// the leaf: a field with no source location is added, not replaced.
				filePath := resolved.leaf.Location().File
				isDefinedInConfig := filePath != ""
				if block != nil && isDefinedInConfig {
					filePath = block.file
				}

				if !isDefinedInConfig {
					if destChange.Operation == OperationRemove {
						// If the field is not defined in the config and the operation is remove, it is more likely a CLI default
						// in this case we skip the change
						continue
					}

					if destChange.Operation == OperationReplace {
						// If the field is not defined in the config and the operation is replace, it is more likely a CLI default
						// in this case we add it explicitly to the resource location.
						// The reclassification is also recorded on the shared change so
						// the command's output reports what was actually written.
						destChange.Operation = OperationAdd
						configChange.Operation = OperationAdd
					}

					if block != nil {
						// The enclosing element was resolved to a block, so a new field
						// on it belongs in that same block.
						filePath = block.file
					} else {
						resourceLocation := b.Config.GetLocation(resourceKey)
						filePath = resourceLocation.File
					}
					if filePath == "" {
						return nil, 0, fmt.Errorf("failed to find location for resource %s for a field %s", resourceKey, fieldPath)
					}

					log.Debugf(ctx, "Field %s has no location, using %s", fullPath, filePath)
				}

				if (destChange.Operation == OperationAdd || destChange.Operation == OperationReplace) && b.SyncRootPath != "" {
					destChange = &ConfigChangeDesc{
						Operation: destChange.Operation,
						Value:     translateWorkspacePaths(destChange.Value, b.SyncRootPath, b.SyncRoot, filepath.Dir(filePath)),
					}
				}

				result = append(result, FieldChange{
					FilePath:     filePath,
					Change:       destChange,
					WritePath:    writePath,
					AltWritePath: altWritePath,
					originalPath: mergedIndexPath.String(),
				})
			}
		}
	}

	return result, skipped, nil
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

// writePaths lists the paths to try, in order: the addressed one, then the fallback
// for a change whose scope had to be guessed.
func (c FieldChange) writePaths() []string {
	if c.AltWritePath == "" {
		return []string{c.WritePath}
	}
	return []string{c.WritePath, c.AltWritePath}
}
