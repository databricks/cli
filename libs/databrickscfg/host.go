package databrickscfg

import (
	"net/url"
	"strings"
)

// normalizeHost returns the string representation of only
// the scheme and host part of the specified host, without
// the scheme's default port.
func normalizeHost(host string) string {
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
	port := u.Port()
	if (u.Scheme == "https" && port == "443") || (u.Scheme == "http" && port == "80") {
		normalized.Host = strings.TrimSuffix(u.Host, ":"+port)
	}

	return normalized.String()
}
