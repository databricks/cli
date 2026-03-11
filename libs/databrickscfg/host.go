package databrickscfg

import (
	"net/url"
	"strings"
)

// NormalizeHost returns the canonical representation of a Databricks host.
// It ensures the host has an https:// scheme and strips any path, query,
// or fragment components, returning only scheme://host[:port].
func NormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return host
	}

	// If no scheme, prepend https:// before parsing.
	// This is necessary because url.Parse treats schemeless input
	// (e.g. "myhost.com") as a path, not a host.
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}

	u, err := url.Parse(host)
	if err != nil {
		return host
	}
	if u.Host == "" {
		return host
	}

	return (&url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}).String()
}
