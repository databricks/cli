package phases

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
)

func applyBindPhase(ctx context.Context, b *bundle.Bundle, bindings map[string]*deployplan.BindEntry, current *bundledeployments.Deployment) *bundledeployments.Deployment {
	if b.MigratingToDirect {
		statemgmt.CommitMigration(ctx, b)
		if logdiag.HasError(ctx) {
			return current
		}
		b.MigratingToDirect = false
	}
	db := &b.DeploymentBundle.StateDB
	statePath := db.Path
	mode := dstate.ReadingFile
	if db.IsDeploymentMetadataService() {
		mode = dstate.WritingDeploymentHistoryOnly
	}
	if err := db.UpgradeToWrite(); err != nil {
		logdiag.LogError(ctx, err)
		return current
	}
	if db.RecordsHistory() {
		var err error
		current, err = prepareBindHistory(ctx, b, bindings, current)
		if err != nil {
			if db.IsDeploymentMetadataService() {
				logdiag.LogError(ctx, err)
			} else {
				log.Warnf(ctx, "Could not record binding history: %s", err)
			}
		}
	}
	if !logdiag.HasError(ctx) {
		if err := db.Bind(ctx, bindings); err != nil {
			logdiag.LogError(ctx, err)
		}
	}
	state, err := db.Finalize(ctx)
	b.Metrics.ResourceState = state
	if err != nil {
		logdiag.LogError(ctx, err)
	}
	statemgmt.PushResourcesState(ctx, b)
	if _, err := db.CompleteVersion(ctx, !logdiag.HasError(ctx)); err != nil {
		if db.IsDeploymentMetadataService() {
			logdiag.LogError(ctx, err)
		} else {
			log.Warnf(ctx, "Failed to complete binding history version: %s", err)
		}
	}
	if logdiag.HasError(ctx) {
		return current
	}
	if reportPerResource(b) {
		for _, key := range slices.Sorted(maps.Keys(bindings)) {
			cmdio.LogString(ctx, "Bound "+strings.TrimPrefix(key, "resources."))
		}
	}
	history := dstate.OpenDmsArgs{DeploymentID: db.DeploymentID, LastVersionID: db.VersionID}
	if err := db.Open(ctx, statePath, mode, dstate.WithRecovery(true), history); err != nil {
		logdiag.LogError(ctx, err)
		return current
	}
	if db.RecordsHistory() {
		bundle.ApplyContext(ctx, b, metadata.AnnotateDeploymentVersion(db.GetSerial()+1))
		if db.DeploymentID != "" {
			bundle.ApplyContext(ctx, b, metadata.AnnotateDeployment(db.DeploymentID))
		}
	}
	return current
}

func prepareBindHistory(ctx context.Context, b *bundle.Bundle, bindings map[string]*deployplan.BindEntry, current *bundledeployments.Deployment) (*bundledeployments.Deployment, error) {
	current, err := createOrUpdateDeployment(ctx, b, current)
	if err != nil {
		return current, err
	}
	staged := make([]bundledeployments.StagedOperation, 0, len(bindings))
	for _, key := range slices.Sorted(maps.Keys(bindings)) {
		staged = append(staged, bundledeployments.StagedOperation{
			ResourceKey: strings.TrimPrefix(key, dms.StatePrefix),
			ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
		})
	}
	if err := startVersion(ctx, b, dms.VersionTypeDeploy, staged); err != nil {
		return current, err
	}
	logDeploymentVersion(ctx, b)
	return current, nil
}
