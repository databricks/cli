package testutil

import (
	"testing"
)

type Cloud int

const (
	AWS Cloud = iota
	Azure
	GCP
)

// Implement [Requirement].
func (c Cloud) Verify(t *testing.T) {
	if c != GetCloud(t) {
		t.Skipf("Skipping %s-specific test", c)
	}
}

func (c Cloud) String() string {
	switch c {
	case AWS:
		return "AWS"
	case Azure:
		return "Azure"
	case GCP:
		return "GCP"
	default:
		return "unknown"
	}
}

func GetCloud(t *testing.T) Cloud {
	env := GetEnvOrSkipTest(t, "CLOUD_ENV")
	switch env {
	case "aws":
		return AWS
	case "azure":
		return Azure
	case "gcp":
		return GCP
	default:
		t.Fatalf("Unknown cloud environment: %s", env)
	}
	return -1
}
