package project

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/databricks/bricks/lib/flavor"
	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/libraries"
)

type runsOnCluster interface {
	RequiresCluster() bool
}

type restartable interface {
	RequiresRestart() bool
}

func (p *project) RequiresCluster() bool {
	for _, f := range p.flavors {
		if !f.Detected(p) {
			continue
		}
		r, ok := f.(runsOnCluster)
		if !ok {
			continue
		}
		if r.RequiresCluster() {
			return true
		}
	}
	return false
}

func (p *project) RequiresRestart() bool {
	for _, f := range p.flavors {
		if !f.Detected(p) {
			continue
		}
		r, ok := f.(restartable)
		if !ok {
			continue
		}
		if r.RequiresRestart() {
			return true
		}
	}
	return false
}

func (p *project) Install(ctx context.Context, status func(string)) error {
	if !p.RequiresCluster() {
		// nothing to do
		return nil
	}
	clusterId, err := p.GetDevelopmentClusterId(ctx)
	if err != nil {
		// cluster not found is also fine, abort execution
		return nil
	}
	info, err := p.wsc.Clusters.GetByClusterId(ctx, clusterId)
	if err != nil {
		// TODO: special behavior for (auto)deleted clusters
		// re-create, if possible?
		return err
	}
	if p.RequiresRestart() && info.IsRunningOrResizing() {
		_, err = p.wsc.Clusters.RestartAndWait(ctx, clusters.RestartCluster{
			ClusterId: clusterId,
		}, func(i *retries.Info[clusters.ClusterInfo]) {
			status(i.Info.StateMessage)
		})
	} else if !info.IsRunningOrResizing() {
		_, err = p.wsc.Clusters.StartByClusterIdAndWait(ctx, clusterId,
			func(i *retries.Info[clusters.ClusterInfo]) {
				status(i.Info.StateMessage)
			})
	}
	if err != nil {
		return err
	}
	if !p.artifacts.HasLibraries() {
		return nil
	}
	var libs []libraries.Library
	for _, a := range p.artifacts {
		k, _, remote := p.remotePath(a)
		switch k {
		case flavor.LocalJar:
			libs = append(libs, libraries.Library{Jar: remote})
		case flavor.LocalWheel:
			libs = append(libs, libraries.Library{Whl: remote})
		case flavor.LocalEgg:
			libs = append(libs, libraries.Library{Egg: remote})
		case flavor.RegistryLibrary:
			libs = append(libs, a.Library)
		default:
			continue
		}
	}
	// TODO: uninstall previous versions of libraries
	return p.wsc.Libraries.UpdateAndWait(ctx, libraries.Update{
		ClusterId: clusterId,
		Install:   libs,
	}, func(i *retries.Info[libraries.ClusterLibraryStatuses]) {
		byStatus := map[string][]string{}
		for _, lib := range i.Info.LibraryStatuses {
			if lib.IsLibraryForAllClusters {
				continue
			}
			if lib.Status == libraries.LibraryFullStatusStatusInstalled ||
				lib.Status == libraries.LibraryFullStatusStatusUninstallOnRestart {
				continue
			}
			name := lib.Library.String()
			if strings.HasPrefix(name, "jar:dbfs:") || strings.HasPrefix(name, "whl:dbfs:") {
				name = filepath.Base(name)
			}
			byStatus[string(lib.Status)] = append(byStatus[string(lib.Status)], name)
		}
		msg := []string{}
		for k, v := range byStatus {
			sort.Strings(v)
			msg = append(msg, fmt.Sprintf("%s (%s)", k, strings.Join(v, ", ")))
		}
		sort.Strings(msg)
		status(strings.Join(msg, ", "))
	})
}
