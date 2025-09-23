package workspace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
)

const metadataFileName = "metadata.json"

type WorkspaceMetadata struct {
	Port int `json:"port"`
}

func getWorkspaceRootDir(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
	me, err := client.CurrentUser.Me(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return fmt.Sprintf("/Workspace/Users/%s/.databricks/ssh-tunnel", me.UserName), nil
}

func GetWorkspaceVersionedDir(ctx context.Context, client *databricks.WorkspaceClient, version string) (string, error) {
	contentDir, err := getWorkspaceRootDir(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace root directory: %w", err)
	}
	return filepath.ToSlash(filepath.Join(contentDir, version)), nil
}

func GetWorkspaceContentDir(ctx context.Context, client *databricks.WorkspaceClient, version, clusterID string) (string, error) {
	contentDir, err := GetWorkspaceVersionedDir(ctx, client, version)
	if err != nil {
		return "", fmt.Errorf("failed to get versioned workspace directory: %w", err)
	}
	return filepath.ToSlash(filepath.Join(contentDir, clusterID)), nil
}

func GetWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, clusterID string) (int, error) {
	contentDir, err := GetWorkspaceContentDir(ctx, client, version, clusterID)
	if err != nil {
		return 0, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	metadataPath := filepath.ToSlash(filepath.Join(contentDir, metadataFileName))

	content, err := client.Workspace.Download(ctx, metadataPath)
	if err != nil {
		return 0, fmt.Errorf("failed to download metadata file: %w", err)
	}
	defer content.Close()

	metadataBytes, err := io.ReadAll(content)
	if err != nil {
		return 0, fmt.Errorf("failed to read metadata content: %w", err)
	}

	var metadata WorkspaceMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return 0, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return metadata.Port, nil
}

func SaveWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, clusterID string, port int) error {
	metadataBytes, err := json.Marshal(WorkspaceMetadata{Port: port})
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	contentDir, err := GetWorkspaceContentDir(ctx, client, version, clusterID)
	if err != nil {
		return fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	workspaceFiler, err := filer.NewWorkspaceFilesClient(client, contentDir)
	if err != nil {
		return fmt.Errorf("failed to create workspace files client: %w", err)
	}

	err = workspaceFiler.Write(ctx, metadataFileName, bytes.NewReader(metadataBytes), filer.OverwriteIfExists, filer.CreateParentDirectories)
	if err != nil {
		return fmt.Errorf("failed to write metadata file: %w", err)
	}
	return nil
}
