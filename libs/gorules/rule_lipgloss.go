package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

// NoLipglossNewRenderer steers renderer construction through cmdio.NewRenderer,
// which decides color and its depth for the target writer
// (libs/cmdio/lipgloss.go). A bare lipgloss.NewRenderer bypasses that gate and
// reads color from process env, so it is only allowed inside libs/cmdio itself.
func NoLipglossNewRenderer(m dsl.Matcher) {
	m.Match(`lipgloss.NewRenderer($*_)`).
		Where(!m.File().PkgPath.Matches(`libs/cmdio$`)).
		Report(`Do not call lipgloss.NewRenderer directly; use cmdio.NewRenderer(ctx, w) so color handling stays centralized`)
}

// NoLipglossAdaptiveColor keeps light/dark selection explicit; lipgloss v2
// removes AdaptiveColor.
func NoLipglossAdaptiveColor(m dsl.Matcher) {
	m.Match(`lipgloss.AdaptiveColor{$*_}`).
		Report(`Do not use lipgloss.AdaptiveColor; select the variant with cmdio.LightDark(isDark)`)
}

// NoTermenvHyperlink routes OSC 8 links through cmdio.Hyperlink; lipgloss v2
// no longer depends on termenv.
func NoTermenvHyperlink(m dsl.Matcher) {
	m.Match(`termenv.Hyperlink($*_)`).
		Report(`Do not use termenv.Hyperlink; use cmdio.Hyperlink`)
}
