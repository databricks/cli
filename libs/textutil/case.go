package textutil

import "unicode"

// CamelToSnakeCase converts camelCase or PascalCase to snake_case. A run of
// capitals is kept together as one word, so "HTTPServer" becomes "http_server"
// rather than "h_t_t_p_server".
func CamelToSnakeCase(name string) string {
	runes := []rune(name)
	out := make([]rune, 0, len(runes)*2)
	for i, r := range runes {
		if i > 0 && unicode.IsUpper(r) {
			prev := runes[i-1]
			nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextIsLower) {
				out = append(out, '_')
			}
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
