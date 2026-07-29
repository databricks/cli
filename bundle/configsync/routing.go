package configsync

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

type mergedAncestor struct {
	path  *structpath.PatternNode
	value dyn.Value
}

type mergedSelection struct {
	path               *structpath.PatternNode
	value              dyn.Value
	ancestors          []mergedAncestor
	newSequenceElement bool
}

func resolveMergedSelection(pathStr string, b *bundle.Bundle, operation OperationType) (mergedSelection, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return mergedSelection{}, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	current := b.Config.Value()
	var resolved *structpath.PatternNode
	selection := mergedSelection{}
	if current.IsValid() {
		selection.ancestors = append(selection.ancestors, mergedAncestor{value: current})
	}

	for _, component := range node.AsSlice() {
		if key, ok := component.StringKey(); ok {
			resolved = structpath.NewPatternStringKey(resolved, key)
			if !current.IsValid() {
				continue
			}
			next, err := dyn.GetByPath(current, dyn.NewPath(dyn.Key(key)))
			if err != nil {
				current = dyn.InvalidValue
				continue
			}
			current = next
			selection.ancestors = append(selection.ancestors, mergedAncestor{path: resolved, value: current})
			continue
		}

		if index, ok := component.Index(); ok {
			resolved = structpath.NewPatternIndex(resolved, index)
			if !current.IsValid() {
				continue
			}
			sequence, ok := current.AsSequence()
			if !ok || index < 0 || index >= len(sequence) {
				current = dyn.InvalidValue
				continue
			}
			current = sequence[index]
			selection.ancestors = append(selection.ancestors, mergedAncestor{path: resolved, value: current})
			continue
		}

		if key, value, ok := component.KeyValue(); ok {
			if !current.IsValid() || current.Kind() != dyn.KindSequence {
				return mergedSelection{}, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
			}

			sequence := current.MustSequence()
			found := -1
			for i, element := range sequence {
				keyValue, err := dyn.GetByPath(element, dyn.NewPath(dyn.Key(key)))
				if err == nil && keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					found = i
					break
				}
			}

			if found < 0 {
				if operation != OperationAdd {
					return mergedSelection{}, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
				}
				resolved = structpath.NewPatternBracketStar(resolved)
				selection.newSequenceElement = true
				current = dyn.InvalidValue
				continue
			}

			resolved = structpath.NewPatternIndex(resolved, found)
			current = sequence[found]
			selection.ancestors = append(selection.ancestors, mergedAncestor{path: resolved, value: current})
			continue
		}

		return mergedSelection{}, fmt.Errorf("unsupported path component in %s", pathStr)
	}

	selection.path = resolved
	selection.value = current
	return selection, nil
}

func patternFromDynPath(path dyn.Path) *structpath.PatternNode {
	var pattern *structpath.PatternNode
	for _, component := range path {
		if key := component.Key(); key != "" {
			pattern = structpath.NewPatternStringKey(pattern, key)
		} else {
			pattern = structpath.NewPatternIndex(pattern, component.Index())
		}
	}
	return pattern
}

func appendPatternSuffix(base dyn.Path, full, prefix *structpath.PatternNode) (*structpath.PatternNode, error) {
	result := patternFromDynPath(base)
	fullNodes := full.AsSlice()
	prefixLength := 0
	if prefix != nil {
		prefixLength = prefix.Len()
	}
	if prefixLength > len(fullNodes) {
		return nil, errors.New("path prefix is longer than path")
	}

	for _, component := range fullNodes[prefixLength:] {
		if key, ok := component.StringKey(); ok {
			result = structpath.NewPatternStringKey(result, key)
			continue
		}
		if index, ok := component.Index(); ok {
			result = structpath.NewPatternIndex(result, index)
			continue
		}
		if component.BracketStar() {
			result = structpath.NewPatternBracketStar(result)
			continue
		}
		return nil, fmt.Errorf("unsupported path suffix in %s", full.String())
	}
	return result, nil
}

func cloneChangeWithOperation(change *ConfigChangeDesc, operation OperationType) *ConfigChangeDesc {
	copy := *change
	copy.Operation = operation
	return &copy
}

func sourceSiblingsForSelection(sources *sourceIndex, selection mergedSelection, target string) []dyn.Value {
	for _, ancestor := range slices.Backward(selection.ancestors) {

		if ancestor.value.Kind() != dyn.KindSequence {
			continue
		}
		var result []dyn.Value
		for _, ref := range sources.refsFor(ancestor.value, ancestor.path, target) {
			sequence, ok := ref.value.AsSequence()
			if ok {
				result = append(result, sequence...)
			}
		}
		return result
	}
	return nil
}

func sourceRefsSpanScopes(refs []sourceRef, target string) bool {
	if target == "" || len(refs) < 2 {
		return false
	}
	hasTop, hasTarget := false, false
	for _, ref := range refs {
		if sourceRefIsTarget(ref, target) {
			hasTarget = true
		} else {
			hasTop = true
		}
	}
	return hasTop && hasTarget
}

