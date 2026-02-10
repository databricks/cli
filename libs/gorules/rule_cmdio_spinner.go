package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoNewCmdioSpinner forbids new calls to cmdio.Spinner except in cmd/workspace/ and cmd/account/ directories.
// This rule ensures that spinner usage is limited to auto-generated workspace and account commands.
func NoNewCmdioSpinner(m dsl.Matcher) {
	m.Match(`cmdio.Spinner($*_)`).
		Where(!m.File().PkgPath.Matches(`.*/cmd/workspace/.*`) &&
			!m.File().PkgPath.Matches(`.*/cmd/account/.*`)).
		Report(`cmdio.Spinner is deprecated, no new call sites allowed. Use cmdio.NewSpinner() instead`)
}
