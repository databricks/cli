package main

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/clijson"
)

func TestBindReport(t *testing.T) {
	idx, err := loadSdkIndex("../..")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := clijson.Parse("../../.codegen/cli.json")
	if err != nil {
		t.Fatal(err)
	}
	b := fromContract(doc.Commands)
	ok, miss := 0, 0
	for _, s := range b.Services {
		for _, m := range s.Methods {
			sm, consts := idx.match(s, m)
			if sm == nil {
				miss++
				fmt.Printf("MISS %s.%s %s %s %s\n", s.Package.Name, s.Name, m.Name, m.Path, methodVerb(s, m))
				continue
			}
			ok++
			fmt.Printf("OK %s.%s %s -> %s.%s(%s) %s %s %v\n", s.Package.Name, s.Name, m.Name, sm.Pkg.Rel, sm.Name, sm.Request, sm.Kind, sm.verb, consts)
		}
	}
	fmt.Println("ok", ok, "miss", miss)
}
