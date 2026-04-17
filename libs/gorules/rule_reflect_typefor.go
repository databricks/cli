package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// UseReflectTypeFor detects reflect.TypeOf((*T)(nil)).Elem() and suggests reflect.TypeFor[T]() instead.
func UseReflectTypeFor(m dsl.Matcher) {
	m.Match(`reflect.TypeOf(($x)(nil)).Elem()`).
		Report(`Use reflect.TypeFor instead of reflect.TypeOf((*T)(nil)).Elem()`)
}
