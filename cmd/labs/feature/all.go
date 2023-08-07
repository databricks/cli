package feature

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func LoadAll(ctx context.Context) (features []*Feature, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	labsDir, err := os.ReadDir(filepath.Join(home, ".databricks", "labs"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, v := range labsDir {
		if !v.IsDir() {
			continue
		}
		feature, err := NewFeature(v.Name())
		if err != nil {
			return nil, fmt.Errorf("%s: %w", v.Name(), err)
		}
		err = feature.loadMetadata()
		if err != nil {
			return nil, fmt.Errorf("%s metadata: %w", v.Name(), err)
		}
		features = append(features, feature)
	}
	return features, nil
}
