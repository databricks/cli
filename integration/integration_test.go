package integration_test

import (
	"flag"
)

var SkipLocal bool

func init() {
	// This flag is a noop to match acceptance test compatibility.
	// It ensures the -skiplocal option works when running integration tests.
	flag.BoolVar(&SkipLocal, "skiplocal", false, "Skip tests that are enabled to run on Local")
}