func sourceRefForScope(refs []sourceRef, target string, wantTarget bool) (sourceRef, bool, error) {
	var match sourceRef
	count := 0
	for _, ref := range refs {
		if sourceRefIsTarget(ref, target) == wantTarget {
			match = ref
			count++
		}
	}
	if count > 1 {
		return sourceRef{}, false, errors.New("multiple physical source blocks match the destination scope")
	}
	return match, count == 1, nil
}

func normalizeSequenceAdd(
	sources *sourceIndex,
	ref sourceRef,
	path *structpath.PatternNode,
	change *ConfigChangeDesc,
) (*structpath.PatternNode, *ConfigChangeDesc) {
	_, hasIndex := path.Index()
	if !hasIndex && !path.BracketStar() {
		return path, change
	}
	parent := path.Parent()
	concrete, ok := concretePath(parent)
	if !ok {
		return path, change
	}
	source := sources.files[ref.file]
	if source == nil {
		return path, change
	}
	if value, err := dyn.GetByPath(source.root, concrete); err == nil && value.Kind() == dyn.KindSequence {
		return path, change
	}
	copy := *change
	copy.Value = []any{change.Value}
	copy.sequenceElementAdd = true
	return parent, &copy
}

func concretePath(path *structpath.PatternNode) (dyn.Path, bool) {
	result := dyn.EmptyPath
	for _, component := range path.AsSlice() {
		if key, ok := component.StringKey(); ok {
			result = result.Append(dyn.Key(key))
			continue
		}
		if index, ok := component.Index(); ok {
			result = result.Append(dyn.Index(index))
			continue
		}
		return nil, false
	}
	return result, true
}

func makeFieldChange(
	b *bundle.Bundle,
	sources *sourceIndex,
	ref sourceRef,
	path *structpath.PatternNode,
	change *ConfigChangeDesc,
	siblings []dyn.Value,
) FieldChange {
	if (change.Operation == OperationAdd || change.Operation == OperationReplace) && b.SyncRootPath != "" {
		change = cloneChangeWithOperation(change, change.Operation)
		change.Value = translateWorkspacePaths(change.Value, b.SyncRootPath, b.SyncRoot, filepath.Dir(ref.file))
	}
	fieldChange := FieldChange{
		FilePath:        ref.file,
		Change:          change,
		FieldCandidates: []string{path.String()},
		sourceSiblings:  slices.Clone(siblings),
	}
	if source := sources.files[ref.file]; source != nil {
		fieldChange.originalFileContent = slices.Clone(source.content)
		if concrete, ok := concretePath(path); ok {
			fieldChange.sourceValue, _ = dyn.GetByPath(source.root, concrete)
		}
	}
	return fieldChange
}

func routeFieldChange(
	ctx context.Context,
	b *bundle.Bundle,
	sources *sourceIndex,
	resourceKey string,
	fieldPath string,
	change *ConfigChangeDesc,
) ([]FieldChange, error) {
	fullPath := resourceKey + "." + fieldPath
	resourceDepth := pathDepth(resourceKey)
	selection, err := resolveMergedSelection(fullPath, b, change.Operation)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
	}
	target := b.Config.Bundle.Target

	if change.Operation == OperationRemove {
		if !selection.value.IsValid() {
			return nil, fmt.Errorf("cannot remove %s because it is not present in the source configuration", fullPath)
		}
		refs := sources.refsFor(selection.value, selection.path, target)
		if len(refs) == 0 {
			// Values injected by mutators or presets have no source node to remove.
			return nil, nil
		}
		if sourceRefsSpanScopes(refs, target) {
			return nil, fmt.Errorf("cannot remove %s because it is defined in both top-level and target scopes", fullPath)
		}
		result := make([]FieldChange, 0, len(refs))
		for _, ref := range refs {
			result = append(result, makeFieldChange(b, sources, ref, patternFromDynPath(ref.path), change, nil))
		}
		return result, nil
	}

	if selection.value.IsValid() {
		refs := sources.refsFor(selection.value, selection.path, target)
		if len(refs) > 0 {
			operation := change.Operation
			if operation == OperationAdd {
				operation = OperationReplace
			}
			ref, ok := chooseSourceRef(refs, target, target != "")
			if !ok {
				return nil, fmt.Errorf("failed to choose source location for %s", fullPath)
			}
			return []FieldChange{makeFieldChange(b, sources, ref, patternFromDynPath(ref.path), cloneChangeWithOperation(change, operation), nil)}, nil
		}
	}

	operation := change.Operation
	if operation == OperationReplace {
		operation = OperationAdd
		// Preserve the command's existing output contract: a resolved value with
		// no physical source is reported as an add because that is the YAML edit.
		change.Operation = OperationAdd
	}

	siblings := sourceSiblingsForSelection(sources, selection, target)
	if !selection.newSequenceElement {
		for _, ancestor := range slices.Backward(selection.ancestors) {
			if ancestor.path == nil || ancestor.path.Len() < resourceDepth {
				continue
			}

			refs := sources.refsFor(ancestor.value, ancestor.path, target)
			if len(refs) == 0 {
				continue
			}
			ref, _ := chooseSourceRef(refs, target, target != "")
			physicalPath, err := appendPatternSuffix(ref.path, selection.path, ancestor.path)
			if err != nil {
				return nil, err
			}
			updatedChange := cloneChangeWithOperation(change, operation)
			if updatedChange.Operation == OperationAdd {
				physicalPath, updatedChange = normalizeSequenceAdd(sources, ref, physicalPath, updatedChange)
			}
			return []FieldChange{makeFieldChange(b, sources, ref, physicalPath, updatedChange, siblings)}, nil
		}
		return nil, fmt.Errorf("failed to find source location for %s", fullPath)
	}

	// A genuinely new sequence element defaults to the broadest (top-level)
	// scope. Search all ancestors for that scope before considering the target.
	passes := []bool{false}
	if target != "" {
		passes = append(passes, true)
	}
	for _, wantTarget := range passes {
		for _, ancestor := range slices.Backward(selection.ancestors) {
			if ancestor.path == nil || ancestor.path.Len() < resourceDepth {
				continue
			}

			refs := sources.refsFor(ancestor.value, ancestor.path, target)
			ref, ok, err := sourceRefForScope(refs, target, wantTarget)
			if err != nil {
				return nil, fmt.Errorf("cannot place %s: %w", fullPath, err)
			}
			if !ok {
				continue
			}
			physicalPath, err := appendPatternSuffix(ref.path, selection.path, ancestor.path)
			if err != nil {
				return nil, err
			}
			updatedChange := cloneChangeWithOperation(change, operation)
			physicalPath, updatedChange = normalizeSequenceAdd(sources, ref, physicalPath, updatedChange)
			return []FieldChange{makeFieldChange(b, sources, ref, physicalPath, updatedChange, siblings)}, nil
		}
	}

	return nil, fmt.Errorf("failed to find source location for %s", fullPath)
}

