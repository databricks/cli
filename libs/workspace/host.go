package workspace

import (
	"net/url"
	"strings"
)

// NormalizeHost returns the string representation of only
// the scheme and host part of the specified host.
func NormalizeHost(host string) string {
	u, err := url.Parse(host)
	if err != nil {
		return host
	}
	if u.Scheme == "" || u.Host == "" {
		return host
	}

	normalized := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	return normalized.String()
}

// Match hosts using only Host part of the URL to allow cases when scheme is not specified
func MatchHost(host1, host2 string) bool {
	if host1 == "" || host2 == "" {
		return false
	}
	u1, err := url.Parse(fixUrlIfNeeded(host1))
	if err != nil {
		return false
	}
	u2, err := url.Parse(fixUrlIfNeeded(host2))
	if err != nil {
		return false
	}
	return u1.Host == u2.Host
}

func fixUrlIfNeeded(s string) string {
	if !strings.HasPrefix(s, "http") {
		return "https://" + s
	}
	return s
}
