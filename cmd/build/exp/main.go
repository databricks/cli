package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/databricks/bricks/lib/ui"
)

func main() {
	ctx := context.Background()
	err := ui.SpinStages(ctx, []ui.Stage{
		{InProgress: "Building", Callback: func(ctx context.Context, status func(string)) error {
			time.Sleep(1 * time.Second)
			status("first message")
			time.Sleep(1 * time.Second)
			status("second message")
			time.Sleep(1 * time.Second)
			return nil
		}, Complete: "Built!"},
		{InProgress: "Uploading", Callback: func(ctx context.Context, status func(string)) error {
			status("third message")
			time.Sleep(1 * time.Second)
			return nil
		}, Complete: "Uploaded!"},
		{InProgress: "Installing", Callback: func(ctx context.Context, status func(string)) error {
			time.Sleep(1 * time.Second)
			return fmt.Errorf("nope")
		}, Complete: "Installed!"},
	})
	if err != nil {
		os.Exit(1)
	}
}
