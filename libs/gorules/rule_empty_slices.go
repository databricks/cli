package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// DeclaringEmptySlices detects empty slice declarations and suggests using nil slices.
// See: https://go.dev/wiki/CodeReviewComments#declaring-empty-slices
func DeclaringEmptySlices(m dsl.Matcher) {
	m.Match(`$x := []$t{}`).
		Report(`Consider using 'var $x []$t' to declare a nil slice instead of an empty slice literal`).
		Suggest(`var $x []$t`)

	m.Match(`$x := make([]$t, 0)`).
		Report(`Consider using 'var $x []$t' to declare a nil slice instead of using make([]$t, 0)`).
		Suggest(`var $x []$t`)
}