func appendRelativePath(base string, relative dyn.Path) (string, error) {
	path, err := structpath.ParsePattern(base)
	if err != nil {
		return "", err
	}
	for _, component := range relative {
		if key := component.Key(); key != "" {
			path = structpath.NewPatternStringKey(path, key)
		} else {
			path = structpath.NewPatternIndex(path, component.Index())
		}
	}
	return path.String(), nil
}

type valueChange struct {
	path      dyn.Path
	operation OperationType
	oldValue  any
	newValue  any
}

func diffValues(oldValue, newValue any) []valueChange {
	var result []valueChange
	diffValuesAt(oldValue, newValue, dyn.EmptyPath, &result)
	return result
}

func diffValuesAt(oldValue, newValue any, path dyn.Path, result *[]valueChange) {
	if reflect.DeepEqual(oldValue, newValue) {
		return
	}

	oldMap, oldIsMap := oldValue.(map[string]any)
	newMap, newIsMap := newValue.(map[string]any)
	if oldIsMap && newIsMap {
		keys := make(map[string]struct{}, len(oldMap)+len(newMap))
		for key := range oldMap {
			keys[key] = struct{}{}
		}
		for key := range newMap {
			keys[key] = struct{}{}
		}
		for _, key := range slices.Sorted(maps.Keys(keys)) {
			oldChild, oldOK := oldMap[key]
			newChild, newOK := newMap[key]
			childPath := path.Append(dyn.Key(key))
			switch {
			case oldOK && newOK:
				diffValuesAt(oldChild, newChild, childPath, result)
			case oldOK:
				*result = append(*result, valueChange{path: childPath, operation: OperationRemove, oldValue: oldChild})
			default:
				*result = append(*result, valueChange{path: childPath, operation: OperationAdd, newValue: newChild})
			}
		}
		return
	}

	oldSequence, oldIsSequence := oldValue.([]any)
	newSequence, newIsSequence := newValue.([]any)
	if oldIsSequence && newIsSequence && len(oldSequence) == len(newSequence) {
		for i := range oldSequence {
			diffValuesAt(oldSequence[i], newSequence[i], path.Append(dyn.Index(i)), result)
		}
		return
	}

	operation := OperationReplace
	if oldValue == nil {
		operation = OperationAdd
	} else if newValue == nil {
		operation = OperationRemove
	}
	*result = append(*result, valueChange{path: path, operation: operation, oldValue: oldValue, newValue: newValue})
}

type renamePair struct {
	removePath string
	key        string
	oldKey     string
	newKey     string
}

type unresolvedRenameGroup struct {
	removePaths []string
	addPaths    []string
}

func keyedElementPath(path string) (parent, key, value string, ok bool) {
	node, err := structpath.ParsePath(path)
	if err != nil {
		return "", "", "", false
	}
	key, value, ok = node.KeyValue()
	if !ok {
		return "", "", "", false
	}
	return node.Parent().String(), key, value, true
}

