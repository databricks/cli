package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoOsChdir forbids os.Chdir in test files. Use t.Chdir() instead.
func NoOsChdir(m dsl.Matcher) {
	m.Match(`os.Chdir($*_)`).
		Where(m.File().Name.Matches(`_test\.go$`)).
		Report(`Use t.Chdir() instead of os.Chdir() in tests`)
}
