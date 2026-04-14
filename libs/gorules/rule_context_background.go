package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoContextBackground detects context.Background() outside of main.go files.
func NoContextBackground(m dsl.Matcher) {
	m.Match(`context.Background()`).
		Where(!m.File().Name.Matches(`^main\.go$`)).
		Report(`Do not use context.Background(); use t.Context() in tests or pass context from caller`)
}
