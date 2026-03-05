package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// UseTestContext detects context.Background() in test files and suggests using t.Context().
func UseTestContext(m dsl.Matcher) {
	m.Match(`context.Background()`).
		Where(m.File().Name.Matches(`_test\.go$`)).
		Report(`Use t.Context() or b.Context() in tests instead of context.Background()`)
}
