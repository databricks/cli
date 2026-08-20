package dockercredentials

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/databricks/databricks-sdk-go/common/environment"
)

const (
	// OAuthTokenUsername is the username returned to Docker with an OAuth access token.
	OAuthTokenUsername = "oauthtoken"
	registryHostInfix  = ".container."
)

// Registry identifies the workspace, region, and canonical host of a Databricks Artifact Registry endpoint.
type Registry struct {
	WorkspaceID string
	Region      string
	Host        string
}

// RegistryHost derives <workspace-id>.container.<region>.<dns-zone> while preserving the workspace's cloud and environment zone.
func RegistryHost(workspaceID, region, workspaceHost string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	region = strings.TrimSpace(region)
	if workspaceID == "" {
		return "", errors.New("workspace ID is required")
	}
	if region == "" {
		return "", errors.New("region is required")
	}
	if !isDNSLabel(workspaceID) {
		return "", fmt.Errorf("invalid workspace ID %q", workspaceID)
	}
	if !isDNSLabel(region) {
		return "", fmt.Errorf("invalid region %q", region)
	}
	dnsZone, err := registryDNSZoneForWorkspaceHost(workspaceHost)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%s%s%s", workspaceID, registryHostInfix, region, dnsZone), nil
}

// normalizeServerAddress canonicalizes Docker's URL-or-host input and permits only HTTPS on the default registry port.
func normalizeServerAddress(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("server address is required")
	}

	if strings.Contains(value, "://") {
		u, err := url.Parse(value)
		if err != nil {
			return "", fmt.Errorf("parse server address %q: %w", raw, err)
		}
		if !strings.EqualFold(u.Scheme, "https") {
			return "", fmt.Errorf("unsupported registry URL scheme %q", u.Scheme)
		}
		value = u.Host
	} else if i := strings.IndexByte(value, '/'); i >= 0 {
		value = value[:i]
	}

	if host, port, ok, err := splitOptionalPort(value); err != nil {
		return "", err
	} else if ok {
		value = host
		if err := validatePort(port); err != nil {
			return "", err
		}
		if port != "443" {
			return "", fmt.Errorf("unsupported registry port %q", port)
		}
	}

	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if value == "" {
		return "", errors.New("server address is required")
	}
	return value, nil
}

// ParseRegistryHost normalizes a Databricks Artifact Registry address and extracts its workspace and region.
func ParseRegistryHost(raw string) (Registry, error) {
	host, err := normalizeServerAddress(raw)
	if err != nil {
		return Registry{}, err
	}

	dnsZone, ok := matchingDatabricksDNSZone(host)
	if !ok {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}

	trimmed := strings.TrimSuffix(host, dnsZone)
	workspaceID, region, ok := strings.Cut(trimmed, registryHostInfix)
	if !ok || !isDNSLabel(workspaceID) || !isDNSLabel(region) {
		return Registry{}, fmt.Errorf("%q is not a Databricks Artifact Registry host", host)
	}

	return Registry{
		WorkspaceID: workspaceID,
		Region:      region,
		Host:        host,
	}, nil
}

// registryDNSZoneForWorkspaceHost derives the registry suffix from the workspace's SDK-known cloud and environment zone.
func registryDNSZoneForWorkspaceHost(raw string) (string, error) {
	host, err := normalizeServerAddress(raw)
	if err != nil {
		return "", fmt.Errorf("parse workspace host: %w", err)
	}
	dnsZone, ok := matchingDatabricksDNSZone(host)
	if !ok {
		return "", fmt.Errorf("%q is not a supported Databricks workspace host", host)
	}
	return dnsZone, nil
}

// matchingDatabricksDNSZone selects the most specific suffix from all SDK-known Databricks environments.
func matchingDatabricksDNSZone(host string) (string, bool) {
	return matchingDatabricksDNSZoneInEnvironments(host, environment.AllEnvironments())
}

// matchingDatabricksDNSZoneInEnvironments prefers the longest suffix so environment-specific zones beat generic ones.
func matchingDatabricksDNSZoneInEnvironments(host string, envs []environment.DatabricksEnvironment) (string, bool) {
	var match string
	for _, e := range envs {
		dnsZone := strings.ToLower(e.DnsZone)
		if dnsZone == "" {
			continue
		}
		if strings.HasSuffix(host, dnsZone) && len(dnsZone) > len(match) {
			match = dnsZone
		}
	}
	return match, match != ""
}

// splitOptionalPort extracts bracketed or plain host ports while leaving non-port colon forms for host validation.
func splitOptionalPort(value string) (host, port string, ok bool, err error) {
	host, port, err = net.SplitHostPort(value)
	if err == nil {
		return host, port, true, nil
	}

	if strings.Count(value, ":") == 1 {
		host, port, found := strings.Cut(value, ":")
		if found && port != "" {
			return host, port, true, nil
		}
	}

	return "", "", false, nil
}

// validatePort enforces the decimal TCP port range that URL parsing alone does not validate.
func validatePort(port string) error {
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("invalid registry port %q", port)
	}
	return nil
}

// isDNSLabel restricts interpolated registry components to unescaped lowercase ASCII DNS labels.
func isDNSLabel(label string) bool {
	if label == "" || len(label) > 63 {
		return false
	}
	for i, r := range label {
		if r > unicode.MaxASCII {
			return false
		}
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		if r == '-' && i > 0 && i < len(label)-1 {
			continue
		}
		return false
	}
	return true
}
