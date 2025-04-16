package databrickscfg

import "net/url"

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
