package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"
	"github.com/felixge/fgprof"
)

func main() {
	// Start CPU profiling
	f, err := os.Create("wall_clock.pprof")
	if err != nil {
		panic(err)
	}

	stop := fgprof.Start(f, fgprof.FormatPprof)

	start := time.Now()

	ctx := context.Background()
	err = root.Execute(ctx, cmd.New(ctx))
	if err != nil {
		os.Exit(1)
	}

	fmt.Println("Execution time:", time.Since(start))

	// Stop CPU profiling and write the profile to disk
	err = stop()
	if err != nil {
		panic(err)
	}
}
