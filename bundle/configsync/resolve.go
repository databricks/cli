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
	FilePath        string
	Change          *ConfigChangeDesc
	FieldCandidates []string
}

// sequenceStep records a sequence element that a change path navigated through.
// The element's index inside a physical block is only known once the block has
// been chosen, so the merged position is kept until then.
type sequenceStep struct {
	// position in the pattern built so far, i.e. how many components precede
	// this sequence's index.
	component int
	// path of the sequence relative to a block, e.g. resources.jobs.j.tasks.
	sequencePath dyn.Path
	element      dyn.Value
	// newElement marks an Add whose key is not in the merged sequence yet.
	newElement bool
}

// resolvedChange locates a change in the source YAML.
type resolvedChange struct {
	// path is relative to the block, with merged indices still in place.
	path  *structpath.PatternNode
	steps []sequenceStep
	// leaf is the merged value the change addresses, invalid for a new field.
	leaf dyn.Value
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

	return resolvedChange{path: result, steps: steps, leaf: currentValue}, nil
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
	parentPathStr := scope + parentPath.String()
	ops := operations[parentPathStr]

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
		renames := pairRenames(b, blocks, resourceKey, resourceChanges)

		// Create indices map for this resource, path -> indices, that we could use to replace with added elements
		indicesToReplaceMap := make(map[string][]int)

		indexOperations := make(map[string][]struct {
			index     int
			operation OperationType
		})

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			// The add half of a rename carries no source location of its own; the
			// pair is written from the remove half, which does.
			if _, ok := renames.addPaths[fieldPath]; ok {
				continue
			}
			if _, ok := renames.unpairedPaths[fieldPath]; ok {
				log.Debugf(ctx, "config-remote-sync: skipping %s: a split element cannot be removed and recreated in one run", fullPath)
				continue
			}
			rename, isRename := renames.byRemovePath[fieldPath]

			resolved, err := resolveSelectors(fullPath, b, configChange.Operation)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}
			resolvedPath := resolved.path

			// A sequence element addressed by the merged view has to be mapped onto
			// the physical block that defines it, because the merged order and the
			// per-block order differ once a sequence is split across blocks. An
			// element assembled from several blocks has a part in each, so removing
			// or renaming it yields one destination per block.
			destinations := []routeDestination{{path: resolvedPath}}
			routed := false
			if blocks != nil && len(resolved.steps) > 0 {
				var routeErr error
				if isRename {
					destinations, routeErr = routeRenameElement(b, blocks, resourceKey, fieldPath)
					if routeErr == nil && len(destinations) == 0 {
						// The element could not be attributed to a block; leave the
						// whole pair for a later run rather than half-applying it.
						continue
					}
				} else {
					destinations, routeErr = blocks.routeDestinations(resolved)
				}
				if routeErr != nil {
					if errors.Is(routeErr, errAmbiguousBlock) {
						// Applying this change would mean guessing a location.
						// Leave it unapplied; a later run can pick it up.
						log.Debugf(ctx, "config-remote-sync: skipping %s: %v", fullPath, routeErr)
						continue
					}
					return nil, fmt.Errorf("failed to route change %s: %w", fullPath, routeErr)
				}
				routed = true
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
				scope := ""
				if routed {
					scope = block.scopeKey()
				}

				// A rename rewrites only the key field of the element, so it is a
				// replace at the element's position rather than a change to the
				// element itself. The index is still adjusted below, because
				// removals earlier in the same block shift it.
				if isRename {
					resolvedPath = adjustArrayIndex(resolvedPath, scope, indexOperations)
					destChange.Operation = OperationReplace
					destChange.Value = rename.newKey
					resolvedPath = structpath.NewPatternStringKey(resolvedPath, rename.keyField)
					candidate := blocks.candidatePath(block, resolvedPath.String())
					result = append(result, FieldChange{
						FilePath:        block.file,
						Change:          destChange,
						FieldCandidates: []string{candidate},
					})
					continue
				}

				// If the element is removed, we can use the index to replace it with added element
				// That may improve the diff in cases when the task is recreated because of renaming
				if destChange.Operation == OperationRemove {
					freeIndex, ok := resolvedPath.Index()
					if ok {
						parentPath := scope + resolvedPath.Parent().String()
						indicesToReplaceMap[parentPath] = append(indicesToReplaceMap[parentPath], freeIndex)
					}
				}

				if destChange.Operation == OperationAdd && resolvedPath.BracketStar() {
					parentPath := scope + resolvedPath.Parent().String()
					indices, ok := indicesToReplaceMap[parentPath]
					if ok && len(indices) > 0 {
						index := indices[0]
						indicesToReplaceMap[parentPath] = indices[1:]
						resolvedPath = structpath.NewPatternIndex(resolvedPath.Parent(), index)
					}
				}

				resolvedPath = adjustArrayIndex(resolvedPath, scope, indexOperations)

				// Track this operation for future index adjustments (only for array element operations)
				if originalIndex, ok := resolvedPath.Index(); ok {
					parentPath := scope + resolvedPath.Parent().String()
					indexOperations[parentPath] = append(indexOperations[parentPath], struct {
						index     int
						operation OperationType
					}{originalIndex, destChange.Operation})
				}

				resolvedPathStr := resolvedPath.String()
				var candidates []string
				if routed {
					// The block is known, so there is exactly one path to write.
					candidates = []string{blocks.candidatePath(block, resolvedPathStr)}
				} else {
					candidates = []string{resolvedPathStr}
					if targetName != "" {
						targetPrefixedPath := "targets." + targetName + "." + resolvedPathStr
						candidates = append(candidates, targetPrefixedPath)
					}
				}

				// A routed change has a known destination file even when the leaf
				// itself is new, but "defined in the config" must still be decided by
				// the leaf: a field with no source location is added, not replaced.
				filePath := resolved.leaf.Location().File
				isDefinedInConfig := filePath != ""
				if routed && isDefinedInConfig {
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

					if routed {
						// The enclosing element was resolved to a block, so a new field
						// on it belongs in that same block.
						filePath = block.file
					} else {
						resourceLocation := b.Config.GetLocation(resourceKey)
						filePath = resourceLocation.File
					}
					if filePath == "" {
						return nil, fmt.Errorf("failed to find location for resource %s for a field %s", resourceKey, fieldPath)
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
					FilePath:        filePath,
					Change:          destChange,
					FieldCandidates: candidates,
				})
			}
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
