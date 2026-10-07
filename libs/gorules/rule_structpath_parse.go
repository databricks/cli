package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoComputedStructpathParse forbids parsing computed structpath strings in production
// code. Parsing re-validates the string, can panic on user keys with special characters
// and costs allocations; build the path from parts with structpath.NewPath,
// NewPathSlice or NewPattern instead. Parsing string literals and parsing in tests is fine.
func NoComputedStructpathParse(m dsl.Matcher) {
	m.Match(`structpath.MustParsePath($s)`, `structpath.MustParsePattern($s)`).
		Where(!m["s"].Const && !m.File().Name.Matches(`_test\.go$`) && !m.File().PkgPath.Matches(`internal/bundletest`)).
		Report(`build computed paths with structpath.NewPath / NewPattern instead of parsing them`)

	m.Match(`structpath.MustParsePaths($*_, $s, $*_)`).
		Where(!m["s"].Const && !m.File().Name.Matches(`_test\.go$`) && !m.File().PkgPath.Matches(`internal/bundletest`)).
		Report(`build computed paths with structpath.NewPathSlice instead of parsing them`)

	m.Match(
		`structpath.ParsePath(fmt.Sprintf($*_))`,
		`structpath.ParsePath($a + $b)`,
		`structpath.ParsePattern(fmt.Sprintf($*_))`,
		`structpath.ParsePattern($a + $b)`,
	).
		Where(!m.File().Name.Matches(`_test\.go$`)).
		Report(`build the path with structpath.NewPath / NewPattern instead of formatting and parsing a string`)
}
