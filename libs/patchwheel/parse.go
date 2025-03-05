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

// calculateNewVersion generates a new version string and filename based on the wheel info and modification time.
// The version is updated according to the following rules:
//   - if there is an existing part after + it is dropped
//   - append +<mtime of the original wheel> to version
//
// Example transform: "1.2.3" -> "1.2.3+2025030412345678"
func calculateNewVersion(info WheelInfo, mtime time.Time) (newVersion, newFilename string) {
	baseVersion, _, _ := strings.Cut(info.Version, "+")

	dt := mtime.Format("20060102150405.00")
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
// https://peps.python.org/pep-0491
func ParseWheelFilename(filename string) (WheelInfo, error) {
	parts := strings.Split(filename, "-")
	if len(parts) < 5 {
		return WheelInfo{}, fmt.Errorf("invalid wheel filename format: not enough parts in %s", filename)
	}

	if len(parts) > 6 {
		return WheelInfo{}, fmt.Errorf("invalid wheel filename format: too many parts in %s", filename)
	}

	trimmedLastTag, foundWhl := strings.CutSuffix(parts[len(parts)-1], ".whl")

	if !foundWhl {
		return WheelInfo{}, fmt.Errorf("invalid wheel filename format: missing .whl extension in %s", filename)
	}

	parts[len(parts)-1] = trimmedLastTag

	return WheelInfo{
		Distribution: parts[0],
		Version:      parts[1],
		Tags:         parts[2:],
	}, nil
}
