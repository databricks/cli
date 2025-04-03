package yamlsaver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/databricks/cli/libs/dyn"
	"gopkg.in/yaml.v3"
)

type saver struct {
	nodesWithStyle map[string]yaml.Style
}

func NewSaver() *saver {
	return &saver{}
}

func NewSaverWithStyle(nodesWithStyle map[string]yaml.Style) *saver {
	return &saver{
		nodesWithStyle: nodesWithStyle,
	}
}

func (s *saver) SaveAsYAML(data any, filename string, force bool) error {
	err := os.MkdirAll(filepath.Dir(filename), 0o755)
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

	err = s.encode(data, file)
	if err != nil {
		return err
	}
	return nil
}

func (s *saver) encode(data any, w io.Writer) error {
	yamlNode, err := s.toYamlNode(dyn.V(data))
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	return enc.Encode(yamlNode)
}

func (s *saver) toYamlNode(v dyn.Value) (*yaml.Node, error) {
	return s.toYamlNodeWithStyle(v, yaml.Style(0))
}

func (s *saver) toYamlNodeWithStyle(v dyn.Value, style yaml.Style) (*yaml.Node, error) {
	switch v.Kind() {
	case dyn.KindMap:
		m, _ := v.AsMap()

		// We're using location lines to define the order of keys in YAML.
		// The location is set when we convert API response struct to config.Value representation
		// See convert.convertMap for details
		pairs := m.Pairs()
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].Value.Location().Line < pairs[j].Value.Location().Line
		})

		var content []*yaml.Node
		for _, pair := range pairs {
			pk := pair.Key
			pv := pair.Value
			node := yaml.Node{Kind: yaml.ScalarNode, Value: pk.MustString(), Style: style}
			var nestedNodeStyle yaml.Style
			if customStyle, ok := s.hasStyle(pk.MustString()); ok {
				nestedNodeStyle = customStyle
			} else {
				nestedNodeStyle = style
			}
			c, err := s.toYamlNodeWithStyle(pv, nestedNodeStyle)
			if err != nil {
				return nil, err
			}
			content = append(content, &node)
			content = append(content, c)
		}

		return &yaml.Node{Kind: yaml.MappingNode, Content: content, Style: style}, nil
	case dyn.KindSequence:
		seq, _ := v.AsSequence()
		var content []*yaml.Node
		for _, item := range seq {
			node, err := s.toYamlNodeWithStyle(item, style)
			if err != nil {
				return nil, err
			}
			content = append(content, node)
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content, Style: style}, nil
	case dyn.KindNil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null", Style: style}, nil
	case dyn.KindString:
		// If the string is a scalar value (bool, int, float and etc.), we want to quote it.
		if isScalarValueInString(v) {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString(), Style: yaml.DoubleQuotedStyle}, nil
		}
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString(), Style: style}, nil
	case dyn.KindBool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatBool(v.MustBool()), Style: style}, nil
	case dyn.KindInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: strconv.FormatInt(v.MustInt(), 10), Style: style}, nil
	case dyn.KindFloat:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustFloat()), Style: style}, nil
	case dyn.KindTime:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustTime().String(), Style: style}, nil
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid kind: %d", v.Kind()))
	}
}

func (s *saver) hasStyle(key string) (yaml.Style, bool) {
	style, ok := s.nodesWithStyle[key]
	return style, ok
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
	case "":
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
