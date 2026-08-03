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
	for _, remove := range removes {
		// Every add the removed element could equally well have become. Two
		// identically-bodied elements renamed in one run produce two removes that
		// each match both adds, and the pairing decides which key goes to which
		// element: getting it backwards moves a key into the wrong block, and
		// anything referring to the element by key (depends_on) then points at the
		// wrong one. Nothing in the change set says which pairing was intended, so
		// hold every half back rather than pick one.
		var matches []keyedElement
		for _, add := range adds {
			if _, taken := set.addPaths[add.path]; taken {
				continue
			}
			if remove.parent != add.parent || remove.keyField != add.keyField {
				continue
			}
			if sameElementApartFromKey(b, resourceKey, remove, add) {
				matches = append(matches, add)
			}
		}
		if len(matches) == 1 {
			add := matches[0]
			set.byRemovePath[remove.path] = renamedElement{
				keyField: remove.keyField,
				oldKey:   remove.key,
				newKey:   add.key,
				addPath:  add.path,
			}
			set.addPaths[add.path] = struct{}{}
			continue
		}
		if len(matches) > 1 {
			const reason = "several elements with the same contents were renamed in one run, so the new keys cannot be matched to them"
			set.unpairedPaths[remove.path] = reason
			for _, add := range matches {
				set.unpairedPaths[add.path] = reason
			}
			continue
		}

		// The removal was not matched to an addition. A plain removal is fine: it
		// deletes every part of the element, which is what the user asked for.
		// But an addition of the same element under a new key, with one of its
		// fields also edited, cannot be recognised as a rename: applying the halves
		// separately would delete a split element from every block and recreate it
		// in one, collapsing the split and moving fields into a scope the user did
		// not choose. Hold both halves back in that case only.
		//
		// "The same element with a field edited" means it still has the same fields;
		// a genuinely new element is an unrelated addition that has to be applied,
		// even when a removal happens to be in the same run.
		var candidates []string
		for _, add := range adds {
			if remove.parent != add.parent || remove.keyField != add.keyField {
				continue
			}
			if _, taken := set.addPaths[add.path]; taken {
				continue
			}
			if sameFieldsApartFromKey(b, resourceKey, remove, add) {
				candidates = append(candidates, add.path)
			}
		}
		if len(candidates) == 0 {
			continue
		}
		if !multiBlockElement(b, blocks, resourceKey, remove.path) {
			continue
		}
		const reason = "a split element cannot be removed and recreated in one run"
		set.unpairedPaths[remove.path] = reason
		for _, path := range candidates {
			set.unpairedPaths[path] = reason
		}
	}
	return set
}

// multiBlockElement reports whether the element at path is assembled from more
// than one physical block.
func multiBlockElement(b *bundle.Bundle, blocks *blockResolver, resourceKey, path string) bool {
	resolved, err := resolveSelectors(resourceKey+"."+path, b, OperationRemove)
	if err != nil || len(resolved.steps) == 0 {
		return false
	}
	last := resolved.steps[len(resolved.steps)-1]
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

// sameElementApartFromKey reports whether the added element is the removed one
// with a different key. The remove half carries no value, so the old element is
// read from the merged configuration.
func sameElementApartFromKey(b *bundle.Bundle, resourceKey string, remove, add keyedElement) bool {
	resolved, err := resolveSelectors(resourceKey+"."+remove.path, b, OperationRemove)
	if err != nil || !resolved.leaf.IsValid() {
		return false
	}
	oldValue, ok := resolved.leaf.AsAny().(map[string]any)
	if !ok {
		return false
	}
	newValue, ok := add.value.(map[string]any)
	if !ok {
		return false
	}
	return reflect.DeepEqual(withoutKey(oldValue, remove.keyField), withoutKey(newValue, add.keyField))
}

// sameFieldsApartFromKey reports whether the added element has the same fields as
// the removed one, ignoring their values. A rename that also edited a field keeps
// the element's shape; an unrelated new element generally does not.
func sameFieldsApartFromKey(b *bundle.Bundle, resourceKey string, remove, add keyedElement) bool {
	resolved, err := resolveSelectors(resourceKey+"."+remove.path, b, OperationRemove)
	if err != nil || !resolved.leaf.IsValid() {
		return false
	}
	oldValue, ok := resolved.leaf.AsAny().(map[string]any)
	if !ok {
		return false
	}
	newValue, ok := add.value.(map[string]any)
	if !ok {
		return false
	}
	return slices.Equal(
		slices.Sorted(maps.Keys(withoutKey(oldValue, remove.keyField))),
		slices.Sorted(maps.Keys(withoutKey(newValue, add.keyField))),
	)
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
