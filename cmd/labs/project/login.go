package project

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/cfgpickers"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
)

type loginConfig struct {
	*Entrypoint      `json:"-"`
	WorkspaceProfile string `json:"workspace_profile,omitempty"`
	AccountProfile   string `json:"account_profile,omitempty"`
	ClusterID        string `json:"cluster_id,omitempty"`
	WarehouseID      string `json:"warehouse_id,omitempty"`
}

func (lc *loginConfig) askWorkspace(ctx context.Context, cfg *config.Config) (*databricks.WorkspaceClient, error) {
	if cfg.IsAccountClient() {
		return nil, nil
	}
	err := lc.askWorkspaceProfile(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("profile: %w", err)
	}
	w, err := databricks.NewWorkspaceClient((*databricks.Config)(cfg))
	if err != nil {
		return nil, fmt.Errorf("client: %w", err)
	}
	err = lc.askCluster(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("cluster: %w", err)
	}
	err = lc.askWarehouse(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("warehouse: %w", err)
	}
	return w, nil
}

func (lc *loginConfig) askWorkspaceProfile(ctx context.Context, cfg *config.Config) (err error) {
	if cfg.Profile != "" {
		lc.WorkspaceProfile = cfg.Profile
		return
	}
	if !cmdio.IsPromptSupported(ctx) {
		return ErrNotInTTY
	}
	lc.WorkspaceProfile, err = root.AskForWorkspaceProfile(ctx)
	cfg.Profile = lc.WorkspaceProfile
	return
}

func (lc *loginConfig) askCluster(ctx context.Context, w *databricks.WorkspaceClient) (err error) {
	if !lc.NeedsCluster() {
		return
	}
	if w.Config.ClusterID != "" {
		lc.ClusterID = w.Config.ClusterID
		return
	}
	if !cmdio.IsPromptSupported(ctx) {
		return ErrNotInTTY
	}
	clusterID, err := cfgpickers.AskForCluster(ctx, w,
		cfgpickers.WithDatabricksConnect(lc.Installer.MinRuntimeVersion))
	if err != nil {
		return fmt.Errorf("select: %w", err)
	}
	w.Config.ClusterID = clusterID
	lc.ClusterID = clusterID
	return
}

func (lc *loginConfig) askWarehouse(ctx context.Context, w *databricks.WorkspaceClient) (err error) {
	if !lc.NeedsWarehouse() {
		return
	}
	if w.Config.WarehouseID != "" {
		lc.WarehouseID = w.Config.WarehouseID
		return
	}
	if !cmdio.IsPromptSupported(ctx) {
		return ErrNotInTTY
	}
	lc.WarehouseID, err = cfgpickers.AskForWarehouse(ctx, w,
		cfgpickers.WithWarehouseTypes(lc.Installer.WarehouseTypes...))
	return
}

func (lc *loginConfig) askAccountProfile(ctx context.Context, cfg *config.Config) (err error) {
	if !lc.HasAccountLevelCommands() {
		return nil
	}
	if !cmdio.IsPromptSupported(ctx) {
		return ErrNotInTTY
	}
	lc.AccountProfile, err = root.AskForAccountProfile(ctx)
	cfg.Profile = lc.AccountProfile
	return
}

func (lc *loginConfig) save(ctx context.Context) error {
	authFile := lc.loginFile(ctx)
	raw, err := json.MarshalIndent(lc, "", "  ")
	if err != nil {
		return err
	}
	log.Debugf(ctx, "Writing auth configuration to: %s", authFile)
	return os.WriteFile(authFile, raw, ownerRW)
}
