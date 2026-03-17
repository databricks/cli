package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoTimeNowUnixMilliInTestServer forbids direct time.Now().UnixMilli() calls in libs/testserver.
// Use nowMilli() instead to guarantee unique, strictly increasing timestamps.
// Integer millisecond timestamps get indexed replacements in test output (e.g. [UNIX_TIME_MILLIS][0])
// and collisions between resources cause flaky tests.
func NoTimeNowUnixMilliInTestServer(m dsl.Matcher) {
	m.Match(`time.Now().UnixMilli()`).
		Where(m.File().PkgPath.Matches(`.*/libs/testserver`) &&
			!m.File().Name.Matches(`fake_workspace\.go$`)).
		Report(`Use nowMilli() instead of time.Now().UnixMilli() in testserver to ensure unique timestamps`)
}
