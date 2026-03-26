package schema

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
)

// ProviderChecksums holds the SHA256 checksums for the Databricks Terraform
// provider archive for supported Linux architectures.
type ProviderChecksums struct {
	LinuxAmd64 string
	LinuxArm64 string
}

// FetchProviderChecksums downloads the SHA256SUMS file from the GitHub release
// for the given provider version and extracts checksums for the linux_amd64 and
// linux_arm64 archives.
// https://github.com/databricks/terraform-provider-databricks/releases
func FetchProviderChecksums(version string) (*ProviderChecksums, error) {
	url := fmt.Sprintf(
		"https://github.com/databricks/terraform-provider-databricks/releases/download/v%s/terraform-provider-databricks_%s_SHA256SUMS",
		version, version,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("downloading SHA256SUMS for provider v%s: %w", version, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading SHA256SUMS for provider v%s: HTTP %s", version, resp.Status)
	}

	checksums := &ProviderChecksums{}
	amd64Suffix := fmt.Sprintf("terraform-provider-databricks_%s_linux_amd64.zip", version)
	arm64Suffix := fmt.Sprintf("terraform-provider-databricks_%s_linux_arm64.zip", version)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		switch parts[1] {
		case amd64Suffix:
			checksums.LinuxAmd64 = parts[0]
		case arm64Suffix:
			checksums.LinuxArm64 = parts[0]
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading SHA256SUMS for provider v%s: %w", version, err)
	}

	if checksums.LinuxAmd64 == "" {
		return nil, fmt.Errorf("checksum not found for %s in SHA256SUMS", amd64Suffix)
	}
	if checksums.LinuxArm64 == "" {
		return nil, fmt.Errorf("checksum not found for %s in SHA256SUMS", arm64Suffix)
	}

	return checksums, nil
}
