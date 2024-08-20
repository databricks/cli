package test_types

// Recursive types cannot be defined inline without making them anonymous,
// so we define them here instead.
type Foo struct {
	Bar *Bar `json:"bar,omitempty"`
}

type Bar struct {
	Foo Foo `json:"foo,omitempty"`
}

type Outer struct {
	Foo Foo `json:"foo"`
}
