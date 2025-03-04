package patchwheel

import (
	"fmt"
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
	parts := strings.Split(filename, "-")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid wheel filename format: not enough parts in %s", filename)
	}
	if !strings.HasSuffix(parts[len(parts)-1], ".whl") {
		return nil, fmt.Errorf("invalid wheel filename format: missing .whl extension in %s", filename)
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
