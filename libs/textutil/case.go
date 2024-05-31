package textutil

import "unicode"

func CamelToSnakeCase(name string) string {
	var out []rune = make([]rune, 0, len(name)*2)
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) {
			out = append(out, '_')
		}
		out = append(out, unicode.ToLower(r))
	}
	return string(out)
}
