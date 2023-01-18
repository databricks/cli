package schema

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Docs struct {
	Documentation string          `json:"documentation"`
	Children      map[string]Docs `json:"children"`
}

func LoadDocs(path string) (*Docs, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadAll(f)
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
