package tnstate

import "fmt"

type CLIVersion struct {
	Major int64 `json:"major"`
	Minor int64 `json:"minor"`
	Patch int64 `json:"patch"`
}

func (v CLIVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v CLIVersion) EqualORGreater(other CLIVersion) bool {
	return (v.Major > other.Major) || (v.Major == other.Major && v.Minor > other.Minor) || (v.Major == other.Major && v.Minor == other.Minor && v.Patch >= other.Patch)
}
