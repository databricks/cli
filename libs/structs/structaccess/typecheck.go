package structaccess

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// ValidateByString reports whether the given path string is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// This is a convenience function that parses the path string and calls ValidatePattern.
// Wildcards are allowed in the path.
func ValidateByString(t reflect.Type, path string) error {
	if path == "" {
		return nil
	}

	patternNode, err := structpath.ParsePattern(path)
	if err != nil {
		return err
	}

	return ValidatePattern(t, patternNode)
}

// ValidatePath reports whether the given path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// Paths cannot contain wildcards.
func ValidatePath(t reflect.Type, path *structpath.PathNode) error {
	// PathNode is type definition of PatternNode, so we can cast directly
	return ValidatePattern(t, (*structpath.PatternNode)(path))
}

// ValidatePattern reports whether the given pattern path is valid for the provided type.
// It returns nil if the path resolves fully, or an error indicating where resolution failed.
// Patterns may include wildcards ([*] and .*).
func ValidatePattern(t reflect.Type, path *structpath.PatternNode) error {
	if path.IsRoot() {
		return nil
	}
	return validateNodeSlice(t, path.AsSlice())
}

// validateNodeSlice is the implementation for ValidatePattern.
func validateNodeSlice(t reflect.Type, nodes []*structpath.PatternNode) error {
	cur := t
	for _, node := range nodes {
		// Always dereference pointers at the type level.
		for cur.Kind() == reflect.Pointer {
			cur = cur.Elem()
		}

		// Index access: slice/array
		if idx, isIndex := node.Index(); isIndex {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				// Terraform represents single-block fields as lists and uses [0] to access them.
				// Treat [0] on a struct as a no-op so TF-style paths work against DABs structs.
				if idx == 0 && kind == reflect.Struct {
					continue
				}
				return fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		// Handle wildcards - treat like index/key access
		if node.BracketStar() {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot use [*] on %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}
		if node.DotStar() {
			if cur.Kind() != reflect.Map {
				return fmt.Errorf("%s: cannot use .* on %s", node.String(), cur.Kind())
			}
			cur = cur.Elem()
			continue
		}

		// Handle key-value selector: validates that we can index the slice/array
		if _, _, isKeyValue := node.KeyValue(); isKeyValue {
			if cur.Kind() == reflect.Struct {
				if embedType := findEmbedFieldType(cur); embedType != nil {
					cur = embedType
				}
			}
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return fmt.Errorf("%s: cannot use key-value syntax on %s", node.String(), kind)
			}
			cur = cur.Elem()
			continue
		}

		key, ok := node.StringKey()

		if !ok {
			return errors.New("unsupported path node type")
		}

		switch cur.Kind() {
		case reflect.Struct:
			sf, _, ok := FindStructFieldByKeyType(cur, key)
			if !ok {
				return fmt.Errorf("%s: field %q not found in %s", node.String(), key, cur.String())
			}
			cur = sf.Type
		case reflect.Map:
			kt := cur.Key()
			if kt.Kind() != reflect.String {
				return fmt.Errorf("%s: map key must be string, got %s", node.String(), kt)
			}
			cur = cur.Elem()
		default:
			return fmt.Errorf("%s: cannot access key %q on %s", node.String(), key, cur.Kind())
		}
	}

	return nil
}

// FindStructFieldByKeyType searches exported fields of struct type t for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (pointer or value) recursively.
// Returns the StructField, the declaring owner type, and whether it was found.
func FindStructFieldByKeyType(t reflect.Type, key string) (reflect.StructField, reflect.Type, bool) {
	index, sf, ok := findFieldIndexByKeyType(t, key)
	if !ok {
		return reflect.StructField{}, reflect.TypeOf(nil), false
	}
	return sf, ownerTypeAt(t, index), true
}

