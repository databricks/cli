package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoComputedStructpathParse forbids parsing computed structpath strings in production
// code. Parsing re-validates the string, can panic on user keys with special characters
// and costs allocations; build the path from parts with structpath.NewPath,
// NewPathSlice or NewPattern instead. Parsing string literals and parsing in tests is fine.
// Values passed through a variable are not tracked.
func NoComputedStructpathParse(m dsl.Matcher) {
	m.Import("github.com/databricks/cli/libs/structs/structpath")
	m.Import("github.com/databricks/cli/bundle/config")

	m.Match(`structpath.MustParsePath($s)`, `structpath.MustParsePattern($s)`, `structpath.MustParsePaths($*s)`).
		Where(!m["s"].Const && !m.File().Name.Matches(`_test\.go$`) && !m.File().PkgPath.Matches(`internal/bundletest`)).
		Report(`build computed paths with structpath.NewPath / NewPathSlice / NewPattern instead of parsing them`)

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

	m.Match(
		`$c.GetLocations($a + $b)`,
		`$c.GetLocation($a + $b)`,
		`$c.DefinitionLocation($a + $b)`,
		`$c.GetLocations(fmt.Sprintf($*_))`,
		`$c.GetLocation(fmt.Sprintf($*_))`,
		`$c.DefinitionLocation(fmt.Sprintf($*_))`,
	).
		Where(!m.File().Name.Matches(`_test\.go$`) &&
			(m["c"].Type.Is("config.Root") || m["c"].Type.Is("*config.Root"))).
		Report(`pass a *structpath.PathNode to GetLocationsOf / GetLocationOf / DefinitionLocationOf instead of a built string`)

	m.Match(`$c.GetLocations($p.String())`, `$c.GetLocation($p.String())`, `$c.DefinitionLocation($p.String())`).
		Where(!m.File().Name.Matches(`_test\.go$`) &&
			(m["c"].Type.Is("config.Root") || m["c"].Type.Is("*config.Root")) &&
			m["p"].Type.Is("*structpath.PathNode")).
		Report(`pass the *structpath.PathNode to GetLocationsOf / GetLocationOf / DefinitionLocationOf instead of its string`)
}
