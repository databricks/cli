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

// CalculateNewVersion generates a new version string and filename based on the wheel info and modification time.
// The version is updated according to the following rules:
//   - if there is an existing part after + it is dropped
//   - append +<mtime of the original wheel> to version
func CalculateNewVersion(info *WheelInfo, mtime time.Time) (newVersion, newFilename string) {
	baseVersion := strings.SplitN(info.Version, "+", 2)[0]
	
	dt := strings.Replace(mtime.Format("20060102150405.00"), ".", "", 1)
	dt = strings.Replace(dt, ".", "", 1)
	newVersion = baseVersion + "+" + dt
	
	newFilename = fmt.Sprintf("%s-%s-%s.whl",
		info.Distribution,
		newVersion,
		strings.Join(info.Tags, "-"))
		
	return newVersion, newFilename
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
