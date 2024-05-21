package main

import (
	"context"

	"github.com/databricks/cli/cmd"
	"golang.org/x/exp/maps"
)

type Package struct {
	Name   string
	Groups []*Group
}

func Packages() []Package {
	root := cmd.New(context.Background())
	packages := make(map[string]Package)
	for _, c := range root.Commands() {
		pkg := c.Annotations["package"]
		if pkg == "" {
			continue
		}

		g := Find(c.Use)
		p, ok := packages[pkg]
		if !ok {
			p = Package{
				Name:   pkg,
				Groups: []*Group{g},
			}
		} else {
			p.Groups = append(p.Groups, g)
		}

		packages[pkg] = p
	}

	return maps.Values(packages)
}
