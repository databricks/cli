package configsync

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file implements the subset of JSON Patch (RFC 6902) over yaml.v3 node
// trees that configsync needs: add, remove and replace. It is adapted from
// github.com/palantir/pkg/yamlpatch and keeps its semantics and error
// messages; applyChange relies on the "parent path ... does not exist"
// wording to create missing parents.
//
// Portions Copyright (c) 2025 Palantir Technologies. All rights reserved.
// Licensed under the BSD 3-Clause License:
// https://github.com/palantir/pkg/blob/master/LICENSE

// yamlPath is a parsed RFC 6901 JSON Pointer. The leading empty segment
// addresses the document node.
type yamlPath []string

var (
	rfc6901Decoder = strings.NewReplacer("~1", "/", "~0", "~")
	rfc6901Encoder = strings.NewReplacer("~", "~0", "/", "~1")
)

func parseYAMLPath(s string) (yamlPath, error) {
	if !strings.HasPrefix(s, "/") {
		return nil, errors.New("path must begin with leading slash")
	}
	if s == "/" {
		return yamlPath{""}, nil
	}
	parts := strings.Split(s, "/")
	for i := range parts {
		parts[i] = rfc6901Decoder.Replace(parts[i])
	}
	return parts, nil
}

func (p yamlPath) String() string {
	switch len(p) {
	case 0:
		return ""
	case 1:
		return "/"
	}
	parts := make([]string, len(p))
	for i, part := range p {
		parts[i] = rfc6901Encoder.Replace(part)
	}
	return strings.Join(parts, "/")
}

func (p yamlPath) key() string {
	if len(p) == 0 {
		return ""
	}
	return p[len(p)-1]
}

// applyYAMLPatch applies one operation to a YAML document and re-encodes it
// with 2-space indentation. Comments survive; some whitespace may change.
func applyYAMLPatch(content []byte, op OperationType, path yamlPath, value any) ([]byte, error) {
	doc := new(yaml.Node)
	if err := yaml.Unmarshal(content, doc); err != nil {
		return nil, fmt.Errorf("failed to convert YAML bytes to node: unmarshal original yaml bytes: %w", err)
	}

	var err error
	switch op {
	case OperationAdd:
		err = yamlPatchAdd(doc, path, value)
	case OperationRemove:
		err = yamlPatchRemove(doc, path)
	case OperationReplace:
		err = yamlPatchReplace(doc, path, value)
	default:
		err = errors.New("unexpected op")
	}
	if err != nil {
		return nil, fmt.Errorf("op %s %s: %w", op, path, err)
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return nil, fmt.Errorf("failed to convert patched node to bytes: marshal patched node: %w", err)
	}
	return buf.Bytes(), enc.Close()
}

func yamlPatchAdd(doc *yaml.Node, path yamlPath, value any) error {
	valueNode, err := valueToYAMLNode(value)
	if err != nil {
		return err
	}
	if path.String() == "/" {
		setDocumentContent(doc, valueNode)
		return nil
	}
	parent, _, err := yamlParentAndLeaf(doc, path)
	if err != nil {
		return err
	}
	return parent.add(path.key(), valueNode)
}

func yamlPatchRemove(doc *yaml.Node, path yamlPath) error {
	parent, _, err := yamlParentAndLeaf(doc, path)
	if err != nil {
		return err
	}
	return parent.remove(path.key())
}

func yamlPatchReplace(doc *yaml.Node, path yamlPath, value any) error {
	parent, _, err := yamlParentAndLeaf(doc, path)
	if err != nil {
		return err
	}
	valueNode, err := valueToYAMLNode(value)
	if err != nil {
		return err
	}
	return parent.set(path.key(), valueNode)
}

// yamlParentAndLeaf walks path and returns the container holding the last
// segment plus the node at that segment (nil if absent). It errors if any
// parent along the way does not exist.
func yamlParentAndLeaf(doc *yaml.Node, path yamlPath) (yamlContainer, *yaml.Node, error) {
	node := doc
	var container yamlContainer
	for i, part := range path {
		if node == nil {
			return yamlContainer{}, nil, fmt.Errorf("parent path %s does not exist", path[:i])
		}
		var err error
		container, err = newYAMLContainer(node)
		if err != nil {
			return yamlContainer{}, nil, err
		}
		node, err = container.get(part)
		if err != nil {
			return yamlContainer{}, nil, err
		}
	}
	return container, node, nil
}

// valueToYAMLNode round-trips value through yaml.Marshal and strips style
// and position so it is laid out like the surrounding document.
func valueToYAMLNode(value any) (*yaml.Node, error) {
	b, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	node := new(yaml.Node)
	if err := yaml.Unmarshal(b, node); err != nil {
		return nil, fmt.Errorf("unmarshal original yaml bytes: %w", err)
	}
	if node.Kind == yaml.DocumentNode {
		node = node.Content[0]
	}
	clearYAMLStyle(node)
	return node, nil
}

func clearYAMLStyle(node *yaml.Node) {
	node.Style = 0
	node.Line = 0
	node.Column = 0
	for _, n := range node.Content {
		clearYAMLStyle(n)
	}
}

func setDocumentContent(doc, value *yaml.Node) {
	switch {
	case doc.Kind == yaml.DocumentNode && isYAMLNull(doc.Content[0]):
		doc.Content[0] = value
	case doc.Kind == 0 && len(doc.Content) == 0:
		doc.Kind = yaml.DocumentNode
		doc.Content = []*yaml.Node{value}
	}
}

func isYAMLNull(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Tag == "!!null"
}

// yamlContainer is a mapping, sequence or document node addressed by
// JSON Pointer segments.
type yamlContainer struct {
	node *yaml.Node
}

