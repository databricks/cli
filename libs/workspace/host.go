package workspace

import "net/url"

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
