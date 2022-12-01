package interpolation

import "reflect"

// String values in maps are not addressable and therefore not settable
// through Go's reflection mechanism. This interface solves this limitation
// by wrapping the setter differently for addressable values and map values.
type setter interface {
	Set(string)
}

type nilSetter struct{}

func (nilSetter) Set(_ string) {
	panic("nil setter")
}

type anySetter struct {
	rv reflect.Value
}

func (s anySetter) Set(str string) {
	s.rv.SetString(str)
}

type mapSetter struct {
	// map[string]string
	m reflect.Value

	// key
	k reflect.Value
}

func (s mapSetter) Set(str string) {
	s.m.SetMapIndex(s.k, reflect.ValueOf(str))
}

type getter interface {
	Get() string
}

type anyGetter struct {
	rv reflect.Value
}

func (g anyGetter) Get() string {
	return g.rv.String()
}
