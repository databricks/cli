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
	DNSZone     string
}

// ServesWorkspaceHost reports whether r may receive tokens for workspaceHost.
// Registry and workspace DNS zones must match exactly. Loopback test-server
// registries and real workspaces never match each other.
func (r Registry) ServesWorkspaceHost(workspaceHost string) bool {
	return r.servesWorkspaceHostInZones(workspaceHost, registryDNSZones())
}

func (r Registry) servesWorkspaceHostInZones(workspaceHost string, zones []string) bool {
	zone, err := registryDNSZoneForWorkspaceHostInZones(workspaceHost, zones)
	return err == nil && r.DNSZone == zone
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
	return registryHostInZones(workspaceID, region, workspaceHost, registryDNSZones())
}

func registryHostInZones(workspaceID, region, workspaceHost string, zones []string) (string, error) {
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
	dnsZone, err := registryDNSZoneForWorkspaceHostInZones(workspaceHost, zones)
	if err != nil {
		return "", err
	}
	return registryHostForZone(workspaceID, region, dnsZone), nil
}

func registryHostForZone(workspaceID, region, dnsZone string) string {
	return fmt.Sprintf("%s.container.%s%s", workspaceID, region, dnsZone)
}

var databricksDNSZones = collectRegistryDNSZones()

func collectRegistryDNSZones() []string {
	environments := environment.AllEnvironments()
	result := make([]string, 0, len(environments))
	for _, env := range environments {
		if env.DnsZone != "" {
			result = append(result, env.DnsZone)
		}
	}
	return result
}

func registryDNSZones() []string {
	return databricksDNSZones
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
	return parseRegistryHost(raw, registryDNSZones())
}

func parseRegistryHost(raw string, zones []string) (Registry, error) {
	host, err := normalizeServerAddress(raw)
	if err != nil {
		return Registry{}, err
	}

	dnsZone, ok := matchingDNSZone(host, zones)
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
	return Registry{WorkspaceID: workspaceID, Host: host, DNSZone: dnsZone}, nil
}

func registryDNSZoneForWorkspaceHost(raw string) (string, error) {
	return registryDNSZoneForWorkspaceHostInZones(raw, registryDNSZones())
}

func registryDNSZoneForWorkspaceHostInZones(raw string, zones []string) (string, error) {
	if isLocalWorkspaceHost(raw) {
		return localRegistryDNSZone, nil
	}
	host, err := normalizeServerAddress(raw)
	if err != nil {
		return "", fmt.Errorf("parse workspace host: %w", err)
	}
	dnsZone, ok := matchingDNSZone(host, zones)
	if !ok {
		return "", fmt.Errorf("%q is not a supported Databricks workspace host", host)
	}
	return dnsZone, nil
}

// isLocalWorkspaceHost matches reserved loopback test hosts accepted by the
// Docker credential helper. They must never match a real workspace registry.
func isLocalWorkspaceHost(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := u.Hostname()
	return host == "127.0.0.1" || host == "localhost" || strings.HasSuffix(host, ".localhost")
}

func matchingDNSZone(host string, zones []string) (string, bool) {
	var matched string
	for _, zone := range zones {
		if strings.HasSuffix(host, zone) && len(zone) > len(matched) {
			matched = zone
		}
	}
	return matched, matched != ""
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