func valueWithoutKey(value any, key string) any {
	mapping, ok := value.(map[string]any)
	if !ok {
		return value
	}
	copy := make(map[string]any, len(mapping))
	for field, fieldValue := range mapping {
		if field != key {
			copy[field] = fieldValue
		}
	}
	return copy
}

func pairRenames(changes ResourceChanges) (map[string]renamePair, map[string]struct{}, []unresolvedRenameGroup) {
	type candidate struct {
		path   string
		key    string
		value  string
		change *ConfigChangeDesc
	}
	type group struct {
		removes []candidate
		adds    []candidate
	}

	groups := make(map[string]*group)
	for path, change := range changes {
		parent, key, value, ok := keyedElementPath(path)
		if !ok || (change.Operation != OperationRemove && change.Operation != OperationAdd) {
			continue
		}
		groupKey := parent + "\x00" + key
		g := groups[groupKey]
		if g == nil {
			g = &group{}
			groups[groupKey] = g
		}
		entry := candidate{path: path, key: key, value: value, change: change}
		if change.Operation == OperationRemove {
			g.removes = append(g.removes, entry)
		} else {
			g.adds = append(g.adds, entry)
		}
	}

	pairsByAdd := make(map[string]renamePair)
	pairedRemoves := make(map[string]struct{})
	var unresolved []unresolvedRenameGroup
	for _, groupKey := range slices.Sorted(maps.Keys(groups)) {
		g := groups[groupKey]
		slices.SortFunc(g.removes, func(a, b candidate) int { return cmp.Compare(a.path, b.path) })
		slices.SortFunc(g.adds, func(a, b candidate) int { return cmp.Compare(a.path, b.path) })
		usedRemoves := make([]bool, len(g.removes))
		usedAdds := make([]bool, len(g.adds))

		addPair := func(removeIndex, addIndex int) {
			remove := g.removes[removeIndex]
			add := g.adds[addIndex]
			pair := renamePair{
				removePath: remove.path,
				key:        remove.key,
				oldKey:     remove.value,
				newKey:     add.value,
			}
			pairsByAdd[add.path] = pair
			pairedRemoves[remove.path] = struct{}{}
			usedRemoves[removeIndex] = true
			usedAdds[addIndex] = true
		}

		for {
			paired := false
			for removeIndex, remove := range g.removes {
				if usedRemoves[removeIndex] {
					continue
				}
				matchingAdd := -1
				addMatches := 0
				for addIndex, add := range g.adds {
					if usedAdds[addIndex] {
						continue
					}
					if reflect.DeepEqual(
						valueWithoutKey(remove.change.configValue, remove.key),
						valueWithoutKey(add.change.Value, add.key),
					) {
						matchingAdd = addIndex
						addMatches++
					}
				}
				if addMatches != 1 {
					continue
				}

				removeMatches := 0
				add := g.adds[matchingAdd]
				for otherRemoveIndex, otherRemove := range g.removes {
					if usedRemoves[otherRemoveIndex] {
						continue
					}
					if reflect.DeepEqual(
						valueWithoutKey(otherRemove.change.configValue, otherRemove.key),
						valueWithoutKey(add.change.Value, add.key),
					) {
						removeMatches++
					}
				}
				if removeMatches != 1 {
					continue
				}
				addPair(removeIndex, matchingAdd)
				paired = true
			}
			if !paired {
				break
			}
		}

		unmatchedRemoves, unmatchedAdds := 0, 0
		for _, used := range usedRemoves {
			if !used {
				unmatchedRemoves++
			}
		}
		for _, used := range usedAdds {
			if !used {
				unmatchedAdds++
			}
		}
		if unmatchedRemoves > 0 && unmatchedAdds > 0 {
			group := unresolvedRenameGroup{}
			for index, remove := range g.removes {
				if !usedRemoves[index] {
					group.removePaths = append(group.removePaths, remove.path)
				}
			}
			for index, add := range g.adds {
				if !usedAdds[index] {
					group.addPaths = append(group.addPaths, add.path)
				}
			}
			unresolved = append(unresolved, group)
		}
	}

	return pairsByAdd, pairedRemoves, unresolved
}

func fieldChangeSequenceBlock(fieldChange FieldChange) (string, error) {
	path, err := structpath.ParsePattern(fieldChange.FieldCandidates[0])
	if err != nil {
		return "", err
	}
	if _, indexed := path.Index(); indexed || path.BracketStar() {
		path = path.Parent()
	}
	return fieldChange.FilePath + "\x00" + path.String(), nil
}

