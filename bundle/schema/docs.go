package schema

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Docs struct {
	Documentation string          `json:"documentation"`
	Children      map[string]Docs `json:"children"`
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
