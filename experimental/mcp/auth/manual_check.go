//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/experimental/mcp/auth"
)

func main() {
	ctx := context.Background()

	fmt.Println("Testing CheckAuthentication...")
	fmt.Println("================================")

	// Test 1: Without skip env var (real check)
	fmt.Println("\n1. Testing without DATABRICKS_MCP_SKIP_AUTH_CHECK:")
	os.Unsetenv("DATABRICKS_MCP_SKIP_AUTH_CHECK")
	err := auth.CheckAuthentication(ctx)
	if err != nil {
		fmt.Printf("   Error (expected if not authenticated): %v\n", err)
	} else {
		fmt.Println("   ✓ Authentication check passed")
	}

	// Test 2: With skip env var
	fmt.Println("\n2. Testing with DATABRICKS_MCP_SKIP_AUTH_CHECK=1:")
	os.Setenv("DATABRICKS_MCP_SKIP_AUTH_CHECK", "1")
	err = auth.CheckAuthentication(ctx)
	if err != nil {
		fmt.Printf("   ✗ Unexpected error: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Println("   ✓ Skip check worked correctly")
	}

	fmt.Println("\n================================")
	fmt.Println("Manual test completed!")
}
