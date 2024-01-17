package yamlsaver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

func SaveAsYAML(data any, filename string, force bool) error {
	err := os.MkdirAll(filepath.Dir(filename), 0755)
	if err != nil {
		return err
	}

	// check that file exists
	info, err := os.Stat(filename)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("%s is a directory", filename)
		}
		if !force {
			return fmt.Errorf("%s already exists. Use --force to overwrite", filename)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	err = encode(data, file)
	if err != nil {
		return err
	}
	return nil
}

func encode(data any, w io.Writer) error {
	yamlNode, err := ToYamlNode(dyn.V(data))
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(yamlNode)
}

func ToYamlNode(v dyn.Value) (*yaml.Node, error) {
	switch v.Kind() {
	case dyn.KindMap:
		m, _ := v.AsMap()
		keys := maps.Keys(m)
		// We're using location lines to define the order of keys in YAML.
		// The location is set when we convert API response struct to config.Value representation
		// See convert.convertMap for details
		sort.SliceStable(keys, func(i, j int) bool {
			return m[keys[i]].Location().Line < m[keys[j]].Location().Line
		})

		content := make([]*yaml.Node, 0)
		for _, k := range keys {
			item := m[k]
			node := yaml.Node{Kind: yaml.ScalarNode, Value: k}
			c, err := ToYamlNode(item)
			if err != nil {
				return nil, err
			}
			content = append(content, &node)
			content = append(content, c)
		}

		return &yaml.Node{Kind: yaml.MappingNode, Content: content}, nil
	case dyn.KindSequence:
		s, _ := v.AsSequence()
		content := make([]*yaml.Node, 0)
		for _, item := range s {
			node, err := ToYamlNode(item)
			if err != nil {
				return nil, err
			}
			content = append(content, node)
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content}, nil
	case dyn.KindNil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}, nil
	case dyn.KindString:
		// If the string is a scalar value (bool, int, float and etc.), we want to quote it.
		if isScalarValueInString(v) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString(), Style: yaml.DoubleQuotedStyle}, nil
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString()}, nil
	case dyn.KindBool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustBool())}, nil
	case dyn.KindInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustInt())}, nil
	case dyn.KindFloat:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustFloat())}, nil
	case dyn.KindTime:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustTime().UTC().String()}, nil
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid kind: %d", v.Kind()))
	}
}

func isScalarValueInString(v dyn.Value) bool {
	if v.Kind() != dyn.KindString {
		return false
	}

	// Parse value of the string and check if it's a scalar value.
	// If it's a scalar value, we want to quote it.
	switch v.MustString() {
	case "true", "false":
		return true
	default:
		_, err := parseNumber(v.MustString())
		return err == nil
	}
}

func parseNumber(s string) (any, error) {
	if i, err := strconv.ParseInt(s, 0, 64); err == nil {
		return i, nil
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("invalid number: %s", s)
}
