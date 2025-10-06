package configure

import (
	"errors"
	"net/url"
	"strings"
)

// normalizeHost normalizes host input to prevent double https:// prefixes.
// If the input already starts with https://, it returns it as-is.
// If the input doesn't start with https://, it prepends https://.
func normalizeHost(input string) string {
	input = strings.TrimSpace(input)
	u, err := url.Parse(input)
	// If the input is not a valid URL, return it as-is
	if err != nil {
		return input
	}

	// If it already starts with https:// or http://, return as-is
	if u.Scheme == "https" || u.Scheme == "http" {
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
