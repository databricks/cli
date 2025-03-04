package patchwheel

import (
	"fmt"
	"path/filepath"
	"strings"
)

// WheelInfo contains information extracted from a wheel filename
type WheelInfo struct {
	Distribution string   // Package distribution name
	Version      string   // Package version
	Tags         []string // Python tags (python_tag, abi_tag, platform_tag)
}

// ParseWheelFilename parses a wheel filename and extracts its components.
// Wheel filenames follow the pattern: {distribution}-{version}-{python_tag}-{abi_tag}-{platform_tag}.whl
func ParseWheelFilename(filename string) (*WheelInfo, error) {
	base := filepath.Base(filename)
	parts := strings.Split(base, "-")
	if len(parts) < 5 || !strings.HasSuffix(parts[len(parts)-1], ".whl") {
		return nil, fmt.Errorf("invalid wheel filename format: %s", filename)
	}

	// The last three parts are always tags
	tagStartIdx := len(parts) - 3
	tags := parts[tagStartIdx:]
	tags[2] = strings.TrimSuffix(tags[2], ".whl")

	// Everything before the tags except the version is the distribution
	versionIdx := tagStartIdx - 1

	// Distribution may contain hyphens, so join all parts before the version
	distribution := strings.Join(parts[:versionIdx], "-")
	version := parts[versionIdx]

	return &WheelInfo{
		Distribution: distribution,
		Version:      version,
		Tags:         tags,
	}, nil
}
