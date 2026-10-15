package cfgpickers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/sdkclients"
	mirror "github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"github.com/databricks/cli/libs/utils"
	compute "github.com/databricks/sdk-go/clusters/v2"
	"github.com/databricks/sdk-go/scim/v1"
	"golang.org/x/mod/semver"
)

var minUcRuntime = canonicalVersion("v12.0")

var (
	dbrVersionRegex         = regexp.MustCompile(`^(\d+(?:\.\d+)?)\.x-.*`)
	dbrSnapshotVersionRegex = regexp.MustCompile(`^(\d+)\.x-snapshot.*`)
)

func canonicalVersion(v string) string {
	return semver.Canonical("v" + strings.TrimPrefix(v, "v"))
}

func GetRuntimeVersion(cluster compute.ClusterInfo) (string, bool) {
	// Snapshots are checked first: dbrVersionRegex now matches major-only versions
	// like "14.x-...", which would otherwise swallow "14.x-snapshot-..." here.
	if match := dbrSnapshotVersionRegex.FindStringSubmatch(utils.Deref(cluster.SparkVersion)); len(match) > 1 {
		// we return 14.999 for 14.x-snapshot for semver.Compare() to work properly
		return match[1] + ".999", true
	}
	if match := dbrVersionRegex.FindStringSubmatch(utils.Deref(cluster.SparkVersion)); len(match) > 1 {
		return match[1], true
	}
	return "", false
}

func IsCompatibleWithUC(cluster compute.ClusterInfo, minVersion string) bool {
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
	case compute.DataSecurityMode_UserIsolation, compute.DataSecurityMode_SingleUser:
		return true
	default:
		return false
	}
}

var ErrNoCompatibleClusters = errors.New("no compatible clusters found")

type compatibleCluster struct {
	compute.ClusterInfo
	versionName string
	// renderedState caches the colorized ClusterDetails.State for display in
	// SelectOptions templates, which can't access ctx-bound color helpers.
	renderedState string
}

func (v compatibleCluster) Access() string {
	switch v.DataSecurityMode {
	case compute.DataSecurityMode_UserIsolation:
		return "Shared"
	case compute.DataSecurityMode_SingleUser:
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
	return v.renderedState
}

type clusterFilter func(cluster *Cluster, me *scim.User) bool

// Filter function to check if a cluster is created through the UI or an API call.
// Cluster is a cluster as listed for the picker. sdk-go does not model
// cluster_source, which the picker filters on, so it is carried separately.
type Cluster struct {
	compute.ClusterInfo
	Source string
}

func isUsableCluster(cluster *Cluster) bool {
	return cluster.Source == "UI" || cluster.Source == "API"
}

func WithDatabricksConnect(minVersion string) func(*Cluster, *scim.User) bool {
	return func(cluster *Cluster, me *scim.User) bool {
		if !IsCompatibleWithUC(cluster.ClusterInfo, minVersion) {
			return false
		}
		// Only clusters created through the UI or an API call are usable for DBConnect.
		// `CanUseClient: "NOTEBOOKS"`` didn't seem to have an effect.
		if !isUsableCluster(cluster) {
			return false
		}
		if utils.Deref(cluster.SingleUserName) != "" && utils.Deref(cluster.SingleUserName) != utils.Deref(me.UserName) {
			return false
		}
		return true
	}
}

// WithoutSystemClusters removes clusters created for system purposes (e.g. job runs, pipeline maintenance, etc.).
// It does this by keeping only clusters created through the UI or an API call.
func WithoutSystemClusters() func(*Cluster, *scim.User) bool {
	return func(cluster *Cluster, me *scim.User) bool {
		return isUsableCluster(cluster)
	}
}

func loadInteractiveClusters(ctx context.Context, w *sdkclients.Clients, filters []clusterFilter) ([]compatibleCluster, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Loading list of clusters to select from")
	defer sp.Close()
	// Filters out system clusters, with the maximum page size to optimize for load time.
	all, err := listUserClusters(ctx, w)
	if err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}
	me, err := w.Scim.Me(ctx, scim.MeRequest{})
	if err != nil {
		return nil, fmt.Errorf("current user: %w", err)
	}
	versions := map[string]string{}
	sv, err := w.Clusters.ListSparkVersions(ctx, compute.GetSparkVersionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("list runtime versions: %w", err)
	}
	for _, v := range sv.Versions {
		versions[utils.Deref(v.Key)] = utils.Deref(v.Name)
	}
	var compatible []compatibleCluster
	for _, cluster := range all {
		var skip bool
		for _, filter := range filters {
			if !filter(&cluster, me) {
				skip = true
			}
		}
		if skip {
			continue
		}
		var renderedState string
		switch cluster.State {
		case compute.ClusterState_ClusterState_Running, compute.ClusterState_ClusterState_Resizing:
			renderedState = cmdio.Green(ctx, string(cluster.State))
		case compute.ClusterState_ClusterState_Error, compute.ClusterState_ClusterState_Terminated, compute.ClusterState_ClusterState_Terminating, compute.ClusterState_ClusterState_Unknown:
			renderedState = cmdio.Red(ctx, string(cluster.State))
		default:
			renderedState = cmdio.Blue(ctx, string(cluster.State))
		}
		compatible = append(compatible, compatibleCluster{
			ClusterInfo:   cluster.ClusterInfo,
			versionName:   versions[utils.Deref(cluster.SparkVersion)],
			renderedState: renderedState,
		})
	}
	return compatible, nil
}

