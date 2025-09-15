package configure

import (
	"errors"
	"net/url"
	"strings"
)

// NormalizeHost normalizes host input to prevent double https:// prefixes.
// If the input already starts with https://, it returns it as-is.
// If the input doesn't start with https://, it prepends https://.
func NormalizeHost(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "https://"
	}

	// If it already starts with https://, return as-is
	if strings.HasPrefix(strings.ToLower(input), "https://") {
		return input
	}

	// Otherwise, prepend https://
	return "https://" + input
}

func validateHost(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Host == "" || u.Scheme != "https" {
		return errors.New("must start with https://")
	}
	if u.Path != "" && u.Path != "/" {
		return errors.New("must use empty path")
	}
	return nil
}