func validateUnresolvedRenameGroup(
	ctx context.Context,
	b *bundle.Bundle,
	sources *sourceIndex,
	resourceKey string,
	changes ResourceChanges,
	group unresolvedRenameGroup,
) error {
	blocks := make(map[string]struct{})
	paths := append(slices.Clone(group.removePaths), group.addPaths...)
	for _, path := range paths {
		fieldChanges, err := routeFieldChange(ctx, b, sources, resourceKey, path, changes[path])
		if err != nil {
			return err
		}
		if len(fieldChanges) == 0 {
			return fmt.Errorf("change %s has no source destination", path)
		}
		for _, fieldChange := range fieldChanges {
			block, err := fieldChangeSequenceBlock(fieldChange)
			if err != nil {
				return err
			}
			blocks[block] = struct{}{}
		}
	}
	if len(blocks) == 1 {
		return nil
	}

	parent, key, _, _ := keyedElementPath(group.removePaths[0])
	return fmt.Errorf(
		"cannot distinguish renames from independent additions and removals in %s by %s across different source blocks",
		parent,
		key,
	)
}

func routeRename(
	ctx context.Context,
	b *bundle.Bundle,
	sources *sourceIndex,
	resourceKey string,
	pair renamePair,
	removeChange, addChange *ConfigChangeDesc,
) ([]FieldChange, error) {
	oldFullPath := resourceKey + "." + pair.removePath
	selection, err := resolveMergedSelection(oldFullPath, b, OperationRemove)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve renamed element %s: %w", oldFullPath, err)
	}
	if !selection.value.IsValid() {
		return nil, fmt.Errorf("renamed element %s is not present in the configuration", oldFullPath)
	}

	refs := sources.refsFor(selection.value, selection.path, b.Config.Bundle.Target)
	if len(refs) == 0 {
		return nil, fmt.Errorf("failed to find source location for renamed element %s", oldFullPath)
	}
	if sourceRefsSpanScopes(refs, b.Config.Bundle.Target) {
		return nil, fmt.Errorf("cannot rename %s because it is defined in both top-level and target scopes", oldFullPath)
	}

	result := make([]FieldChange, 0, len(refs))
	for _, ref := range refs {
		keyPath := structpath.NewPatternStringKey(patternFromDynPath(ref.path), pair.key)
		keyChange := &ConfigChangeDesc{
			Operation:   OperationReplace,
			Value:       pair.newKey,
			configValue: pair.oldKey,
		}
		result = append(result, makeFieldChange(b, sources, ref, keyPath, keyChange, nil))
	}

	oldValue := removeChange.configValue
	if oldValue == nil {
		oldValue = selection.value.AsAny()
	}
	oldValue = valueWithoutKey(oldValue, pair.key)
	newValue := valueWithoutKey(addChange.Value, pair.key)
	for _, valueChange := range diffValues(oldValue, newValue) {
		if len(valueChange.path) == 0 {
			return nil, fmt.Errorf("renamed element %s changed to an incompatible value", oldFullPath)
		}
		syntheticPath, err := appendRelativePath(pair.removePath, valueChange.path)
		if err != nil {
			return nil, err
		}
		synthetic := &ConfigChangeDesc{
			Operation:   valueChange.operation,
			Value:       valueChange.newValue,
			configValue: valueChange.oldValue,
		}
		routed, err := routeChange(ctx, b, sources, resourceKey, syntheticPath, synthetic)
		if err != nil {
			return nil, err
		}
		result = append(result, routed...)
	}

	return result, nil
}

type sequencePair struct {
	oldIndex int
	newIndex int
	equal    bool
}

type sequencePlan struct {
	pairs    []sequencePair
	removals []int
	adds     []int
}

func positionalIdentityField(fieldPath string) string {
	path, err := structpath.ParsePath(fieldPath)
	if err != nil {
		return ""
	}
	key, ok := path.StringKey()
	if ok && key == "clusters" {
		return "label"
	}
	return ""
}

func positionalIdentity(value any, field string) (string, bool) {
	mapping, ok := value.(map[string]any)
	if !ok || field == "" {
		return "", false
	}
	identity, exists := mapping[field]
	if !exists || identity == nil {
		if field == "label" {
			return "default", true
		}
		return "", false
	}
	text, ok := identity.(string)
	if !ok {
		return "", false
	}
	if field == "label" {
		text = strings.ToLower(text)
	}
	return text, true
}