func AskForCluster(ctx context.Context, w *sdkclients.Clients, filters ...clusterFilter) (string, error) {
	compatible, err := loadInteractiveClusters(ctx, w, filters)
	if err != nil {
		return "", fmt.Errorf("load: %w", err)
	}
	if len(compatible) == 0 {
		return "", ErrNoCompatibleClusters
	}
	if len(compatible) == 1 {
		return utils.Deref(compatible[0].ClusterId), nil
	}
	i, err := cmdio.RunSelect(ctx, cmdio.SelectOptions{
		Label: "Choose compatible cluster",
		Items: compatible,
		Searcher: func(input string, idx int) bool {
			lower := strings.ToLower(utils.Deref(compatible[idx].ClusterName))
			return strings.Contains(lower, strings.ToLower(input))
		},
		StartInSearchMode: true,
		LabelTemplate:     "{{.ClusterName | faint}}",
		Active:            `{{.ClusterName | bold}} ({{.State}} {{.Access}} Runtime {{.Runtime}}) ({{.ClusterId | faint}})`,
		Inactive:          `{{.ClusterName}} ({{.State}} {{.Access}} Runtime {{.Runtime}})`,
		Selected:          `{{ "Configured cluster" | faint }}: {{ .ClusterName | bold }} ({{.ClusterId | faint}})`,
	})
	if err != nil {
		return "", err
	}
	return utils.Deref(compatible[i].ClusterId), nil
}

// listUserClusters lists clusters created through the UI or API. It is a raw
// call because sdk-go models neither the filter_by request parameter nor the
// cluster_source response field.
func listUserClusters(ctx context.Context, w *sdkclients.Clients) ([]Cluster, error) {
	api, err := apiclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	type listedCluster struct {
		mirror.ClusterInfo
		ClusterSource string `json:"cluster_source,omitempty"`
	}
	var all []Cluster
	pageToken := ""
	for {
		query := url.Values{
			"filter_by.cluster_sources": {"API", "UI"},
			"page_size":                 {"100"},
		}
		if pageToken != "" {
			query.Set("page_token", pageToken)
		}
		var resp struct {
			Clusters      []listedCluster `json:"clusters"`
			NextPageToken string          `json:"next_page_token"`
		}
		err := api.Do(ctx, http.MethodGet, "/api/2.1/clusters/list?"+query.Encode(), auth.WorkspaceIDHeaders(w.Config), nil, nil, &resp)
		if err != nil {
			return nil, err
		}
		for i := range resp.Clusters {
			info, err := mirror.ClusterInfoToSDK(&resp.Clusters[i].ClusterInfo)
			if err != nil {
				return nil, err
			}
			all = append(all, Cluster{ClusterInfo: *info, Source: resp.Clusters[i].ClusterSource})
		}
		if resp.NextPageToken == "" {
			return all, nil
		}
		pageToken = resp.NextPageToken
	}
}