func newYAMLContainer(node *yaml.Node) (yamlContainer, error) {
	switch node.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		return yamlContainer{node}, nil
	case yaml.DocumentNode:
		if len(node.Content) != 1 {
			return yamlContainer{}, fmt.Errorf("unexpected yaml node: expected DocumentNode to have 1 child, got %v", node.Content)
		}
		return yamlContainer{node}, nil
	case yaml.AliasNode:
		return newYAMLContainer(node.Alias)
	case yaml.ScalarNode:
		return yamlContainer{}, errors.New("unexpected yaml node: scalar can not be a container")
	default:
		return yamlContainer{}, fmt.Errorf("unexpected yaml node: kind %d tag %s", node.Kind, node.Tag)
	}
}

var errIllegalDocumentAccess = errors.New("*goyamlDocumentContainer does not allow non-empty key access")

func (c yamlContainer) get(key string) (*yaml.Node, error) {
	n := c.node
	switch n.Kind {
	case yaml.MappingNode:
		if err := c.validateMapping(); err != nil {
			return nil, err
		}
		_, v := c.findKey(key)
		return v, nil
	case yaml.SequenceNode:
		if key == "-" {
			return nil, nil
		}
		idx, err := parseSeqIndex(key)
		if err != nil {
			return nil, err
		}
		if idx > len(n.Content)-1 {
			return nil, nil
		}
		return n.Content[idx], nil
	default:
		if key != "" {
			return nil, errIllegalDocumentAccess
		}
		if isYAMLNull(n.Content[0]) {
			return nil, nil
		}
		return n.Content[0], nil
	}
}

func (c yamlContainer) set(key string, val *yaml.Node) error {
	n := c.node
	switch n.Kind {
	case yaml.MappingNode:
		if err := c.validateMapping(); err != nil {
			return err
		}
		i, _ := c.findKey(key)
		if i == -1 {
			return fmt.Errorf("key %s does not exist and can not be replaced", key)
		}
		n.Content[i+1] = val
		return nil
	case yaml.SequenceNode:
		idx, err := parseSeqIndex(key)
		if err != nil {
			return err
		}
		if idx > len(n.Content)-1 {
			return fmt.Errorf("set index key out of bounds (idx %d, len %d)", idx, len(n.Content))
		}
		n.Content[idx] = val
		return nil
	default:
		if key != "" {
			return errIllegalDocumentAccess
		}
		if isYAMLNull(n.Content[0]) {
			return errors.New("document value does not exist and can not be replaced")
		}
		n.Content[0] = val
		return nil
	}
}

func (c yamlContainer) add(key string, val *yaml.Node) error {
	n := c.node
	switch n.Kind {
	case yaml.MappingNode:
		if err := c.validateMapping(); err != nil {
			return err
		}
		if _, v := c.findKey(key); v != nil {
			return fmt.Errorf("key %s already exists and can not be added", key)
		}
		n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, val)
		return nil
	case yaml.SequenceNode:
		if key == "-" {
			n.Content = append(n.Content, val)
			return nil
		}
		idx, err := parseSeqIndex(key)
		if err != nil {
			return err
		}
		if idx > len(n.Content) {
			return fmt.Errorf("add index key out of bounds (idx %d, len %d)", idx, len(n.Content))
		}
		n.Content = slices.Insert(n.Content, idx, val)
		return nil
	default:
		if key != "" {
			return errIllegalDocumentAccess
		}
		if !isYAMLNull(n.Content[0]) {
			return errors.New("document value already exists and can not be added")
		}
		n.Content[0] = val
		return nil
	}
}

func (c yamlContainer) remove(key string) error {
	n := c.node
	switch n.Kind {
	case yaml.MappingNode:
		if err := c.validateMapping(); err != nil {
			return err
		}
		i, _ := c.findKey(key)
		if i == -1 {
			return fmt.Errorf("key %s does not exist and can not be removed", key)
		}
		n.Content = slices.Delete(n.Content, i, i+2)
		return nil
	case yaml.SequenceNode:
		idx, err := parseSeqIndex(key)
		if err != nil {
			return err
		}
		if idx > len(n.Content)-1 {
			return fmt.Errorf("remove index key out of bounds (idx %d, len %d)", idx, len(n.Content))
		}
		n.Content = slices.Delete(n.Content, idx, idx+1)
		return nil
	default:
		if key != "" {
			return errIllegalDocumentAccess
		}
		return errors.New("document does not implement Remove()")
	}
}

// findKey returns the index of the key node and the value node for key in a
// mapping (stored as [k0, v0, k1, v1, ...]), or -1 and nil.
func (c yamlContainer) findKey(key string) (int, *yaml.Node) {
	for i := 0; i < len(c.node.Content); i += 2 {
		if c.node.Content[i].Value == key {
			return i, c.node.Content[i+1]
		}
	}
	return -1, nil
}

func (c yamlContainer) validateMapping() error {
	if len(c.node.Content)%2 != 0 {
		return fmt.Errorf("expected MappingNode to have even number of children, got %d", len(c.node.Content))
	}
	for i := 0; i < len(c.node.Content); i += 2 {
		if k := c.node.Content[i]; k.Kind != yaml.ScalarNode {
			return fmt.Errorf("jsonpatch only supports scalar mapping keys, got %d %s", k.Kind, k.Tag)
		}
	}
	return nil
}

func parseSeqIndex(key string) (int, error) {
	idx, err := strconv.Atoi(key)
	if err != nil {
		return 0, fmt.Errorf("index into SequenceNode with non-integer %q key: %w", key, err)
	}
	if idx < 0 {
		return 0, fmt.Errorf("index into SequenceNode with negative %q key", key)
	}
	return idx, nil
}
