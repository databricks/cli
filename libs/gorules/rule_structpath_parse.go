package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoComputedStructpathParse forbids building structpath paths and patterns by parsing
// strings in production code. Parsing re-validates the string, can panic on user keys
// with special characters and costs allocations; build the path from parts with
// structpath.NewPath, NewPathSlice or NewPattern instead. ParsePath / ParsePattern with
// the error handled remain for strings that are data (user input, map keys). Tests may
// parse. Values passed through a variable are not tracked.
func NoComputedStructpathParse(m dsl.Matcher) {
	m.Import("github.com/databricks/cli/libs/structs/structpath")

	m.Match(`structpath.MustParsePath($*_)`, `structpath.MustParsePattern($*_)`, `structpath.MustParsePaths($*_)`).
		Where(!m.File().Name.Matches(`_test\.go$`) && !m.File().PkgPath.Matches(`internal/bundletest`)).
		Report(`build paths with structpath.NewPath / NewPathSlice / NewPattern instead of parsing them`)

	m.Match(
		`structpath.ParsePath(fmt.Sprintf($*_))`,
		`structpath.ParsePath($a + $b)`,
		`structpath.ParsePattern(fmt.Sprintf($*_))`,
		`structpath.ParsePattern($a + $b)`,
	).
		Where(!m.File().Name.Matches(`_test\.go$`)).
		Report(`build the path with structpath.NewPath / NewPattern instead of formatting and parsing a string`)

	// Re-parsing a node as its own kind is a no-op; parsing a pattern as a path is fine
	// (it rejects wildcards).
	m.Match(`structpath.ParsePath($p.String())`).
		Where(!m.File().Name.Matches(`_test\.go$`) && m["p"].Type.Is("*structpath.PathNode")).
		Report(`use the structpath node instead of rendering and parsing it`)
	m.Match(`structpath.ParsePattern($p.String())`).
		Where(!m.File().Name.Matches(`_test\.go$`) && m["p"].Type.Is("*structpath.PatternNode")).
		Report(`use the structpath node instead of rendering and parsing it`)

	// Nested single-step constructors are hard to read; NewPath and NewPattern take the
	// parts in order (with AnyKey / AnyIndex for wildcards).
	m.Match(`structpath.$f(structpath.$g($*_), $*_)`).
		Where(m["f"].Text.Matches(`^NewPattern(StringKey|DotString|BracketString|Index|DotStar|BracketStar|KeyValue)$`) &&
			m["g"].Text.Matches(`^(NewPattern|MustParsePattern)`) && !m.File().PkgPath.Matches(`libs/structs/structpath$`)).
		Report(`build the pattern with structpath.NewPattern(prev, parts...) instead of nesting constructors`)
	m.Match(`structpath.$f(structpath.$g($*_), $*_)`).
		Where(m["f"].Text.Matches(`^New(StringKey|DotString|BracketString|Index|KeyValue)$`) &&
			m["g"].Text.Matches(`^(New(StringKey|DotString|BracketString|Index|KeyValue|Path)|MustParsePath)$`) && !m.File().PkgPath.Matches(`libs/structs/structpath$`)).
		Report(`build the path with structpath.NewPath(prev, parts...) instead of nesting constructors`)
}
