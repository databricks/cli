package testutil

type Cloud int

const (
	AWS Cloud = iota
	Azure
	GCP
)

// Implement [Requirement].
func (c Cloud) Verify(t TestingT) {
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

func GetCloud(t TestingT) Cloud {
	env := GetEnvOrSkipTest(t, "CLOUD_ENV")
	switch env {
	case "aws":
		return AWS
	case "azure":
		return Azure
	case "gcp":
		return GCP
	// CLOUD_ENV is set to "ucws" in the "aws-prod-ucws" test environment
	case "ucws":
		return AWS
	default:
		t.Fatalf("Unknown cloud environment: %s", env)
	}
	return -1
}

func IsAWSCloud(t TestingT) bool {
	return GetCloud(t) == AWS
}
