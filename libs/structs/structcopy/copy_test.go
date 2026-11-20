package structcopy_test

import (
	"testing"

	"github.com/databricks/cli/libs/structs/structcopy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures mirror how the SDK shapes one resource: a response, a create request the
// state embeds, and an update request whose path parameter reuses a body field's name.

type (
	remoteKind string
	createKind string
)

type remoteThing struct {
	Comment   string            `json:"comment,omitempty"`
	CreatedAt int64             `json:"created_at,omitempty"`
	Kind      remoteKind        `json:"kind,omitempty"`
	Name      string            `json:"name,omitempty"`
	Size      int               `json:"size,omitempty"`
	Tags      map[string]string `json:"tags,omitempty"`
	ThingId   string            `json:"thing_id,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

type createThing struct {
	Comment        string            `json:"comment,omitempty"`
	Kind           createKind        `json:"kind,omitempty"`
	Name           string            `json:"name,omitempty"`
	Size           int               `json:"size,omitempty"`
	SkipValidation bool              `json:"skip_validation,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

type thingState struct {
	createThing

	ThingId string `json:"thing_id,omitempty"`
}

type updateThing struct {
	Comment string `json:"comment,omitempty"`
	// Name is the path parameter that addresses the thing, not the body field of createThing.
	Name    string            `json:"-" url:"-"`
	NewName string            `json:"new_name,omitempty"`
	Owner   string            `json:"owner,omitempty"`
	Size    int               `json:"size,omitempty"`
	Tags    map[string]string `json:"tags,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func TestCopySharedFields(t *testing.T) {
	src := &remoteThing{
		Comment:   "a comment",
		CreatedAt: 1700000000,
		Kind:      "LARGE",
		Name:      "thing",
		Size:      3,
		Tags:      map[string]string{"team": "data"},
		ThingId:   "id-1",
	}

	got := structcopy.New[remoteThing, createThing]().Copy(src)

	assert.Equal(t, &createThing{
		Comment: "a comment",
		Kind:    "LARGE",
		Name:    "thing",
		Size:    3,
		Tags:    map[string]string{"team": "data"},
	}, got)
}

func TestCopyKeepsForceSendFieldsOfCopiedFields(t *testing.T) {
	src := &remoteThing{
		// CreatedAt and ThingId have no counterpart in createThing; the SDK marshaller
		// rejects a ForceSendFields name its struct does not have.
		ForceSendFields: []string{"Comment", "CreatedAt", "Size", "ThingId"},
	}

	got := structcopy.New[remoteThing, createThing]().Copy(src)

	assert.Equal(t, []string{"Comment", "Size"}, got.ForceSendFields)
}

func TestCopyWithoutForcedFieldsLeavesForceSendFieldsNil(t *testing.T) {
	got := structcopy.New[remoteThing, createThing]().Copy(&remoteThing{Name: "thing"})

	assert.Nil(t, got.ForceSendFields)
}

func TestCopyDoesNotShareForceSendFieldsWithSource(t *testing.T) {
	src := &createThing{ForceSendFields: []string{"Comment", "Size"}}

	got := structcopy.New[createThing, createThing]().Copy(src)
	require.Equal(t, []string{"Comment", "Size"}, got.ForceSendFields)
	got.ForceSendFields[0] = "Name"

	assert.Equal(t, []string{"Comment", "Size"}, src.ForceSendFields)
}

func TestCopyIntoEmbeddedStruct(t *testing.T) {
	src := &remoteThing{
		Name:            "thing",
		ThingId:         "id-1",
		ForceSendFields: []string{"Size", "ThingId"},
	}

	got := structcopy.New[remoteThing, thingState]().Copy(src)

	assert.Equal(t, &thingState{
		Name: "thing",
		// ThingId is declared by thingState, which has no ForceSendFields of its own.
		ForceSendFields: []string{"Size"},
		ThingId:         "id-1",
	}, got)
}

func TestCopyFromEmbeddedStruct(t *testing.T) {
	src := &thingState{
		Comment:         "a comment",
		Tags:            map[string]string{"team": "data"},
		ForceSendFields: []string{"Size"},
		ThingId:         "id-1",
	}

	got := structcopy.New[thingState, updateThing](structcopy.Skip("Name")).Copy(src)

	assert.Equal(t, &updateThing{
		Comment:         "a comment",
		Tags:            map[string]string{"team": "data"},
		ForceSendFields: []string{"Size"},
	}, got)
}

type stateWithOwnForceSend struct {
	createThing

	ThingId string `json:"thing_id,omitempty"`

	ForceSendFields []string `json:"-" url:"-"`
}

func TestCopyKeepsForceSendFieldsPerStruct(t *testing.T) {
	src := &remoteThing{ForceSendFields: []string{"ThingId", "Size"}}

	got := structcopy.New[remoteThing, stateWithOwnForceSend]().Copy(src)

	// A ForceSendFields names fields of the struct that declares it, not of one it embeds.
	assert.Equal(t, []string{"Size"}, got.createThing.ForceSendFields)
	assert.Equal(t, []string{"ThingId"}, got.ForceSendFields)
}

func TestForceSendOnPromotedFieldUsesForceSendFieldsOfItsStruct(t *testing.T) {
	copier := structcopy.New[remoteThing, stateWithOwnForceSend](structcopy.ForceSend("ThingId", "Comment"))

	got := copier.Copy(&remoteThing{})

	assert.Equal(t, []string{"Comment"}, got.createThing.ForceSendFields)
	assert.Equal(t, []string{"ThingId"}, got.ForceSendFields)
}

type withUnexportedField struct {
	Name   string `json:"name,omitempty"`
	secret string
}

func TestCopyIgnoresUnexportedFields(t *testing.T) {
	src := &withUnexportedField{Name: "thing", secret: "hunter2"}

	got := structcopy.New[withUnexportedField, withUnexportedField]().Copy(src)

	assert.Equal(t, &withUnexportedField{Name: "thing"}, got)
}

type shadowingState struct {
	createThing

	// Name hides createThing.Name, the way a state type overrides a field of the request it embeds.
	Name string `json:"name"`
}

func TestCopyUsesFieldThatShadowsEmbeddedOne(t *testing.T) {
	got := structcopy.New[remoteThing, shadowingState]().Copy(&remoteThing{Name: "thing"})

	assert.Equal(t, "thing", got.Name)
	assert.Empty(t, got.createThing.Name)
}

func TestSkipLeavesFieldUnset(t *testing.T) {
	src := &createThing{
		Comment:         "a comment",
		Name:            "thing",
		ForceSendFields: []string{"Comment", "Name"},
	}

	got := structcopy.New[createThing, updateThing](structcopy.Skip("Name", "Comment")).Copy(src)

	assert.Empty(t, got.Name)
	assert.Empty(t, got.Comment)
	// A skipped field carries no value, so there is nothing to force-send.
	assert.Nil(t, got.ForceSendFields)
}

func TestForceSendListsFieldOnEveryCopy(t *testing.T) {
	copier := structcopy.New[createThing, updateThing](
		structcopy.Skip("Name"),
		structcopy.ForceSend("Comment", "Owner"),
	)

	got := copier.Copy(&createThing{})
	assert.Equal(t, []string{"Comment", "Owner"}, got.ForceSendFields)

	got = copier.Copy(&createThing{ForceSendFields: []string{"Size", "Comment"}})
	assert.Equal(t, []string{"Comment", "Owner", "Size"}, got.ForceSendFields)
}

func TestForceSendDoesNotAccumulateAcrossCopies(t *testing.T) {
	copier := structcopy.New[createThing, updateThing](
		structcopy.Skip("Name"),
		structcopy.ForceSend("Comment"),
	)

	copier.Copy(&createThing{ForceSendFields: []string{"Size"}})
	got := copier.Copy(&createThing{})

	assert.Equal(t, []string{"Comment"}, got.ForceSendFields)
}

func TestDescribe(t *testing.T) {
	copier := structcopy.New[thingState, updateThing](
		structcopy.Skip("Name"),
		structcopy.ForceSend("Comment"),
	)

	got, err := copier.Describe()

	require.NoError(t, err)
	assert.Equal(t, `structcopy_test.thingState -> structcopy_test.updateThing
  Comment: copied, always sent
  Kind: not in destination
  Name: skipped
  NewName: not in source
  Owner: not in source
  Size: copied
  SkipValidation: not in destination
  Tags: copied
  ThingId: not in destination
`, got)
}

func TestDescribeNamesConvertedType(t *testing.T) {
	got, err := structcopy.New[remoteThing, createThing]().Describe()

	require.NoError(t, err)
	assert.Contains(t, got, "  Kind: copied, converted from structcopy_test.remoteKind\n")
}

type (
	sizeAsString struct {
		Size string `json:"size,omitempty"`
	}
	sizeRenamedOnWire struct {
		Size int `json:"capacity,omitempty"`
	}
	tagsWithoutForceSend struct {
		Tags map[string]string `json:"tags,omitempty"`
	}
	embeddedPointer struct {
		*createThing
	}
	getThing struct {
		Name string `json:"-" url:"-"`
	}
)

func TestInvalidCopier(t *testing.T) {
	tests := []struct {
		name     string
		describe func() (string, error)
		copy     func()
		want     string
	}{
		{
			name:     "source is not a struct",
			describe: structcopy.New[string, createThing]().Describe,
			copy:     func() { structcopy.New[string, createThing]().Copy(new(string)) },
			want:     "structcopy: string -> structcopy_test.createThing: string is not a struct",
		},
		{
			name:     "destination is not a struct",
			describe: structcopy.New[createThing, int]().Describe,
			copy:     func() { structcopy.New[createThing, int]().Copy(&createThing{}) },
			want:     "structcopy: structcopy_test.createThing -> int: int is not a struct",
		},
		{
			name:     "field has another type in the destination",
			describe: structcopy.New[createThing, sizeAsString]().Describe,
			copy:     func() { structcopy.New[createThing, sizeAsString]().Copy(&createThing{}) },
			want:     "structcopy: structcopy_test.createThing -> structcopy_test.sizeAsString: Size is int in the source but string in the destination; skip it and set it by hand",
		},
		{
			name:     "field has another JSON name in the destination",
			describe: structcopy.New[createThing, sizeRenamedOnWire]().Describe,
			copy:     func() { structcopy.New[createThing, sizeRenamedOnWire]().Copy(&createThing{}) },
			want:     `structcopy: structcopy_test.createThing -> structcopy_test.sizeRenamedOnWire: Size is "size" in the source's JSON but "capacity" in the destination's; skip it and set it by hand`,
		},
		{
			name:     "field is a path parameter in the destination",
			describe: structcopy.New[createThing, updateThing]().Describe,
			copy:     func() { structcopy.New[createThing, updateThing]().Copy(&createThing{}) },
			want:     `structcopy: structcopy_test.createThing -> structcopy_test.updateThing: Name is not in the destination's JSON (json:"-"); skip it and set it by hand`,
		},
		{
			name:     "field is a path parameter in both",
			describe: structcopy.New[getThing, updateThing]().Describe,
			copy:     func() { structcopy.New[getThing, updateThing]().Copy(&getThing{}) },
			want:     `structcopy: structcopy_test.getThing -> structcopy_test.updateThing: Name is not in the destination's JSON (json:"-"); skip it and set it by hand`,
		},
		{
			name:     "skipped field is not in the destination",
			describe: structcopy.New[createThing, updateThing](structcopy.Skip("Name", "Kind")).Describe,
			copy:     func() { structcopy.New[createThing, updateThing](structcopy.Skip("Name", "Kind")).Copy(&createThing{}) },
			want:     "structcopy: structcopy_test.createThing -> structcopy_test.updateThing: cannot skip Kind: the destination has no such field",
		},
		{
			name:     "skipped field is not in the source",
			describe: structcopy.New[createThing, updateThing](structcopy.Skip("Name", "Owner")).Describe,
			copy: func() {
				structcopy.New[createThing, updateThing](structcopy.Skip("Name", "Owner")).Copy(&createThing{})
			},
			want: "structcopy: structcopy_test.createThing -> structcopy_test.updateThing: cannot skip Owner: the source has no such field, so it is not copied anyway",
		},
		{
			name:     "force-sent field is not in the destination",
			describe: structcopy.New[remoteThing, createThing](structcopy.ForceSend("Owner")).Describe,
			copy:     func() { structcopy.New[remoteThing, createThing](structcopy.ForceSend("Owner")).Copy(&remoteThing{}) },
			want:     "structcopy: structcopy_test.remoteThing -> structcopy_test.createThing: cannot force-send Owner: the destination has no such field",
		},
		{
			name:     "force-sent field is not a basic type",
			describe: structcopy.New[remoteThing, createThing](structcopy.ForceSend("Tags")).Describe,
			copy:     func() { structcopy.New[remoteThing, createThing](structcopy.ForceSend("Tags")).Copy(&remoteThing{}) },
			want:     "structcopy: structcopy_test.remoteThing -> structcopy_test.createThing: cannot force-send Tags: ForceSendFields has no effect on a map",
		},
		{
			name:     "force-sent field has no ForceSendFields next to it",
			describe: structcopy.New[remoteThing, thingState](structcopy.ForceSend("ThingId")).Describe,
			copy:     func() { structcopy.New[remoteThing, thingState](structcopy.ForceSend("ThingId")).Copy(&remoteThing{}) },
			want:     "structcopy: structcopy_test.remoteThing -> structcopy_test.thingState: cannot force-send ThingId: the struct that declares it has no ForceSendFields",
		},
		{
			name:     "source embeds a pointer",
			describe: structcopy.New[embeddedPointer, createThing]().Describe,
			copy:     func() { structcopy.New[embeddedPointer, createThing]().Copy(&embeddedPointer{}) },
			want:     "structcopy: structcopy_test.embeddedPointer -> structcopy_test.createThing: structcopy_test.embeddedPointer embeds *structcopy_test.createThing; embedded pointers are not supported",
		},
		{
			name:     "destination embeds a pointer",
			describe: structcopy.New[createThing, embeddedPointer]().Describe,
			copy:     func() { structcopy.New[createThing, embeddedPointer]().Copy(&createThing{}) },
			want:     "structcopy: structcopy_test.createThing -> structcopy_test.embeddedPointer: structcopy_test.embeddedPointer embeds *structcopy_test.createThing; embedded pointers are not supported",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.describe()
			require.EqualError(t, err, tt.want)

			assert.PanicsWithError(t, tt.want, tt.copy)
		})
	}
}

func TestSkipAllowsFieldsThatCannotBeCopied(t *testing.T) {
	got := structcopy.New[createThing, sizeAsString](structcopy.Skip("Size")).Copy(&createThing{Size: 3})

	assert.Empty(t, got.Size)
}

func TestCopyIntoStructWithoutForceSendFields(t *testing.T) {
	src := &createThing{
		Tags:            map[string]string{"team": "data"},
		ForceSendFields: []string{"Tags"},
	}

	got := structcopy.New[createThing, tagsWithoutForceSend]().Copy(src)

	assert.Equal(t, &tagsWithoutForceSend{Tags: map[string]string{"team": "data"}}, got)
}
