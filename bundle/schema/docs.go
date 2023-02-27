package schema

import (
	_ "embed"
	"os"

	"gopkg.in/yaml.v3"
)

type Docs struct {
	Description *string          `json:"description"`
	Properties  map[string]*Docs `json:"properties"`
}

func LoadDocs(path string) (*Docs, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	docs := Docs{}
	err = yaml.Unmarshal(bytes, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}

//go:embed bundle_config_docs.yml
var bundleDocs []byte

func GetBundleDocs() (*Docs, error) {
	docs := Docs{}
	err := yaml.Unmarshal(bundleDocs, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}
