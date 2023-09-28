package tags

import "github.com/databricks/databricks-sdk-go/config"

type Cloud interface {
	// ValidateKey checks if a tag key can be used with the cloud provider.
	ValidateKey(key string) error

	// ValidateValue checks if a tag value can be used with the cloud provider.
	ValidateValue(value string) error

	// NormalizeKey normalizes a tag key for the cloud provider.
	NormalizeKey(key string) string

	// NormalizeValue normalizes a tag value for the cloud provider.
	NormalizeValue(value string) string
}

func ForCloud(cfg *config.Config) Cloud {
	var t *tag
	switch {
	case cfg.IsAws():
		t = awsTag
	case cfg.IsAzure():
		t = azureTag
	case cfg.IsGcp():
		t = gcpTag
	default:
		panic("unknown cloud provider")
	}
	return t
}
