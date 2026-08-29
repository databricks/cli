package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// UseBenchmarkLoop detects classic benchmark loops over b.N and suggests
// b.Loop() (Go 1.24+; the inlining regression was fixed in 1.26 so there's
// no longer a reason to keep b.N-based loops).
func UseBenchmarkLoop(m dsl.Matcher) {
	m.Match(
		`for $_ := range $b.N`,
		`for range $b.N`,
	).
		Where(m["b"].Type.Is("*testing.B")).
		Report(`Use 'for $b.Loop()' instead of looping over $b.N (Go 1.24+, performance-correct in 1.26+)`)
}
