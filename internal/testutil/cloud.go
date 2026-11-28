package testutil

type Cloud int

const (
	AWS Cloud = iota
	Azure
	GCP
)

func (c Cloud) NodeTypeID() string {
	switch c {
	case AWS:
		return "i3.xlarge"
	case Azure:
		return "Standard_E4ds_v5"
	case GCP:
		return "n1-standard-4"
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
	case "gcp-ucws":
		return GCP
	// CLOUD_ENV is set to "ucws" in the "aws-prod-ucws" test environment
	case "ucws":
		return AWS
	default:
		t.Fatalf("Unknown cloud environment: %s", env)
	}
	return -1
}
