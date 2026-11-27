package bitmap

import (
	"fmt"
	"strings"
)

// Text renders the bitmap as a "size=N context=C" header followed by one
// "PTF <path>" line per set field in schema order: "P" for a present field, "T"
// and "F" for a bool field seen holding true and false, and a space for each
// that is not set. A bool with "P" alone holds an unresolved variable
// reference; one with both "T" and "F" has both values under a wildcard. Unset
// fields are omitted. bits may be shorter than schema (a bitmap from an older
// CLI) or longer (a newer CLI): a set bit past the schema is a "P" line named
// by futureField.
func Text(schema []string, bits []bool, context uint16) []string {
	schema = schema[:min(len(bits), len(schema))]
	index := indexSchema(schema)
	lines := []string{fmt.Sprintf("size=%d context=%d", len(bits), context)}
	for i, p := range schema {
		if strings.HasSuffix(p, trueSuffix) || strings.HasSuffix(p, falseSuffix) {
			continue
		}
		isTrue := isSet(index, bits, p+trueSuffix)
		isFalse := isSet(index, bits, p+falseSuffix)
		if !bits[i] && !isTrue && !isFalse {
			continue
		}
		lines = append(lines, fmt.Sprintf("%c%c%c %s", letter(bits[i], 'P'), letter(isTrue, 'T'), letter(isFalse, 'F'), p))
	}
	for i := len(schema); i < len(bits); i++ {
		if bits[i] {
			lines = append(lines, "P   "+futureField(i))
		}
	}
	return lines
}

// futureField names a set bit past the end of the schema, a field that only a
// newer CLI can name. Whether it is a field or the value of a bool is unknown.
func futureField(bit int) string {
	return fmt.Sprintf("future #%d", bit)
}

// letter returns c if set, and a space otherwise, so that Text's flags stay aligned.
func letter(set bool, c byte) byte {
	if set {
		return c
	}
	return ' '
}

// JSON is the JSON rendering of a bitmap. Presence, True and False are trees
// keyed by path segment ("*" for any map key, and a list field's name plus
// "[*]" for its elements) with true leaves: Presence holds every set field,
// True and False the bool fields seen with that value. A field with set
// children is a subtree rather than a leaf, since a child is only set when its
// parent is; likewise "tasks[*]" replaces the leaf of "tasks".
type JSON struct {
	Size     int            `json:"size"`
	Context  uint16         `json:"context"`
	Presence map[string]any `json:"presence"`
	True     map[string]any `json:"true"`
	False    map[string]any `json:"false"`
}

// ToJSON renders bits as JSON. bits may be shorter than schema or longer: a set
// bit past the schema is a top-level Presence entry named by futureField.
func ToJSON(schema []string, bits []bool, context uint16) JSON {
	out := JSON{
		Size:     len(bits),
		Context:  context,
		Presence: map[string]any{},
		True:     map[string]any{},
		False:    map[string]any{},
	}
	known := min(len(bits), len(schema))
	for i, p := range schema[:known] {
		if !bits[i] {
			continue
		}
		tree := out.Presence
		if field, ok := strings.CutSuffix(p, trueSuffix); ok {
			tree, p = out.True, field
		} else if field, ok := strings.CutSuffix(p, falseSuffix); ok {
			tree, p = out.False, field
		}
		insertPath(tree, strings.Split(p, "."))
	}
	for i := known; i < len(bits); i++ {
		if bits[i] {
			out.Presence[futureField(i)] = true
		}
	}
	return out
}

// insertPath adds a true leaf at keys, turning a leaf on the way into a
// subtree. A "tasks[*]" key replaces the "tasks" leaf next to it, and a "tasks"
// leaf is not added next to "tasks[*]".
func insertPath(tree map[string]any, keys []string) {
	for i, k := range keys {
		if field, ok := strings.CutSuffix(k, listSuffix); ok {
			delete(tree, field)
		}
		if i == len(keys)-1 {
			if _, ok := tree[k]; !ok && tree[k+listSuffix] == nil {
				tree[k] = true
			}
			return
		}
		child, ok := tree[k].(map[string]any)
		if !ok {
			child = map[string]any{}
			tree[k] = child
		}
		tree = child
	}
}
