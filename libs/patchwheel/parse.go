package patchwheel

import (
	"fmt"
	"strings"
	"time"
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
// Wheel filenames follow the pattern: {distribution}-{version}(-{build tag})?-{python_tag}-{abi_tag}-{platform_tag}.whl
func ParseWheelFilename(filename string) (*WheelInfo, error) {
	parts := strings.Split(filename, "-")
	if len(parts) < 5 {
		return nil, fmt.Errorf("invalid wheel filename format: not enough parts in %s", filename)
	}
	if !strings.HasSuffix(parts[len(parts)-1], ".whl") {
		return nil, fmt.Errorf("invalid wheel filename format: missing .whl extension in %s", filename)
	}

	// The last three parts are always the python, abi and platform tags
	tagStartIdx := len(parts) - 3
	tags := parts[tagStartIdx:]
	tags[2] = strings.TrimSuffix(tags[2], ".whl")

	// The distribution is always the first part
	distribution := parts[0]
	
	// The version is the second part - don't include build tags
	version := parts[1]

	// If there are build tags between version and python tag, include them in tags
	if tagStartIdx > 2 {
		buildTags := parts[2:tagStartIdx]
		tags = append(buildTags, tags...)
	}

	return &WheelInfo{
		Distribution: distribution,
		Version:      version,
		Tags:         tags,
	}, nil
}
