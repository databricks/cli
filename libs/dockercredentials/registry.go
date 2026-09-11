package dockercredentials

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/databricks/databricks-sdk-go/common/environment"
)

const (
	// OAuthTokenUsername is the username returned to Docker with an OAuth access token.
	OAuthTokenUsername = "oauthtoken"
	registryHostInfix  = ".container."
)

// Registry identifies the workspace and canonical host of a Databricks Artifact Registry endpoint.
type Registry struct {
	WorkspaceID string
	Host        string
}

// RegistryHost builds a registry host in the workspace's cloud and environment DNS zone.
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
	return fmt.Sprintf("%s.container.%s%s", workspaceID, region, dnsZone), nil
}

// normalizeServerAddress accepts the bare host or HTTPS URL forms allowed by Docker's credential-helper protocol.
// Databricks Artifact Registry uses standard HTTPS and has no configurable port.
func normalizeServerAddress(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("server address is required")
	}

	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("parse server address %q: %w", raw, err)
	}
	if !strings.EqualFold(u.Scheme, "https") {
		return "", fmt.Errorf("unsupported registry URL scheme %q", u.Scheme)
	}
	if u.Port() != "" {
		return "", errors.New("registry address must not include a port")
	}

	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return "", errors.New("server address is required")
	}
	return host, nil
}

// ParseRegistryHost normalizes a Databricks Artifact Registry address and extracts its workspace ID.
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
		Host:        host,
	}, nil
}

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

func matchingDatabricksDNSZone(host string) (string, bool) {
	dnsZone := strings.ToLower(environment.GetEnvironmentForHostname(host).DnsZone)
	// The SDK defaults unknown hosts to AWS production, so verify that the returned zone actually matched.
	return dnsZone, dnsZone != "" && strings.HasSuffix(host, dnsZone)
}

// isDNSLabel accepts one lowercase ASCII label: letters, digits, and interior hyphens, up to 63 bytes.
// Workspace IDs, regions, and Docker input are not uniformly prevalidated before becoming hostname components.
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
