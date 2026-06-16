package databrickscfg

import "net/url"

// SameHost reports whether a and b refer to the same scheme and host,
// ignoring path, query, and trailing-slash differences.
func SameHost(a, b string) bool {
	return normalizeHost(a) == normalizeHost(b)
}

// normalizeHost returns the string representation of only
// the scheme and host part of the specified host.
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

	return normalized.String()
}