func planSequenceChanges(fieldPath string, oldValues, newValues []any) (sequencePlan, error) {
	identityField := positionalIdentityField(fieldPath)
	oldIdentities := make([]string, len(oldValues))
	newIdentities := make([]string, len(newValues))
	oldIdentityCount := make(map[string]int)
	newIdentityCount := make(map[string]int)
	for index, value := range oldValues {
		if identity, ok := positionalIdentity(value, identityField); ok {
			oldIdentities[index] = identity
			oldIdentityCount[identity]++
		}
	}
	for index, value := range newValues {
		if identity, ok := positionalIdentity(value, identityField); ok {
			newIdentities[index] = identity
			newIdentityCount[identity]++
		}
	}

	const impossible = int(^uint(0)>>1) / 4
	pairCost := func(oldIndex, newIndex int) int {
		if reflect.DeepEqual(oldValues[oldIndex], newValues[newIndex]) {
			return 0
		}
		if identityField == "" {
			return 1
		}
		identity := oldIdentities[oldIndex]
		if identity != "" && identity == newIdentities[newIndex] && oldIdentityCount[identity] == 1 && newIdentityCount[identity] == 1 {
			return 1
		}
		return impossible
	}

	type cell struct {
		cost int
		ways int
	}
	dp := make([][]cell, len(oldValues)+1)
	for index := range dp {
		dp[index] = make([]cell, len(newValues)+1)
		for other := range dp[index] {
			dp[index][other].cost = impossible
		}
	}
	dp[len(oldValues)][len(newValues)] = cell{ways: 1}
	addCandidate := func(best *cell, cost, ways int) {
		if ways == 0 || cost >= impossible {
			return
		}
		if cost < best.cost {
			best.cost = cost
			best.ways = min(ways, 2)
			return
		}
		if cost == best.cost {
			best.ways = min(best.ways+ways, 2)
		}
	}

	for oldIndex := len(oldValues); oldIndex >= 0; oldIndex-- {
		for newIndex := len(newValues); newIndex >= 0; newIndex-- {
			if oldIndex == len(oldValues) && newIndex == len(newValues) {
				continue
			}
			best := cell{cost: impossible}
			if oldIndex < len(oldValues) {
				next := dp[oldIndex+1][newIndex]
				addCandidate(&best, next.cost+1, next.ways)
			}
			if newIndex < len(newValues) {
				next := dp[oldIndex][newIndex+1]
				addCandidate(&best, next.cost+1, next.ways)
			}
			if oldIndex < len(oldValues) && newIndex < len(newValues) {
				next := dp[oldIndex+1][newIndex+1]
				addCandidate(&best, next.cost+pairCost(oldIndex, newIndex), next.ways)
			}
			dp[oldIndex][newIndex] = best
		}
	}

	if dp[0][0].ways != 1 {
		return sequencePlan{}, errors.New("sequence correspondence is ambiguous after its length changed")
	}

	plan := sequencePlan{}
	for oldIndex, newIndex := 0, 0; oldIndex < len(oldValues) || newIndex < len(newValues); {
		current := dp[oldIndex][newIndex]
		if oldIndex < len(oldValues) && newIndex < len(newValues) {
			next := dp[oldIndex+1][newIndex+1]
			if next.ways > 0 && current.cost == next.cost+pairCost(oldIndex, newIndex) {
				plan.pairs = append(plan.pairs, sequencePair{
					oldIndex: oldIndex,
					newIndex: newIndex,
					equal:    reflect.DeepEqual(oldValues[oldIndex], newValues[newIndex]),
				})
				oldIndex++
				newIndex++
				continue
			}
		}
		if oldIndex < len(oldValues) {
			next := dp[oldIndex+1][newIndex]
			if next.ways > 0 && current.cost == next.cost+1 {
				plan.removals = append(plan.removals, oldIndex)
				oldIndex++
				continue
			}
		}
		plan.adds = append(plan.adds, newIndex)
		newIndex++
	}

	return plan, nil
}

type sourceBlock struct {
	file   string
	parent dyn.Path
}

func blockForRef(ref sourceRef) sourceBlock {
	return sourceBlock{file: ref.file, parent: ref.path[:len(ref.path)-1]}
}

func sameBlock(a, b sourceBlock) bool {
	return a.file == b.file && a.parent.Equal(b.parent)
}

func chooseSequenceBlock(
	newIndex int,
	plan sequencePlan,
	elementRefs map[int]sourceRef,
	parentRefs []sourceRef,
) (sourceBlock, error) {
	previousOld := -1
	nextOld := len(elementRefs)
	for _, pair := range plan.pairs {
		if pair.newIndex < newIndex {
			previousOld = max(previousOld, pair.oldIndex)
		}
		if pair.newIndex > newIndex {
			nextOld = min(nextOld, pair.oldIndex)
		}
	}

	candidates := make(map[string]sourceBlock)
	addCandidate := func(ref sourceRef, element bool) {
		block := sourceBlock{file: ref.file, parent: ref.path}
		if element {
			block = blockForRef(ref)
		}
		candidates[block.file+"\x00"+block.parent.String()] = block
	}
	if previousOld >= 0 {
		addCandidate(elementRefs[previousOld], true)
	}
	if nextOld < len(elementRefs) {
		addCandidate(elementRefs[nextOld], true)
	}
	for oldIndex := previousOld + 1; oldIndex < nextOld; oldIndex++ {
		addCandidate(elementRefs[oldIndex], true)
	}
	if len(elementRefs) == 0 {
		for _, ref := range parentRefs {
			addCandidate(ref, false)
		}
	}

	if len(candidates) != 1 {
		return sourceBlock{}, errors.New("sequence insertion has no unique physical source block")
	}
	var result sourceBlock
	for _, block := range candidates {
		result = block
	}
	return result, nil
}

