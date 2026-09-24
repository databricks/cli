//go:build windows

package ssh

import (
	"errors"
	"strings"
	"unicode"
)

func proxyCommandArgumentsForTest(command string) ([]string, error) {
	var args []string
	var arg strings.Builder
	inArg := false
	inQuotes := false
	backslashes := 0
	flush := func() {
		if inArg {
			args = append(args, arg.String())
			arg.Reset()
			inArg = false
		}
	}
	for _, r := range command {
		switch {
		case r == '\\':
			backslashes++
			inArg = true
			continue
		case r == '"':
			arg.WriteString(strings.Repeat("\\", backslashes/2))
			if backslashes%2 == 1 {
				arg.WriteRune('"')
			} else {
				inQuotes = !inQuotes
			}
			backslashes = 0
			inArg = true
			continue
		case unicode.IsSpace(r) && !inQuotes:
			arg.WriteString(strings.Repeat("\\", backslashes))
			backslashes = 0
			flush()
		default:
			arg.WriteString(strings.Repeat("\\", backslashes))
			backslashes = 0
			arg.WriteRune(r)
			inArg = true
		}
	}
	if inQuotes {
		return nil, errors.New("unterminated ProxyCommand argument")
	}
	arg.WriteString(strings.Repeat("\\", backslashes))
	flush()
	return args, nil
}
