package metadata

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

type convertStateForYamlSync struct {
	engine engine.EngineType
}

// ConvertStateForYamlSync converts the state to the direct format for YAML sync.
// This is simplified version of the `bundle migrate` command. State file is saved in the same format as the direct engine but to the different path.
func ConvertStateForYamlSync(targetEngine engine.EngineType) bundle.Mutator {
	return &convertStateForYamlSync{engine: targetEngine}
}

func (m *convertStateForYamlSync) Name() string {
	return "metadata.ConvertStateForYamlSync"
}

func (m *convertStateForYamlSync) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	if m.engine.IsDirect() {
		return nil
	}

	_, enabled := env.ExperimentalYamlSync(ctx)
	if !enabled {
		return nil
	}

	_, snapshotPath := b.StateFilenameConfigSnapshot(ctx)

	err := m.convertState(ctx, b, snapshotPath)
	if err != nil {
		log.Warnf(ctx, "Failed to create config snapshot: %v", err)
		return nil
	}

	log.Infof(ctx, "Config snapshot created at %s", snapshotPath)
	return nil
}

func (m *convertStateForYamlSync) convertState(ctx context.Context, b *bundle.Bundle, snapshotPath string) error {
	terraformResources, err := terraform.ParseResourcesState(ctx, b)
	if err != nil {
		return err
	}

	state := make(map[string]dstate.ResourceEntry)
	etags := map[string]string{}

	for key, resourceEntry := range terraformResources {
		state[key] = dstate.ResourceEntry{
			ID:    resourceEntry.ID,
			State: json.RawMessage("{}"),
		}
		if resourceEntry.ETag != "" {
			etags[key] = resourceEntry.ETag
		}
	}

	_, localTerraformPath := b.StateFilenameTerraform(ctx)
	data, err := os.ReadFile(localTerraformPath)
	if err != nil {
		return err
	}

	var tfState struct {
		Lineage string `json:"lineage"`
		Serial  int    `json:"serial"`
	}
	if err := json.Unmarshal(data, &tfState); err != nil {
		return err
	}

	migratedDB := dstate.NewDatabase(tfState.Lineage, tfState.Serial+1)
	migratedDB.State = state

	deploymentBundle := &direct.DeploymentBundle{
		StateDB: dstate.DeploymentState{
			Path: snapshotPath,
			Data: migratedDB,
		},
	}

	// Get the dynamic value from b.Config and reverse the interpolation
	// b.Config has been modified by terraform.Interpolate which converts bundle-style
	// references (${resources.pipelines.x.id}) to terraform-style (${databricks_pipeline.x.id})
	interpolatedRoot := b.Config.Value()
	uninterpolatedRoot, err := reverseInterpolate(interpolatedRoot)
	if err != nil {
		return fmt.Errorf("failed to reverse interpolation: %w", err)
	}

	var uninterpolatedConfig config.Root
	err = uninterpolatedConfig.Mutate(func(_ dyn.Value) (dyn.Value, error) {
		return uninterpolatedRoot, nil
	})
	if err != nil {
		return fmt.Errorf("failed to create uninterpolated config: %w", err)
	}

	plan, err := deploymentBundle.CalculatePlan(ctx, b.WorkspaceClient(), &uninterpolatedConfig, snapshotPath)
	if err != nil {
		return err
	}

	for _, entry := range plan.Plan {
		entry.Action = deployplan.Update
	}

	for key := range plan.Plan {
		etag := etags[key]
		if etag == "" {
			continue
		}
		sv, ok := deploymentBundle.StructVarCache.Load(key)
		if !ok {
			continue
		}
		err := structaccess.Set(sv.Value, structpath.NewStringKey(nil, "etag"), etag)
		if err != nil {
			log.Warnf(ctx, "Failed to set etag on %q: %v", key, err)
		}
	}

	deploymentBundle.Apply(ctx, b.WorkspaceClient(), &uninterpolatedConfig, plan, direct.MigrateMode(true))

	return nil
}