func expandSequenceReplacement(
	ctx context.Context,
	b *bundle.Bundle,
	sources *sourceIndex,
	resourceKey, fieldPath string,
	change *ConfigChangeDesc,
) ([]FieldChange, bool, error) {
	oldValues, oldOK := change.configValue.([]any)
	newValues, newOK := change.Value.([]any)
	if change.Operation != OperationReplace || !oldOK || !newOK {
		return nil, false, nil
	}

	fullPath := resourceKey + "." + fieldPath
	selection, err := resolveMergedSelection(fullPath, b, OperationReplace)
	if err != nil {
		return nil, true, err
	}
	mergedValues, ok := selection.value.AsSequence()
	if !ok || len(mergedValues) != len(oldValues) {
		return nil, true, fmt.Errorf("sequence %s does not match the current configuration", fullPath)
	}

	parentRefs := sources.refsFor(selection.value, selection.path, b.Config.Bundle.Target)
	if len(parentRefs) == 0 {
		return nil, false, nil
	}
	elementRefs := make(map[int]sourceRef, len(mergedValues))
	for index, value := range mergedValues {
		elementPath := structpath.NewPatternIndex(selection.path, index)
		refs := sources.refsFor(value, elementPath, b.Config.Bundle.Target)
		if len(refs) == 0 {
			return nil, true, fmt.Errorf("sequence element %s[%d] has no source location", fullPath, index)
		}
		elementRefs[index] = refs[0]
	}

	plan, err := planSequenceChanges(fieldPath, oldValues, newValues)
	if err != nil {
		return nil, true, fmt.Errorf("planning sequence changes for %s: %w", fullPath, err)
	}
	siblings := sourceSiblingsForSelection(sources, selection, b.Config.Bundle.Target)
	blocksByNew := make(map[int]sourceBlock, len(plan.pairs)+len(plan.adds))
	for _, pair := range plan.pairs {
		blocksByNew[pair.newIndex] = blockForRef(elementRefs[pair.oldIndex])
	}

	var result []FieldChange
	fieldPattern, err := structpath.ParsePattern(fieldPath)
	if err != nil {
		return nil, true, err
	}
	for _, pair := range plan.pairs {
		if pair.equal {
			continue
		}
		elementPath := structpath.NewPatternIndex(fieldPattern, pair.oldIndex).String()
		for _, valueChange := range diffValues(oldValues[pair.oldIndex], newValues[pair.newIndex]) {
			syntheticPath, err := appendRelativePath(elementPath, valueChange.path)
			if err != nil {
				return nil, true, err
			}
			synthetic := &ConfigChangeDesc{
				Operation:   valueChange.operation,
				Value:       valueChange.newValue,
				configValue: valueChange.oldValue,
			}
			routed, err := routeChange(ctx, b, sources, resourceKey, syntheticPath, synthetic)
			if err != nil {
				return nil, true, err
			}
			result = append(result, routed...)
		}
	}

	for _, oldIndex := range plan.removals {
		elementPath := structpath.NewPatternIndex(fieldPattern, oldIndex).String()
		synthetic := &ConfigChangeDesc{Operation: OperationRemove, configValue: oldValues[oldIndex]}
		routed, err := routeChange(ctx, b, sources, resourceKey, elementPath, synthetic)
		if err != nil {
			return nil, true, err
		}
		result = append(result, routed...)
	}

	for _, newIndex := range plan.adds {
		block, err := chooseSequenceBlock(newIndex, plan, elementRefs, parentRefs)
		if err != nil {
			return nil, true, fmt.Errorf("placing new element in %s: %w", fullPath, err)
		}
		physicalIndex := 0
		for index := range newIndex {
			if prior, ok := blocksByNew[index]; ok && sameBlock(prior, block) {
				physicalIndex++
			}
		}
		blocksByNew[newIndex] = block
		physicalPath := structpath.NewPatternIndex(patternFromDynPath(block.parent), physicalIndex)
		ref := sourceRef{file: block.file, path: block.parent}
		synthetic := &ConfigChangeDesc{Operation: OperationAdd, Value: newValues[newIndex]}
		physicalPath, synthetic = normalizeSequenceAdd(sources, ref, physicalPath, synthetic)
		result = append(result, makeFieldChange(b, sources, ref, physicalPath, synthetic, siblings))
	}

	return result, true, nil
}

func routeChange(
	ctx context.Context,
	b *bundle.Bundle,
	sources *sourceIndex,
	resourceKey, fieldPath string,
	change *ConfigChangeDesc,
) ([]FieldChange, error) {
	routed, expanded, err := expandSequenceReplacement(ctx, b, sources, resourceKey, fieldPath, change)
	if err != nil {
		return nil, err
	}
	if expanded {
		return routed, nil
	}
	return routeFieldChange(ctx, b, sources, resourceKey, fieldPath, change)
}

