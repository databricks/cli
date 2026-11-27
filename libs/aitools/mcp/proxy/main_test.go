package proxy

import (
	"os"
	"testing"

	"github.com/databricks/cli/internal/testutil/hostenv"
)

func TestMain(m *testing.M) {
	finish := hostenv.Isolate()
	os.Exit(finish(m.Run()))
}
