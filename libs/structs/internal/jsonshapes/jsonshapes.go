// Package jsonshapes is a corpus of struct shapes whose JSON behaviour is easy to get
// wrong, shared by the tests of the libs/structs packages.
//
// Each shape pairs a value with the fields encoding/json actually serializes for it, so a
// package can be checked against the wire format without restating the reasoning. The
// interesting shapes are all about embedding: how deep a promoted field may sit, which of
// two same-named fields wins, and when encoding/json gives up and serializes neither.
package jsonshapes

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Shape is one struct whose JSON behaviour a libs/structs package must match.
type Shape struct {
	// Name identifies the shape in test output.
	Name string

	// Value is a pointer to a populated struct.
	Value any

	// JSONFields are the json names encoding/json emits for Value, in any order. They are
	// asserted against json.Marshal by TestCorpusMatchesEncodingJSON, so a shape whose
	// expectation drifts from reality fails in the corpus itself rather than silently
	// teaching every consumer the wrong thing.
	JSONFields []string

	// TypeFields are the json names a type-level walk should yield, which differs from
	// JSONFields only where a value cannot reach what its type declares: an embedded nil
	// pointer contributes its fields to the type but nothing to the wire. Empty means
	// "same as JSONFields".
	TypeFields []string

	// WalkGap, WalkTypeGap and DiffGap record what a package does today where it disagrees
	// with encoding/json. They hold the exact current output, not merely a "this is broken"
	// flag, so any change to the behaviour -- including a different wrong answer -- fails the
	// package's test and forces the entry to be revisited. Nil means "must agree".
	WalkGap     []string
	WalkTypeGap []string
	DiffGap     []string

	// KnownSetGap are json names encoding/json can unmarshal into but structaccess.Set
	// cannot reach today. Consumers assert that Set still fails for them, so fixing the gap
	// breaks the test and the entry has to go -- a ratchet rather than a silent exemption.
	KnownSetGap []string

	// Unreachable are json names that look available on the Go type -- some field declares
	// them -- but that encoding/json does not serialize, so no package may expose them
	// either. Ambiguous embedded fields land here.
	Unreachable []string
}

type Leaf struct {
	Value string `json:"value,omitempty"`
}

type MiddleWithLeaf struct {
	Leaf
}

// DoublyEmbedded reaches a field through two levels of embedding, the shape every postgres
// resource has (PostgresProject -> PostgresProjectConfig -> postgres.ProjectSpec).
type DoublyEmbedded struct {
	MiddleWithLeaf

	Own string `json:"own,omitempty"`
}

type ShallowValue struct {
	Value string `json:"value,omitempty"`
}

type DeepHolder struct {
	Leaf
}

// ShallowWins declares value at two depths. encoding/json takes the shallower one, so Get
// and Set must resolve to the same field the wire format uses.
type ShallowWins struct {
	ShallowValue
	DeepHolder
}

type SideA struct {
	Value string `json:"value,omitempty"`
}

type SideB struct {
	Value string `json:"value,omitempty"`
}

// SameDepthConflict declares value twice at one depth. encoding/json calls that ambiguous
// and emits neither, so the field is not readable or writable either.
type SameDepthConflict struct {
	SideA
	SideB //nolint:govet // the repeated json tag is the point: both embeds declare "value"
}

type DiamondLeft struct {
	Leaf
}

type DiamondRight struct {
	Leaf
}

// Diamond reaches one Leaf by two routes of equal length: ambiguous, like SameDepthConflict.
type Diamond struct {
	DiamondLeft
	DiamondRight //nolint:govet // the repeated json tag is the point: both routes reach Leaf
}

type NilSide struct {
	Value string `json:"value,omitempty"`
}

// AmbiguousViaNilPointer declares value twice at one depth, with one side behind a nil
// pointer. encoding/json resolves fields from the type, so it is ambiguous either way and
// neither is serialized -- a value-level search that skips the nil side sees only one
// declaration and wrongly concludes the field is reachable.
type AmbiguousViaNilPointer struct {
	*NilSide
	SideB //nolint:govet // the repeated json tag is the point: both embeds declare "value"
}

// Cyclic embeds a pointer to itself, so a type-level search that does not remember where it
// has been never terminates. The corpus leaves the pointer nil: encoding/json flattens an
// embedded type once and stops, but a value walk following a self-referential pointer is
// genuinely unbounded, so a populated cycle is not a shape the packages can be compared on.
// structaccess exercises the populated case in its own test.
type Cyclic struct {
	*Cyclic

	Name string `json:"name,omitempty"`
}

// NilPointerEmbed embeds a pointer left nil. Whether a path resolves is a property of the
// type, so it must not depend on the pointer being set.
type NilPointerEmbed struct {
	*Leaf

	Own string `json:"own,omitempty"`
}

// SetPointerEmbed is NilPointerEmbed with the pointer populated.
type SetPointerEmbed struct {
	*Leaf

	Own string `json:"own,omitempty"`
}

// TaggedEmbed carries a json name on an anonymous field, which makes it an ordinary field to
// encoding/json: it serializes as a nested object under that name rather than being flattened.
type TaggedEmbed struct {
	Leaf `json:"leaf"`

	Own string `json:"own,omitempty"`
}

// OptionOnlyEmbed sets an option on the embed's tag without giving it a name, which leaves it
// flattened -- the presence of a tag is not what decides.
type OptionOnlyEmbed struct {
	Leaf `json:",omitempty"`

	Own string `json:"own,omitempty"`
}

// SkippedField declares a field encoding/json never serializes.
type SkippedField struct {
	Kept    string `json:"kept,omitempty"`
	Skipped string `json:"-"`

	unexported string //nolint:unused // present to prove it is ignored
}