func sortFieldChanges(fieldChanges []FieldChange) {
	slices.SortStableFunc(fieldChanges, func(a, b FieldChange) int {
		pathA, errA := structpath.ParsePattern(a.FieldCandidates[0])
		pathB, errB := structpath.ParsePattern(b.FieldCandidates[0])
		if errA != nil || errB != nil {
			return cmp.Compare(a.FieldCandidates[0], b.FieldCandidates[0])
		}
		if pathA.Len() != pathB.Len() {
			return cmp.Compare(pathB.Len(), pathA.Len())
		}
		if a.FilePath == b.FilePath && pathA.Parent().String() == pathB.Parent().String() {
			indexA, indexedA := pathA.Index()
			indexB, indexedB := pathB.Index()
			if indexedA && indexedB {
				if a.Change.Operation == OperationRemove && b.Change.Operation == OperationRemove {
					return cmp.Compare(indexB, indexA)
				}
				if a.Change.Operation == OperationAdd && b.Change.Operation == OperationAdd {
					return cmp.Compare(indexA, indexB)
				}
			}
		}
		priority := func(operation OperationType) int {
			switch operation {
			case OperationReplace:
				return 0
			case OperationRemove:
				return 1
			case OperationAdd:
				return 2
			default:
				return 3
			}
		}
		if priorityA, priorityB := priority(a.Change.Operation), priority(b.Change.Operation); priorityA != priorityB {
			return cmp.Compare(priorityA, priorityB)
		}
		if a.FilePath != b.FilePath {
			return cmp.Compare(a.FilePath, b.FilePath)
		}
		return cmp.Compare(a.FieldCandidates[0], b.FieldCandidates[0])
	})
}

func coalesceSequenceElementAdds(fieldChanges []FieldChange) ([]FieldChange, error) {
	type destination struct {
		file string
		path string
	}

	result := make([]FieldChange, 0, len(fieldChanges))
	destinations := make(map[destination]int)
	for _, fieldChange := range fieldChanges {
		if !fieldChange.Change.sequenceElementAdd {
			result = append(result, fieldChange)
			continue
		}
		values, ok := fieldChange.Change.Value.([]any)
		if !ok {
			return nil, fmt.Errorf("new sequence elements for %s in %s are not a sequence", fieldChange.FieldCandidates[0], fieldChange.FilePath)
		}
		key := destination{file: fieldChange.FilePath, path: fieldChange.FieldCandidates[0]}
		index, exists := destinations[key]
		if !exists {
			copy := fieldChange
			change := *fieldChange.Change
			change.Value = slices.Clone(values)
			copy.Change = &change
			destinations[key] = len(result)
			result = append(result, copy)
			continue
		}

		existing := &result[index]
		existingValues, ok := existing.Change.Value.([]any)
		if !ok {
			return nil, fmt.Errorf("new sequence elements for %s in %s are not a sequence", existing.FieldCandidates[0], existing.FilePath)
		}
		change := *existing.Change
		change.Value = append(slices.Clone(existingValues), values...)
		existing.Change = &change
	}
	return result, nil
}

func resolveChangesWithSourceMapping(ctx context.Context, b *bundle.Bundle, configChanges Changes, sources *sourceIndex) ([]FieldChange, error) {
	var result []FieldChange
	for _, resourceKey := range slices.Sorted(maps.Keys(configChanges)) {
		resourceChanges := configChanges[resourceKey]
		pairsByAdd, pairedRemoves, unresolvedGroups := pairRenames(resourceChanges)
		for _, group := range unresolvedGroups {
			if err := validateUnresolvedRenameGroup(ctx, b, sources, resourceKey, resourceChanges, group); err != nil {
				return nil, fmt.Errorf("routing changes for %s: %w", resourceKey, err)
			}
		}

		fieldPaths := slices.Sorted(maps.Keys(resourceChanges))
		slices.SortStableFunc(fieldPaths, func(a, b string) int {
			depthA, depthB := pathDepth(a), pathDepth(b)
			if depthA != depthB {
				return cmp.Compare(depthB, depthA)
			}
			operationA, operationB := resourceChanges[a].Operation, resourceChanges[b].Operation
			if operationA == OperationRemove && operationB != OperationRemove {
				return -1
			}
			if operationA != OperationRemove && operationB == OperationRemove {
				return 1
			}
			return cmp.Compare(a, b)
		})

		for _, fieldPath := range fieldPaths {
			change := resourceChanges[fieldPath]
			if _, paired := pairedRemoves[fieldPath]; paired {
				continue
			}
			if pair, paired := pairsByAdd[fieldPath]; paired {
				routed, err := routeRename(ctx, b, sources, resourceKey, pair, resourceChanges[pair.removePath], change)
				if err != nil {
					return nil, err
				}
				result = append(result, routed...)
				continue
			}

			routed, err := routeChange(ctx, b, sources, resourceKey, fieldPath, change)
			if err != nil {
				return nil, err
			}
			result = append(result, routed...)
		}
	}

	result, err := coalesceSequenceElementAdds(result)
	if err != nil {
		return nil, err
	}
	sortFieldChanges(result)
	return result, nil
}
