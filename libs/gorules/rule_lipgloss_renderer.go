package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoLipglossNewRenderer steers renderer construction through cmdio.NewRenderer,
// which forces the Ascii profile when the target writer has no color. A bare
// lipgloss.NewRenderer bypasses that gate and reads color from process env, so
// it is only allowed inside libs/cmdio itself (where the wrapper lives).
func NoLipglossNewRenderer(m dsl.Matcher) {
	m.Match(`lipgloss.NewRenderer($*_)`).
		Where(!m.File().PkgPath.Matches(`libs/cmdio$`)).
		Report(`Do not call lipgloss.NewRenderer directly; use cmdio.NewRenderer(ctx, w) so color handling stays centralized`)
}
