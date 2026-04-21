package terraform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/dyn/jsonsaver"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/terraform/tfdyn"
)

// Render converts the ucm config tree into a Terraform JSON resource map
// and writes it to <workingDir>/main.tf.json. The conversion itself lives
// in tfdyn (U3); Render is the thin filesystem adapter.
//
// The output is indented JSON for readability. Keys appear in insertion
// order — libs/dyn/jsonsaver preserves the dyn.Value key order set by the
// tfdyn walkers so diffs between Render runs are minimal.
func (t *Terraform) Render(ctx context.Context, u *ucm.Ucm) error {
	if t == nil {
		return fmt.Errorf("terraform: nil wrapper")
	}

	tree, err := tfdyn.Convert(ctx, u)
	if err != nil {
		return fmt.Errorf("convert ucm to terraform: %w", err)
	}

	data, err := jsonsaver.MarshalIndent(tree, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal terraform json: %w", err)
	}

	outPath := filepath.Join(t.WorkingDir, MainConfigFileName)
	if err := os.WriteFile(outPath, data, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	log.Infof(ctx, "Rendered terraform config at %s", filepath.ToSlash(outPath))
	return nil
}
