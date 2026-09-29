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
	// localRegistryDNSZone stands in for the DNS zone of a workspace served by a loopback test server.
	localRegistryDNSZone = ".localhost"
)

// Registry identifies the workspace and canonical host of a Databricks Artifact Registry endpoint.
type Registry struct {
	WorkspaceID string
	Host        string
}

// ServesWorkspaceHost reports whether r may receive tokens for workspaceHost.
// Loopback test-server registries and real workspaces never match each other.
func (r Registry) ServesWorkspaceHost(workspaceHost string) bool {
	return strings.HasSuffix(r.Host, localRegistryDNSZone) == isLocalWorkspaceHost(workspaceHost)
}

// ValidateWorkspaceHost checks that a workspace host can be used to derive an Artifact Registry host.
func ValidateWorkspaceHost(workspaceHost string) error {
	_, err := registryDNSZoneForWorkspaceHost(workspaceHost)
	return err
}

// ValidateRegion checks that a region can be used as an Artifact Registry hostname label.
func ValidateRegion(region string) error {
	region = strings.TrimSpace(region)
	if region == "" {
		return errors.New("region is required")
	}
	if !isDNSLabel(region) {
		return fmt.Errorf("invalid region %q", region)
	}
	return nil
}

// RegistryHost builds a registry host in the workspace's cloud and environment DNS zone.
func RegistryHost(workspaceID, region, workspaceHost string) (string, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	region = strings.TrimSpace(region)
	if workspaceID == "" {
		return "", errors.New("workspace ID is required")
	}
	if !isDNSLabel(workspaceID) {
		return "", fmt.Errorf("invalid workspace ID %q", workspaceID)
	}
	if err := ValidateRegion(region); err != nil {
		return "", err
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
	if strings.HasSuffix(host, localRegistryDNSZone) {
		dnsZone, ok = localRegistryDNSZone, true
	}
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
	if isLocalWorkspaceHost(raw) {
		return localRegistryDNSZone, nil
	}
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

// isLocalWorkspaceHost matches the plain-HTTP loopback test server that OAuth login also accepts.
func isLocalWorkspaceHost(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && u.Scheme == "http" && u.Hostname() == "127.0.0.1"
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
