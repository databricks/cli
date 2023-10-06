package metadata

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deploy"
)

type computeMetadata struct{}

func ComputeMetadata() bundle.Mutator {
	return &computeMetadata{}
}

func (m *computeMetadata) Name() string {
	return "ComputeMetadata"
}

func walk(config, metadata reflect.Value) (bool, error) {
	if config.Type() != metadata.Type() {
		return false, fmt.Errorf("config and metadata have different types. Config is %s. Metadata is %s", config.Type(), metadata.Type())
	}

	if config.Kind() == reflect.Pointer {
		// Skip if pointer has no value assigned
		if config.IsNil() {
			return false, nil
		}
		// Initialize a new pointer to store metadata values while recursively walking.
		metadata.Set(reflect.New(config.Elem().Type()))
		return walk(config.Elem(), metadata.Elem())
	}

	for i := 0; i < config.NumField(); i++ {
		field := config.Type().Field(i)

		// Skip fields that are not exported.
		if !field.IsExported() {
			continue

		}

		// Assign metadata and return early if metadata tag is found
		// TODO: add tests for points of structs and tags being covered here.
		// TODO: Add note for why it's fine to directly assign pointers here.
		bundleTags, ok := field.Tag.Lookup("bundle")
		if ok && slices.Contains(strings.Split(bundleTags, ","), deploy.MetadataTag) {
			metadata.Field(i).Set(config.Field(i))
			continue
		}

		// Recursively walk into fields that are embedded structs, structs and a pointer
		// to a struct
		if field.Anonymous || field.Type.Kind() == reflect.Struct ||
			(field.Type.Kind() == reflect.Pointer && field.Type.Elem().Kind() == reflect.Struct) {
			foundMetadata, err := walk(config.Field(i), metadata.Field(i))
			if err != nil {
				return foundMetadata, err
			}
		}

		if field.Type.Kind() == reflect.Map && config.Field(i).Len() > 0 {
			iter := config.Field(i).MapRange()
			metadata.Field(i).Set(reflect.MakeMap(field.Type))
			for iter.Next() {
				k := iter.Key()
				v := iter.Value()

				// Filter out the metadata fields from the map element's value
				tmp := reflect.New(v.Type()).Elem()
				err := walk(v, tmp)
				if err != nil {
					return err
				}

				// Assign the computed metadata as an entry in the map
				metadata.Field(i).SetMapIndex(k, tmp)
			}
		}
	}
	return nil
}

func (m *computeMetadata) Apply(_ context.Context, b *bundle.Bundle) error {
	b.Metadata = deploy.Metadata{
		Version: deploy.LatestMetadataVersion,
		Config:  config.Root{},
	}

	config := reflect.ValueOf(b.Config)
	// Third law of reflection in golang: To modify a reflection object, the value must be settable.
	// Settability requires passing the pointer to the config struct.
	// see: https://go.dev/blog/laws-of-reflection
	metadata := reflect.ValueOf(&b.Metadata.Config).Elem()
	return walk(config, metadata)
}
