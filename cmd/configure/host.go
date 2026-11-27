package configure

import (
	"errors"
	"net/url"
	"strings"
)

// normalizeHost ensures a https:// scheme is present and returns only scheme
// and host, like the normalizeHost in libs/databrickscfg/host.go, which also
// drops the default port when matching profiles.
func normalizeHost(input string) string {
	input = strings.TrimSpace(input)
	u, err := url.Parse(input)
	if err != nil {
		return input
	}

	if u.Scheme != "https" && u.Scheme != "http" {
		u, err = url.Parse("https://" + input)
		if err != nil {
			return input
		}
	}

	if u.Host == "" {
		return input
	}

	return (&url.URL{Scheme: u.Scheme, Host: u.Host}).String()
}

func validateHost(s string) error {
	u, err := url.Parse(s)
	if err != nil {
		return err
	}
	if u.Host == "" || u.Scheme != "https" {
		return errors.New("must start with https://")
	}
	if u.Path != "" {
		return errors.New("must use empty path")
	}
	return nil
}
