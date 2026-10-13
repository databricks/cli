package pkgmanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/tailscale/hujson"
)

// RewriteJSON updates project naming and package-manager fields without reformatting the manifest.
func RewriteJSON(data []byte, name string, m Manager) ([]byte, error) {
	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	if pkg == nil {
		return nil, errors.New("package.json must contain a JSON object")
	}
	doc, err := hujson.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("parse package.json: %w", err)
	}
	Rewrite(pkg, m)
	obj := doc.Value.(*hujson.Object)
	setJSONField(obj, "name", name)
	if pin, ok := pkg["packageManager"].(string); ok {
		setJSONField(obj, "packageManager", pin)
	} else {
		for i, member := range obj.Members {
			if member.Name.Value.(hujson.Literal).String() == "packageManager" {
				obj.Members = slices.Delete(obj.Members, i, i+1)
				break
			}
		}
	}
	if scripts, ok := pkg["scripts"].(map[string]any); ok {
		scriptObj := doc.Find("/scripts").Value.(*hujson.Object)
		for i := range scriptObj.Members {
			member := &scriptObj.Members[i]
			key := member.Name.Value.(hujson.Literal).String()
			if script, ok := scripts[key].(string); ok && script != member.Value.Value.(hujson.Literal).String() {
				member.Value.Value = hujson.String(script)
			}
		}
	}
	// Removing the last member can expose a trailing comma in hujson's representation.
	doc.Standardize()
	return doc.Pack(), nil
}

// setJSONField preserves existing whitespace and uses adjacent fields' spacing for insertions.
func setJSONField(obj *hujson.Object, name, value string) {
	for i := range obj.Members {
		member := &obj.Members[i]
		if member.Name.Value.(hujson.Literal).String() == name {
			if literal, ok := member.Value.Value.(hujson.Literal); !ok || literal.Kind() != '"' || literal.String() != value {
				member.Value.Value = hujson.String(value)
			}
			return
		}
	}
	member := hujson.ObjectMember{
		Name:  hujson.Value{Value: hujson.String(name)},
		Value: hujson.Value{Value: hujson.String(value)},
	}
	if len(obj.Members) > 0 {
		previous := obj.Members[len(obj.Members)-1]
		member.Name.BeforeExtra = bytes.Clone(previous.Name.BeforeExtra)
		member.Value.BeforeExtra = bytes.Clone(previous.Value.BeforeExtra)
	} else if bytes.ContainsRune(obj.AfterExtra, '\n') {
		member.Name.BeforeExtra = append(bytes.Clone(obj.AfterExtra), ' ', ' ')
		member.Value.BeforeExtra = []byte(" ")
	}
	obj.Members = append(obj.Members, member)
}
