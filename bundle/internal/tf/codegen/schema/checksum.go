package schema

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
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
// linux_arm64 archives. It also downloads both zips to verify that the parsed
// checksums are correct.
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

	// Sanity check: download both zips and verify the checksums match.
	err = verifyProviderChecksum(version, "linux_amd64", checksums.LinuxAmd64)
	if err != nil {
		return nil, err
	}
	err = verifyProviderChecksum(version, "linux_arm64", checksums.LinuxArm64)
	if err != nil {
		return nil, err
	}

	return checksums, nil
}

// verifyProviderChecksum downloads the provider zip for the given platform and
// verifies it matches the expected SHA256 checksum.
func verifyProviderChecksum(version, platform, expectedChecksum string) error {
	url := fmt.Sprintf(
		"https://github.com/databricks/terraform-provider-databricks/releases/download/v%s/terraform-provider-databricks_%s_%s.zip",
		version, version, platform,
	)

	log.Printf("verifying checksum for %s provider archive", platform)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("downloading provider archive for checksum verification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading provider archive for checksum verification: HTTP %s", resp.Status)
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, resp.Body); err != nil {
		return fmt.Errorf("computing checksum for provider archive: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch for %s provider archive: expected %s, got %s", platform, expectedChecksum, actualChecksum)
	}

	log.Printf("checksum verified for %s provider archive", platform)
	return nil
}
