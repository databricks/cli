package structwalk

type Simple struct {
	X int
}

type Types struct {
	ValidField           string `json:"valid_field"`
	ValidFieldNoTag      string
	IgnoredField         string `json:"-"`
	IgnoredFieldOdd      string `json:"-,omitempty"`
	EmptyTagField        string `json:""`
	unexportedField      string `json:"unexported"` //nolint
	unexportedFieldNoTag string //nolint

	IntField  int
	BoolField bool `json:bool_field` // nolint (bad syntax)
	AnyField  any

	ValidFieldPtr           *string `json:"valid_field_ptr"`
	ValidFieldPtrNoTag      *string
	IgnoredFieldPtr         *string `json:"-"`
	IgnoredFieldOddPtr      *string `json:"-,omitempty"` //nolint
	EmptyTagFieldPtr        *string `json:""`
	unexportedFieldPtr      *string `json:"unexported"` //nolint
	unexportedFieldNoTagPtr *string //nolint

	SliceString []string
	ArrayString [2]string

	Nested    Simple
	NestedPtr *Simple
	Slice     []Simple
	Array     [2]Simple
	Map       map[string]Simple
	MapPtr    map[string]*Simple
	MapIntKey map[int]*Simple

	// Fields with omitempty to test ForceSendFields behaviour
	OmitStr  string `json:"omit_str,omitempty"`
	OmitInt  int    `json:"omit_int,omitempty"`
	OmitBool bool   `json:"omit_bool,omitempty"`

	FuncField func() string `json:"-"`
	ChanField chan string   `json:"-"`

	// List of field names to be force-sent even if they hold zero values.
	ForceSendFields []string `json:"-"`
}

type SelfIndirect struct {
	X *Self
}

type Self struct {
	ValidField string `json:"valid_field"`

	SelfReference *Self

	SelfSlice    []Self
	SelfSlicePtr []*Self

	SelfArrayPtr [5]*Self

	SelfMap    map[string]Self
	SelfMapPtr map[string]*Self

	SelfIndirect    SelfIndirect
	SelfIndirectPtr *SelfIndirect

	// List of field names to be force-sent even if they hold zero values.
	ForceSendFields []string `json:"-"`
}
