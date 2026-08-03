package configsync

import (
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/structs/structpath"
)

// renamedElement is a keyed sequence element whose key changed remotely.
type renamedElement struct {
	// keyField is the field holding the key, e.g. "task_key".
	keyField string
	oldKey   string
	newKey   string
	// addPath is the change path of the add half, kept so it can be skipped.
	addPath string
}

type renameSet struct {
	// byRemovePath maps the remove half's change path to the pair.
	byRemovePath map[string]renamedElement
	// addPaths are the add halves, which must not be routed on their own.
	addPaths map[string]struct{}
	// unpairedPaths are the halves of a suspected key change that could not be
	// matched up one-to-one. Applying them separately would delete the element from
	// every block and recreate it in one, collapsing a split element and moving
	// fields into a scope the user did not choose, so neither half is applied. The
	// value is the reason, for reporting.
	unpairedPaths map[string]string
}

// pairRenames matches removes of keyed elements against adds in the same sequence
// that carry the same content apart from the key.
//
// A remote key change is reported as an unrelated remove plus add, so without
// pairing the element is deleted and recreated: the recreated copy has to be
// placed somewhere, and for an element defined in several blocks there is no
// single right place. Recognising the pair turns it into a key rewrite, which
// every defining block can apply to its own part.
func pairRenames(b *bundle.Bundle, blocks *blockResolver, resourceKey string, changes ResourceChanges) renameSet {
	set := renameSet{
		byRemovePath:  map[string]renamedElement{},
		addPaths:      map[string]struct{}{},
		unpairedPaths: map[string]string{},
	}
	if blocks == nil {
		return set
	}

	removes, adds := keyedElementChanges(changes)

	// The remove half carries no value, so each removed element is read from the
	// merged configuration once: its fields decide which additions it could have
	// become, and its source blocks decide whether a choice between them is safe.
	candidates := make([]renameCandidate, 0, len(removes))
	for _, remove := range removes {
		resolved, err := resolveSelectors(resourceKey+"."+remove.path, b, OperationRemove)
		if err != nil || !resolved.leaf.IsValid() {
			continue
		}
		element, ok := resolved.leaf.AsAny().(map[string]any)
		if !ok {
			continue
		}
		candidates = append(candidates, renameCandidate{
			remove:    remove,
			resolved:  resolved,
			oldFields: withoutKey(element, remove.keyField),
		})
	}

	// Which additions each removal could equally well have become. Matching is by
	// content, so it is symmetric: an addition can be a match for several removals
	// just as a removal can match several additions.
	matches := make([][]keyedElement, len(candidates))
	for i, candidate := range candidates {
		for _, add := range adds {
			if candidate.remove.parent != add.parent || candidate.remove.keyField != add.keyField {
				continue
			}
			if sameElementApartFromKey(candidate.oldFields, add) {
				matches[i] = append(matches[i], add)
			}
		}
	}

	for i, candidate := range candidates {
		// A pair is only unambiguous when the choice is forced from both sides. One
		// removal matching several additions is the obvious case, but so is several
		// removals matching the same addition: pairing the first and deleting the
		// rest would move a key into a block the user did not choose.
		if len(matches[i]) == 1 && countMatchesOf(matches, matches[i][0].path) == 1 {
			add := matches[i][0]
			set.byRemovePath[candidate.remove.path] = renamedElement{
				keyField: candidate.remove.keyField,
				oldKey:   candidate.remove.key,
				newKey:   add.key,
				addPath:  add.path,
			}
			set.addPaths[add.path] = struct{}{}
			continue
		}

		if len(matches[i]) > 0 {
			// The pairing is not forced. Which pairing was intended only matters when
			// the elements live in different blocks, because then it decides which
			// block each new key is written to. Within one block every pairing writes
			// the same set of elements to the same place, so the halves can be applied
			// as a plain removal and addition instead of being held back -- otherwise a
			// rename of two same-bodied elements is dropped, and anything referring to
			// them by key (depends_on) keeps pointing at keys that no longer exist.
			if !ambiguityCrossesBlocks(blocks, candidates, matches, i) {
				continue
			}
			const reason = "several elements with the same contents were renamed in one run and they are not in the same block, so the new keys cannot be matched to them"
			set.unpairedPaths[candidate.remove.path] = reason
			for _, add := range matches[i] {
				set.unpairedPaths[add.path] = reason
			}
			continue
		}

		// The removal matched no addition. A plain removal is fine: it deletes every
		// part of the element, which is what the user asked for. But an addition of
		// the same element under a new key, with one of its fields also edited, cannot
		// be recognised as a rename: applying the halves separately would delete a
		// split element from every block and recreate it in one, collapsing the split
		// and moving fields into a scope the user did not choose. Hold both halves
		// back in that case only.
		//
		// "The same element with a field edited" means it still has the same fields;
		// a genuinely new element is an unrelated addition that has to be applied,
		// even when a removal happens to be in the same run.
		var edited []string
		for _, add := range adds {
			if candidate.remove.parent != add.parent || candidate.remove.keyField != add.keyField {
				continue
			}
			if _, taken := set.addPaths[add.path]; taken {
				continue
			}
			if sameFieldsApartFromKey(candidate.oldFields, add) {
				edited = append(edited, add.path)
			}
		}
		if len(edited) == 0 {
			continue
		}
		if !multiBlockElement(blocks, candidate.resolved) {
			continue
		}
		const reason = "a split element cannot be removed and recreated in one run"
		set.unpairedPaths[candidate.remove.path] = reason
		for _, path := range edited {
			set.unpairedPaths[path] = reason
		}
	}
	return set
}

