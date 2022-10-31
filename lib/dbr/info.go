package dbr

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/retries"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

type RuntimeInfo struct {
	Name          string    `json:"name"`
	SparkVersion  string    `json:"spark_version"`
	PythonVersion string    `json:"python_version"`
	PyPI          []Package `json:"pypi"`
	Jars          []Package `json:"jars"`
}

type Package struct {
	Group   string `json:"group,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (pkg *Package) PyPiName() string {
	return fmt.Sprintf("%s==%s", pkg.Name, pkg.Version)
}

func GetRuntimeInfo(ctx context.Context, w *workspaces.WorkspacesClient,
	clusterId string, status func(string)) (*RuntimeInfo, error) {
	cluster, err := w.Clusters.GetByClusterId(ctx, clusterId)
	if err != nil {
		return nil, err
	}
	if !cluster.IsRunningOrResizing() {
		_, err = w.Clusters.StartByClusterIdAndWait(ctx, clusterId,
			func(i *retries.Info[clusters.ClusterInfo]) {
				status(i.Info.StateMessage)
			})
		if err != nil {
			return nil, err
		}
	}
	command := w.CommandExecutor.Execute(ctx, clusterId, "python", infoScript)
	if command.Failed() {
		return nil, command.Err()
	}
	var info RuntimeInfo
	err = json.Unmarshal([]byte(command.Text()), &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

const infoScript = `import pkg_resources, json, sys, platform, subprocess
from pyspark.version import __version__

jars = []
for j in subprocess.check_output(['ls', '-1', '/databricks/jars']).decode().split("\n"):
	if '--mvn--' in j or '--maven-trees--' in j:
		split = j.split('--')[-1][:-4].split('__')
		if not len(split) == 3:
			continue
		group, artifactId, version = split
		jars.append({
			'group': group,
			'name': artifactId,
			'version': version,
		})
jars = sorted(jars, key=lambda jar: (jar['group'], jar['name']))

python_packages = [
	{"name": n, "version": v}
	for n, v in sorted([(i.key, i.version) 
	for i in pkg_resources.working_set])]
python_packages = sorted(python_packages, key=lambda x: x['name'])

runtime = spark.conf.get('spark.databricks.clusterUsageTags.sparkVersion')
print(json.dumps({
	'name': runtime,
	'spark_version': __version__[0:5],
	'python_version': platform.python_version(),
	'pypi': python_packages,
	'jars': jars,
}))`