// Fields returns the names a type-level walk should yield for the shape.
func (s Shape) Fields() []string {
	if len(s.TypeFields) > 0 {
		return s.TypeFields
	}
	return s.JSONFields
}

// Shapes returns the corpus. Each call builds fresh values so a test may mutate them.
func Shapes() []Shape {
	return []Shape{
		{
			Name:       "doubly embedded",
			Value:      &DoublyEmbedded{MiddleWithLeaf: MiddleWithLeaf{Leaf: Leaf{Value: "v"}}, Own: "o"},
			JSONFields: []string{"value", "own"},
		},
		{
			Name:       "shallower embed wins",
			Value:      &ShallowWins{ShallowValue: ShallowValue{Value: "shallow"}, DeepHolder: DeepHolder{Leaf: Leaf{Value: "deep"}}},
			JSONFields: []string{"value"},
			// Both walks visit each declaration, so a shadowed field is reported twice while the
			// wire format carries one value.
			WalkGap:     []string{"value", "value"},
			WalkTypeGap: []string{"value", "value"},
		},
		{
			Name:        "same depth conflict",
			Value:       &SameDepthConflict{SideA: SideA{Value: "a"}, SideB: SideB{Value: "b"}},
			JSONFields:  nil,
			Unreachable: []string{"value"},
			// structaccess reports the name as not found, as encoding/json does. The walks still
			// visit both declarations and structdiff still reports a change at the path, so the
			// engine can plan an update for a field that cannot be serialized.
			WalkGap:     []string{"value", "value"},
			WalkTypeGap: []string{"value", "value"},
			// Once per declaration, since structdiff walks both.
			DiffGap: []string{"value", "value"},
		},
		{
			Name:        "diamond",
			Value:       &Diamond{DiamondLeft: DiamondLeft{Leaf: Leaf{Value: "l"}}, DiamondRight: DiamondRight{Leaf: Leaf{Value: "r"}}},
			JSONFields:  nil,
			Unreachable: []string{"value"},
			WalkGap:     []string{"value", "value"},
			WalkTypeGap: []string{"value", "value"},
			// Once per declaration, since structdiff walks both.
			DiffGap: []string{"value", "value"},
		},
		{
			Name:        "ambiguous via nil pointer",
			Value:       &AmbiguousViaNilPointer{SideB: SideB{Value: "b"}}, //exhaustruct:ignore
			JSONFields:  nil,
			Unreachable: []string{"value"},
			// The value walk reaches only the non-nil declaration; the type walk sees both.
			WalkGap:     []string{"value"},
			WalkTypeGap: []string{"value", "value"},
			// Once per declaration, since structdiff walks both.
			// Once: the nil side is not reachable in the value, so only one declaration is walked.
			DiffGap: []string{"value"},
		},
		{
			Name:       "cyclic embed",
			Value:      &Cyclic{Name: "n"}, //exhaustruct:ignore
			JSONFields: []string{"name"},
			// The embedded *Cyclic promotes name a level down, where encoding/json shadows it
			// with the outer one. The type walk reports both; the value walk agrees, because the
			// corpus leaves the pointer nil.
			WalkTypeGap: []string{"name", "name"},
		},
		{
			Name:       "nil pointer embed",
			Value:      &NilPointerEmbed{Own: "o"}, //exhaustruct:ignore
			JSONFields: []string{"own"},
			TypeFields: []string{"value", "own"},
		},
		{
			Name:       "set pointer embed",
			Value:      &SetPointerEmbed{Leaf: &Leaf{Value: "v"}, Own: "o"},
			JSONFields: []string{"value", "own"},
			// json.Unmarshal allocates the embedded pointer to reach value; Set refuses to
			// descend through a nil one, so a fresh value cannot be written through. Fixing it
			// means allocating only once the write is known to succeed, or a failed Set leaves
			// an allocated embed behind and changes what the type marshals to.
			KnownSetGap: []string{"value"},
		},
		{
			Name:       "tagged embed is a named field",
			Value:      &TaggedEmbed{Leaf: Leaf{Value: "v"}, Own: "o"},
			JSONFields: []string{"leaf.value", "own"},
			// Not flattened, so the outer object has no "value" member.
			Unreachable: []string{"value"},
		},
		{
			Name:       "option-only embed stays flattened",
			Value:      &OptionOnlyEmbed{Leaf: Leaf{Value: "v"}, Own: "o"},
			JSONFields: []string{"value", "own"},
		},
		{
			Name:        "skipped field",
			Value:       &SkippedField{Kept: "k", Skipped: "s"}, //exhaustruct:ignore
			JSONFields:  []string{"kept"},
			Unreachable: []string{"-"},
		},
	}
}

// Leaves marshals v and returns its scalar leaves keyed by path, so a shape's JSONFields can
// be compared against what encoding/json actually produces even when a shape nests.
//
// Object members are joined with a dot, which is the struct-field rendering. That is enough
// here because no shape in the corpus contains a map; the type-aware version, which has to
// tell a map entry's ['key'] from a field's .name, lives in bundle/config/structstest.
func Leaves(v any) (map[string]string, error) {
	blob, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(blob, &generic); err != nil {
		return nil, err
	}
	out := map[string]string{}
	flattenLeaves("", generic, out)
	return out, nil
}

func flattenLeaves(prefix string, v any, out map[string]string) {
	switch value := v.(type) {
	case map[string]any:
		for key, member := range value {
			if prefix != "" {
				key = prefix + "." + key
			}
			flattenLeaves(key, member, out)
		}
	case []any:
		for i, member := range value {
			flattenLeaves(prefix+"["+strconv.Itoa(i)+"]", member, out)
		}
	case nil:
		// A JSON null carries no scalar leaf.
	default:
		out[prefix] = fmt.Sprintf("%v", value)
	}
}
