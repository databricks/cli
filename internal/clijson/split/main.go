// Command split shards the cli.json document genkit writes into the layout
// clijson.ReadDir reassembles.
//
// Usage: go run ./internal/clijson/split <cli.json> <dir>
package main

import (
	"log"
	"os"

	"github.com/databricks/cli/internal/clijson"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: split <cli.json> <dir>")
	}
	doc, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("split: %v", err)
	}
	if err := clijson.Split(doc, os.Args[2]); err != nil {
		log.Fatalf("split: %v", err)
	}
}
