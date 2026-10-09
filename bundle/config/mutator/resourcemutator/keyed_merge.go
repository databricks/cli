package resourcemutator

import (
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
)

// mergeByKey merges, for every resource in resources (visited in sorted order), the sequence
// at resources.<resourceType>.<name>.<field> by key, skipping sequences for which the merge
// is a no-op (see [keyedMergeIsNoop]). The key of an element is noopKey(element), which must
// agree with keyFn applied to the key field of the element.
func mergeByKey[R, E any](
	b *bundle.Bundle,
	resourceType string,
	resources map[string]*R,
	field, keyField string,
	elems func(*R) []E,
	noopKey func(E) string,
	keyFn func(structvar.View) string,
	sortKeys bool,
) error {
	names := make([]string, 0, len(resources))
	for name := range resources {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		r := resources[name]
		if r == nil {
			continue
		}
		if keyedMergeIsNoop(b, "resources."+resourceType+"."+name+"."+field, elems(r), noopKey, sortKeys) {
			continue
		}
		path := structpath.NewPath(nil, "resources", resourceType, name, field)
		if err := b.Config.MergeElementsByKey(path, keyField, keyFn, sortKeys); err != nil {
			return err
		}
	}
	return nil
}

// stringKey returns a key function for sequence elements whose key is a string
// (an absent key is the empty string).
func stringKey(what string) func(structvar.View) string {
	return func(v structvar.View) string {
		switch v.Kind() {
		case structvar.KindInvalid, structvar.KindNil:
			return ""
		case structvar.KindString:
			s, _ := v.AsString()
			return s
		default:
			panic(what + " must be a string")
		}
	}
}

// keyedMergeIsNoop reports whether merging the elements of the sequence at path by key
// leaves it unchanged: the sequence is not a reference, and every key is non-empty and
// unique (and sorted, if the merge sorts by key). The merge operates on the dynamic
// configuration, which is built from the whole typed configuration, so mutators skip
// it when it would have no effect.
func keyedMergeIsNoop[T any](b *bundle.Bundle, path string, elems []T, key func(T) string, sorted bool) bool {
	if b.Config.IsReference(path) {
		return false
	}
	keys := make([]string, 0, len(elems))
	for _, e := range elems {
		k := key(e)
		if k == "" || slices.Contains(keys, k) {
			return false
		}
		keys = append(keys, k)
	}
	return !sorted || slices.IsSorted(keys)
}