// findFieldIndexByKeyType resolves key to a field of t and returns the chain of field indices
// leading to it, the way reflect.Type.FieldByName does.
//
// Embedded structs are searched breadth-first, mirroring encoding/json: a name declared at
// two embedding depths resolves to the shallower one, so a depth-first search could pick a
// field the wire format does not use. A name declared twice at one depth is ambiguous, which
// encoding/json resolves by serializing neither, so it resolves to nothing here too.
//
// Returning the index chain rather than a type matters: the same struct type can be reachable
// by more than one path, so a caller navigating a value needs the path json would take, not
// merely the type at the end of it.
func findFieldIndexByKeyType(t reflect.Type, key string) ([]int, reflect.StructField, bool) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, reflect.StructField{}, false
	}

	if c, ok := pickCandidate(directCandidates(t, key, nil)); ok {
		return c.index, c.field, true
	}

	// A cycle must not be walked twice, or a key the type never declares sends the search
	// round forever. Only types from *earlier* levels are excluded: a type reachable twice
	// within one level is a diamond, and its two matches are the ambiguity json omits.
	seen := map[reflect.Type]bool{t: true}
	level := embeddedIndexPaths(t, nil)
	for len(level) > 0 {
		var next []embeddedPath
		var found []candidate
		for _, embed := range level {
			matches := directCandidates(embed.typ, key, embed.index)
			if len(matches) > 0 {
				found = append(found, matches...)
				continue
			}
			for _, deeper := range embeddedIndexPaths(embed.typ, embed.index) {
				if seen[deeper.typ] {
					continue
				}
				next = append(next, deeper)
			}
		}
		for _, embed := range next {
			seen[embed.typ] = true
		}
		if len(found) > 0 {
			if c, ok := pickCandidate(found); ok {
				return c.index, c.field, true
			}
			return nil, reflect.StructField{}, false
		}
		level = next
	}

	return nil, reflect.StructField{}, false
}

// embeddedPath is an embedded struct type together with the index chain that reaches it.
type embeddedPath struct {
	typ   reflect.Type
	index []int
}

// embeddedIndexPaths returns the embeds of t that encoding/json flattens, each with the index
// chain from the root that reaches it.
func embeddedIndexPaths(t reflect.Type, prefix []int) []embeddedPath {
	var out []embeddedPath
	for i := range t.NumField() {
		sf := t.Field(i)
		if !IsFlattenedEmbed(sf) {
			continue
		}
		ft := sf.Type
		for ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if ft.Kind() != reflect.Struct {
			continue
		}
		out = append(out, embeddedPath{typ: ft, index: append(append([]int{}, prefix...), i)})
	}
	return out
}

// ownerTypeAt returns the struct type that declares the field the index chain ends at.
func ownerTypeAt(t reflect.Type, index []int) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	for _, i := range index[:len(index)-1] {
		t = t.Field(i).Type
		for t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
	}
	return t
}

// candidate is a field that matches a json name, with the index chain reaching it and whether
// the name came from a json tag. encoding/json prefers a tagged name over an untagged one at
// the same depth, so the distinction has to survive the search.
type candidate struct {
	index  []int
	field  reflect.StructField
	tagged bool
}

// directCandidates returns the struct's own fields that key can name. A field with a json tag
// name is matched on that; a field without one is matched on its Go field name, which is what
// encoding/json serializes it under. An embed that encoding/json flattens is not addressable by
// name at all, so it is not a candidate.
func directCandidates(t reflect.Type, key string, prefix []int) []candidate {
	var out []candidate
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		if sf.Name == EmbeddedSliceFieldName || IsFlattenedEmbed(sf) {
			continue
		}
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" {
			continue
		}
		tagged := name != ""
		if !tagged {
			name = sf.Name
		}
		if name != key {
			continue
		}
		// Skip fields marked as internal/readonly
		btag := structtag.BundleTag(sf.Tag.Get("bundle"))
		if btag.Internal() || btag.ReadOnly() {
			continue
		}
		out = append(out, candidate{
			index:  append(append([]int{}, prefix...), i),
			field:  sf,
			tagged: tagged,
		})
	}
	return out
}

// pickCandidate applies encoding/json's precedence among fields that share a name at one
// depth: a single tagged name wins over untagged ones, a single match of either kind wins, and
// anything else is ambiguous and serialized as nothing.
func pickCandidate(candidates []candidate) (candidate, bool) {
	if len(candidates) == 1 {
		return candidates[0], true
	}
	var tagged []candidate
	for _, c := range candidates {
		if c.tagged {
			tagged = append(tagged, c)
		}
	}
	if len(tagged) == 1 {
		return tagged[0], true
	}
	return candidate{}, false
}

// IsFlattenedEmbed reports whether the field is an embed encoding/json flattens into the
// outer object. An anonymous field that carries a json *name* is a named field instead: it
// serializes as a nested object under that name. The name is what matters, not the presence
// of a tag: `json:",omitempty"` leaves the name empty, so such a field is still flattened.
func IsFlattenedEmbed(sf reflect.StructField) bool {
	if !sf.Anonymous {
		return false
	}
	if structtag.JSONTag(sf.Tag.Get("json")).Name() != "" {
		return false
	}
	// Only an anonymous *struct* is promoted. An embedded scalar, slice or interface is a member
	// named after its type, so it belongs at its own path rather than the parent's.
	ft := sf.Type
	for ft.Kind() == reflect.Pointer {
		ft = ft.Elem()
	}
	return ft.Kind() == reflect.Struct
}
