package configsync

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/tailscale/hujson"
	"go.yaml.in/yaml/v3"
)

// The patcher emits YAML. Convert its syntax tree directly so keys stay in
// source order and untouched JSON numbers never pass through float64.
func formatPatchedJSON(content []byte) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	root, err := jsonFromYAMLNode(&doc)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := json.Indent(&out, root.Pack(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

func jsonFromYAMLNode(n *yaml.Node) (hujson.Value, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		return jsonFromYAMLNode(n.Content[0])
	case yaml.AliasNode:
		return hujson.Value{}, fmt.Errorf("cannot represent YAML alias %q in JSON", n.Value)
	case yaml.MappingNode, yaml.SequenceNode:
		children := make([]hujson.Value, len(n.Content))
		for i, child := range n.Content {
			value, err := jsonFromYAMLNode(child)
			if err != nil {
				return hujson.Value{}, err
			}
			children[i] = value
		}
		if n.Kind == yaml.SequenceNode {
			return hujson.Value{Value: &hujson.Array{Elements: children}}, nil
		}
		object := &hujson.Object{}
		for i := 0; i < len(children); i += 2 {
			object.Members = append(object.Members, hujson.ObjectMember{Name: children[i], Value: children[i+1]})
		}
		return hujson.Value{Value: object}, nil
	case yaml.ScalarNode:
		switch n.ShortTag() {
		case "!!str", "!!timestamp":
			return hujson.Value{Value: hujson.String(n.Value)}, nil
		case "!!null":
			return hujson.Value{Value: hujson.Literal("null")}, nil
		case "!!bool", "!!int", "!!float":
			return hujson.Value{Value: hujson.Literal(n.Value)}, nil
		}
	}
	return hujson.Value{}, fmt.Errorf("cannot represent YAML tag %q in JSON", n.ShortTag())
}
