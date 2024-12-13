package internal

import (
	"fmt"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// 1. Redefine env vars to avoid accidentally picking up user's credentials
	// 2. TESTS_DATABRICKS_TOKEN is also a gate enabling integration tests
	host := os.Getenv("TESTS_DATABRICKS_HOST")
	if host == "" {
		fmt.Println("TESTS_DATABRICKS_HOST is not set, skipping integration tests")
		return
	}
	token := os.Getenv("TESTS_DATABRICKS_TOKEN")

	os.Setenv("DATABRICKS_HOST", host)
	os.Setenv("DATABRICKS_TOKEN", token)

	m.Run()
}
