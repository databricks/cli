// Command skillssnapshot fills the bundled skills snapshot that libs/skills
// embeds. The release build runs it as a goreleaser before hook, so the binaries
// it compiles carry the skills release that is current when they are built.
// Development builds do not run it.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

// snapshotDir is the folder libs/skills embeds. The command runs from the
// repository root.
const snapshotDir = "libs/skills/snapshot"

func main() {
	ref := flag.String("ref", "", "skills release tag to bundle (default: the latest release)")
	flag.Parse()

	bundled, err := fillSnapshot(context.Background(), snapshotDir, *ref)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fill skills snapshot:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Bundled skills release %s in %s\n", bundled, snapshotDir)
}
