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
	"github.com/databricks/databricks-sdk-go/service/iam"
)

const metadataFileName = "metadata.json"

// Interactive SSH sessions set HOME to the user's workspace folder (see
// buildRemoteShellArgs). Everything the tunnel seeds for those sessions lives under
// HOME/.config, so the workspace user-folder root stays uncluttered and the files
// persist and are visible in the workspace file browser. These are paths relative to
// HOME; the client references them as $HOME/<path> and the server as <wsHome>/<path>.
const (
	SessionConfigDir  = ".config"              // XDG_CONFIG_HOME
	SessionCacheDir   = ".config/cache"        // XDG_CACHE_HOME
	SessionIPythonDir = ".config/ipython"      // IPYTHONDIR
	SessionHistFile   = ".config/bash_history" // HISTFILE
	SessionBashrc     = ".config/bashrc"       // bash --rcfile
)

// OSHomeEnvVar carries the compute's OS home path into the interactive session. The
// client captures $HOME into it before redirecting HOME to the workspace folder (see
// buildRemoteShellArgs), so the seeded rcfile can source the OS-home ~/.bashrc for the
// per-compute config placed there. It is resolved live per session rather than baked
// into the rcfile, which persists on the workspace mount and is reused across clusters
// where the OS-home path differs.
const OSHomeEnvVar = "DATABRICKS_OS_HOME"

type WorkspaceMetadata struct {
	Port int `json:"port"`
	// ClusterID is required for Driver Proxy websocket connections (for any compute type, including serverless)
	ClusterID string `json:"cluster_id,omitempty"`
}

// UserWorkspaceHome returns the current user's workspace home folder
// (/Workspace/Users/<email>), used as HOME for interactive SSH sessions.
func UserWorkspaceHome(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
	me, err := client.CurrentUser.Me(ctx, iam.MeRequest{})
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	return "/Workspace/Users/" + me.UserName, nil
}

func getWorkspaceRootDir(ctx context.Context, client *databricks.WorkspaceClient) (string, error) {
	home, err := UserWorkspaceHome(ctx, client)
	if err != nil {
		return "", err
	}
	return home + "/.databricks/ssh-tunnel", nil
}

func GetWorkspaceVersionedDir(ctx context.Context, client *databricks.WorkspaceClient, version string) (string, error) {
	contentDir, err := getWorkspaceRootDir(ctx, client)
	if err != nil {
		return "", fmt.Errorf("failed to get workspace root directory: %w", err)
	}
	return filepath.ToSlash(filepath.Join(contentDir, version)), nil
}

// GetWorkspaceContentDir returns the directory for storing session content.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
func GetWorkspaceContentDir(ctx context.Context, client *databricks.WorkspaceClient, version, sessionID string) (string, error) {
	contentDir, err := GetWorkspaceVersionedDir(ctx, client, version)
	if err != nil {
		return "", fmt.Errorf("failed to get versioned workspace directory: %w", err)
	}
	return filepath.ToSlash(filepath.Join(contentDir, sessionID)), nil
}

// GetWorkspaceMetadata loads session metadata from the workspace.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
func GetWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, sessionID string) (*WorkspaceMetadata, error) {
	contentDir, err := GetWorkspaceContentDir(ctx, client, version, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workspace content directory: %w", err)
	}

	metadataPath := filepath.ToSlash(filepath.Join(contentDir, metadataFileName))

	content, err := client.Workspace.Download(ctx, metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to download metadata file: %w", err)
	}
	defer content.Close()

	metadataBytes, err := io.ReadAll(content)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata content: %w", err)
	}

	var metadata WorkspaceMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	return &metadata, nil
}

// SaveWorkspaceMetadata saves session metadata to the workspace.
// sessionID is the unique identifier for the session (cluster ID for dedicated clusters, connection name for serverless).
func SaveWorkspaceMetadata(ctx context.Context, client *databricks.WorkspaceClient, version, sessionID string, metadata *WorkspaceMetadata) error {
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	contentDir, err := GetWorkspaceContentDir(ctx, client, version, sessionID)
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