// renameCandidate is a removal of a keyed element, resolved once so its fields and
// source blocks can be consulted without walking the tree again.
type renameCandidate struct {
	remove   keyedElement
	resolved resolvedChange
	// oldFields is the removed element without its key field, i.e. what an addition
	// has to look like to be the same element under a new key.
	oldFields map[string]any
}

// matchesAdd reports whether the addition at addPath is among these matches.
func matchesAdd(matches []keyedElement, addPath string) bool {
	return slices.ContainsFunc(matches, func(add keyedElement) bool { return add.path == addPath })
}

// countMatchesOf returns how many removals could have become the addition at
// addPath.
func countMatchesOf(matches [][]keyedElement, addPath string) int {
	count := 0
	for _, candidateMatches := range matches {
		if matchesAdd(candidateMatches, addPath) {
			count++
		}
	}
	return count
}

// ambiguityCrossesBlocks reports whether an unforced pairing for candidates[i]
// would have to choose between blocks. Every removal that could have become one of
// the same additions is considered, because the choice is between them.
func ambiguityCrossesBlocks(blocks *blockResolver, candidates []renameCandidate, matches [][]keyedElement, i int) bool {
	var seen []sourceBlock
	for j, candidate := range candidates {
		if j != i && !sharesAnyAdd(matches[i], matches[j]) {
			continue
		}
		if len(candidate.resolved.steps) == 0 {
			return true
		}
		last := candidate.resolved.steps[len(candidate.resolved.steps)-1]
		elementBlocks := blocks.blocksOf(last.element)
		if len(elementBlocks) != 1 {
			return true
		}
		if !slices.Contains(seen, elementBlocks[0]) {
			seen = append(seen, elementBlocks[0])
		}
	}
	return len(seen) != 1
}

// sharesAnyAdd reports whether two removals could have become the same addition.
func sharesAnyAdd(a, b []keyedElement) bool {
	return slices.ContainsFunc(a, func(add keyedElement) bool { return matchesAdd(b, add.path) })
}

// multiBlockElement reports whether the element the change addresses is assembled
// from more than one physical block.
func multiBlockElement(blocks *blockResolver, change resolvedChange) bool {
	if len(change.steps) == 0 {
		return false
	}
	last := change.steps[len(change.steps)-1]
	return len(blocks.blocksOf(last.element)) > 1
}

// keyedElement is one side of a candidate rename.
type keyedElement struct {
	path     string
	parent   string
	keyField string
	key      string
	value    any
}

// keyedElementChanges splits the changes that address a whole keyed element into
// removes and adds, in a deterministic order.
func keyedElementChanges(changes ResourceChanges) (removes, adds []keyedElement) {
	for _, path := range slices.Sorted(maps.Keys(changes)) {
		change := changes[path]
		if change.Operation != OperationRemove && change.Operation != OperationAdd {
			continue
		}
		node, err := structpath.ParsePath(path)
		if err != nil {
			continue
		}
		keyField, key, ok := node.KeyValue()
		if !ok {
			continue
		}
		element := keyedElement{
			path:     path,
			parent:   node.Parent().String(),
			keyField: keyField,
			key:      key,
			value:    change.Value,
		}
		if change.Operation == OperationRemove {
			removes = append(removes, element)
		} else {
			adds = append(adds, element)
		}
	}
	return removes, adds
}

// sameElementApartFromKey reports whether the added element is the removed one with
// a different key, i.e. a plain rename. oldFields is the removed element without its
// key field.
func sameElementApartFromKey(oldFields map[string]any, add keyedElement) bool {
	newFields, ok := fieldsApartFromKey(add)
	return ok && reflect.DeepEqual(oldFields, newFields)
}

// sameFieldsApartFromKey reports whether the added element has the same fields as the
// removed one, ignoring their values. A rename that also edited a field keeps the
// element's shape; an unrelated new element generally does not.
func sameFieldsApartFromKey(oldFields map[string]any, add keyedElement) bool {
	newFields, ok := fieldsApartFromKey(add)
	return ok && slices.Equal(slices.Sorted(maps.Keys(oldFields)), slices.Sorted(maps.Keys(newFields)))
}

// fieldsApartFromKey returns the added element's fields without its key field.
func fieldsApartFromKey(add keyedElement) (map[string]any, bool) {
	value, ok := add.value.(map[string]any)
	if !ok {
		return nil, false
	}
	return withoutKey(value, add.keyField), true
}

func withoutKey(value map[string]any, keyField string) map[string]any {
	out := make(map[string]any, len(value))
	for field, fieldValue := range value {
		if field != keyField {
			out[field] = fieldValue
		}
	}
	return out
}

// routeRenameElement locates the renamed element in every block that defines it.
// The caller turns each destination into a rewrite of the key field, so the
// element's other fields stay where they are and a split element keeps its parts
// in their original scopes.
func routeRenameElement(b *bundle.Bundle, blocks *blockResolver, resourceKey, removePath string) ([]routeDestination, error) {
	fullPath := resourceKey + "." + removePath
	resolved, err := resolveSelectors(fullPath, b, OperationRemove)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
	}

	destinations, err := blocks.routeDestinations(resolved)
	if err != nil {
		if errors.Is(err, errAmbiguousBlock) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to route rename %s: %w", fullPath, err)
	}
	return destinations, nil
}
