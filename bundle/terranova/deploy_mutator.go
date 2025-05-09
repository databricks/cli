package terranova

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/terranova/db"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/jsonloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
)

type terranovaDeployMutator struct{}

func TerranovaDeploy() bundle.Mutator {
	return &terranovaDeployMutator{}
}

func (m *terranovaDeployMutator) Name() string {
	return "TerranovaDeploy"
}

func (m *terranovaDeployMutator) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	client := b.WorkspaceClient()

	cacheDir, err := b.CacheDir(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	b.ResourceDatabase, err = db.LoadOrCreate(cacheDir)
	if err != nil {
		return diag.FromErr(err)
	}

	countDeployed := 0

	_, err = dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			section := p[1].Key()
			resourceName := p[2].Key()

			spec, ok := specs[section]
			if !ok {
				log.Warnf(ctx, "Resource section not supported: %s", section)
				return v, nil
			}

			err = deployResource(ctx, b.ResourceDatabase, client, spec, section, resourceName, v)
			if err != nil {
				diags = diags.Extend(diag.FromErr(err))
			}
			countDeployed += 1
			return v, nil
		},
	)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	// Not uploading at the moment, just logging to match the output.
	if countDeployed > 0 {
		cmdio.LogString(ctx, "Updating deployment state...")
	}

	return diags
}

func deployResource(ctx context.Context, db *db.JSONFileDB, client *databricks.WorkspaceClient, spec IResource, section, resourceName string, config dyn.Value) error {
	databaseResourceID, err := db.GetResourceID(ctx, section, resourceName)
	if err != nil {
		return err
	}

	config, err = spec.PreprocessConfig(config)
	if err != nil {
		return err
	}

	configBytes, err := json.Marshal(config.AsAny())
	if err != nil {
		return err
	}

	configString := string(configBytes)

	log.Infof(ctx, "Deploying %s.%s: %s", section, resourceName, configString)

	if databaseResourceID == "" {
		resourceID, err := spec.ExtractIDFromConfig(config)
		if err != nil {
			return err
		}

		// Note, resourceID can be empty for cases when resourceID is returned by Create call

		err = db.InsertResourcePre(ctx, section, resourceName, resourceID, configString)
		if err != nil {
			return err
		}

		responseResourceID, err := spec.DoCreate(ctx, resourceID, config, client)
		if err != nil {
			return err
		}

		if resourceID == "" && responseResourceID != "" {
			resourceID = responseResourceID
		} else if resourceID != "" && responseResourceID != "" && resourceID != responseResourceID {
			return errors.New("Internal error")
		}

		log.Infof(ctx, "Created %s.%s with identifier %#v", section, resourceName, resourceID)

		return db.FinalizeResource(ctx, section, resourceName, resourceID)
	} else {
		needUpdate, configOld, err := calculateNeedUpdate(ctx, db, section, resourceName, configString)
		if !needUpdate {
			log.Infof(ctx, "Up-to-date %s.%s with identifier %#v", section, resourceName, databaseResourceID)
			return nil
		}

		if err != nil {
			return err
		}

		err = db.UpdateResourcePre(ctx, section, resourceName, databaseResourceID, configString)
		if err != nil {
			return err
		}

		err = spec.DoUpdate(ctx, databaseResourceID, configOld, config, client)
		if err != nil {
			return err
		}

		log.Infof(ctx, "Updated %s.%s with identifier %#v", section, resourceName, databaseResourceID)

		return db.FinalizeResource(ctx, section, resourceName, databaseResourceID)
	}
}

func calculateNeedUpdate(ctx context.Context, db *db.JSONFileDB, section, resourceName, config string) (bool, dyn.Value, error) {
	// TODO: calculate the need to re-create
	row, ok := db.GetRow(ctx, section, resourceName)
	if !ok {
		return false, dyn.InvalidValue, fmt.Errorf("Internal error: resource %s.%s not found in resources database", section, resourceName)
	}

	if row.Config != config {
		oldConfig, err := jsonloader.LoadJSON([]byte(row.Config), "")
		if err != nil {
			return true, dyn.InvalidValue, fmt.Errorf("failed to decode old config: %w", err)
		}
		return true, oldConfig, nil
	}

	return false, dyn.Value{}, nil
}
