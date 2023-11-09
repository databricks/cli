package databrickscfg

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"golang.org/x/mod/semver"
)

var minUcRuntime = canonicalVersion("v12.0")

var dbrVersionRegex = regexp.MustCompile(`^(\d+\.\d+)\.x-.*`)
var dbrSnapshotVersionRegex = regexp.MustCompile(`^(\d+)\.x-snapshot.*`)

func canonicalVersion(v string) string {
	return semver.Canonical("v" + strings.TrimPrefix(v, "v"))
}

func GetRuntimeVersion(cluster *compute.ClusterDetails) (string, bool) {
	match := dbrVersionRegex.FindStringSubmatch(cluster.SparkVersion)
	if len(match) < 1 {
		match = dbrSnapshotVersionRegex.FindStringSubmatch(cluster.SparkVersion)
		if len(match) > 1 {
			// we return 14.0 for 14.x-snapshot for semver.Compare() to work
			return fmt.Sprintf("%s.0", match[1]), true
		}
		return "", false
	}
	return match[1], true
}

func IsCompatibleWithUC(cluster *compute.ClusterDetails, minVersion string) bool {
	minVersion = canonicalVersion(minVersion)
	if semver.Compare(minUcRuntime, minVersion) >= 0 {
		return false
	}
	runtimeVersion, ok := GetRuntimeVersion(cluster)
	if !ok {
		return false
	}
	clusterRuntime := canonicalVersion(runtimeVersion)
	if semver.Compare(minVersion, clusterRuntime) > 0 {
		return false
	}
	switch cluster.DataSecurityMode {
	case compute.DataSecurityModeUserIsolation, compute.DataSecurityModeSingleUser:
		return true
	default:
		return false
	}
}

var ErrNoCompatibleClusters = errors.New("no compatible clusters found")

type compatibleCluster struct {
	compute.ClusterDetails
	versionName string
}

func (v compatibleCluster) Access() string {
	switch v.DataSecurityMode {
	case compute.DataSecurityModeUserIsolation:
		return "Shared"
	case compute.DataSecurityModeSingleUser:
		return "Assigned"
	default:
		return "Unknown"
	}
}

func (v compatibleCluster) Runtime() string {
	runtime, _, _ := strings.Cut(v.versionName, " (")
	return runtime
}

func (v compatibleCluster) State() string {
	state := v.ClusterDetails.State
	switch state {
	case compute.StateRunning, compute.StateResizing:
		return color.GreenString(state.String())
	case compute.StateError, compute.StateTerminated, compute.StateTerminating, compute.StateUnknown:
		return color.RedString(state.String())
	default:
		return color.BlueString(state.String())
	}
}

func loadClustersCompatibleWithUC(ctx context.Context, w *databricks.WorkspaceClient, minVersion string) ([]compatibleCluster, error) {
	promptSpinner := cmdio.Spinner(ctx)
	promptSpinner <- "Loading list of clusters to select from"
	defer close(promptSpinner)
	all, err := w.Clusters.ListAll(ctx, compute.ListClustersRequest{
		CanUseClient: "NOTEBOOKS",
	})
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	me, err := w.CurrentUser.Me(ctx)
	if err != nil {
		return nil, fmt.Errorf("current user: %w", err)
	}
	versions := map[string]string{}
	sv, err := w.Clusters.SparkVersions(ctx)
	if err != nil {
		return nil, fmt.Errorf("list runtime versions: %w", err)
	}
	for _, v := range sv.Versions {
		versions[v.Key] = v.Name
	}
	var compatible []compatibleCluster
	for _, v := range all {
		if !IsCompatibleWithUC(&v, minVersion) {
			continue
		}
		switch v.ClusterSource {
		case compute.ClusterSourceJob,
			compute.ClusterSourceModels,
			compute.ClusterSourcePipeline,
			compute.ClusterSourcePipelineMaintenance,
			compute.ClusterSourceSql:
			// only UI and API clusters are usable for DBConnect.
			// `CanUseClient: "NOTEBOOKS"`` didn't seem to have an effect.
			continue
		}
		if v.SingleUserName != "" && v.SingleUserName != me.UserName {
			continue
		}
		compatible = append(compatible, compatibleCluster{
			ClusterDetails: v,
			versionName:    versions[v.SparkVersion],
		})
	}
	return compatible, nil
}

func AskForClusterCompatibleWithUC(ctx context.Context, w *databricks.WorkspaceClient, minVersion string) (string, error) {
	compatible, err := loadClustersCompatibleWithUC(ctx, w, minVersion)
	if err != nil {
		return "", fmt.Errorf("load: %w", err)
	}
	if len(compatible) == 0 {
		return "", ErrNoCompatibleClusters
	}
	if len(compatible) == 1 {
		return compatible[0].ClusterId, nil
	}
	i, _, err := cmdio.RunSelect(ctx, &promptui.Select{
		Label: "Choose compatible cluster",
		Items: compatible,
		Searcher: func(input string, idx int) bool {
			lower := strings.ToLower(compatible[idx].ClusterName)
			return strings.Contains(lower, input)
		},
		StartInSearchMode: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{.ClusterName | faint}}",
			Active:   `{{.ClusterName | bold}} ({{.State}} {{.Access}} Runtime {{.Runtime}}) ({{.ClusterId | faint}})`,
			Inactive: `{{.ClusterName}} ({{.State}} {{.Access}} Runtime {{.Runtime}})`,
			Selected: `{{ "Configured cluster" | faint }}: {{ .ClusterName | bold }} ({{.ClusterId | faint}})`,
		},
	})
	if err != nil {
		return "", err
	}
	return compatible[i].ClusterId, nil
}
