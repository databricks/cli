package internal

import (
	"fmt"
	"os"
	"testing"
)

// Main is the entry point for integration tests.
// We use this for all integration tests defined in this subtree to ensure
// they are not inadvertently executed when calling `go test ./...`.
func Main(m *testing.M) {
	value := os.Getenv("CLOUD_ENV")
	if value == "" {
		fmt.Println("CLOUD_ENV is not set, skipping integration tests")
		return
	}

	m.Run()
}
