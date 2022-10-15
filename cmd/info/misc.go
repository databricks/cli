package info

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/project"
	"github.com/databricks/bricks/python"
	"github.com/databricks/databricks-sdk-go/service/clusters"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/spf13/cobra"
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Misc info commands",
}

type Package struct {
	Group   string `json:"group,omitempty"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

type RuntimeInfo struct {
	Name          string    `json:"name"`
	SparkVersion  string    `json:"spark_version"`
	PythonVersion string    `json:"python_version"`
	PyPI          []Package `json:"pypi"`
	Jars          []Package `json:"jars"`
}

var pyPkgsCmd = &cobra.Command{
	Use:     "python-packages",
	Short:   "Get python packages",
	PreRunE: project.Configure,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		prj := project.Get(ctx)
		wsc := prj.WorkspacesClient()
		nodes, err := wsc.Clusters.ListNodeTypes(ctx)
		if err != nil {
			return err
		}
		smallestVM, err := nodes.Smallest(clusters.NodeTypeRequest{
			LocalDisk: true,
		})
		if err != nil {
			return err
		}
		tasks := []jobs.RunSubmitTaskSettings{}

		me, err := wsc.CurrentUser.Me(ctx)
		if err != nil {
			return err
		}
		now := time.Now().UnixNano()
		notebookPath := fmt.Sprintf("/Users/%s/python-packages-%d", me.UserName, now)

		err = wsc.Workspace.Import(ctx, workspace.Import{
			Path: notebookPath,
			Content: base64.StdEncoding.Strict().EncodeToString([]byte(python.TrimLeadingWhitespace(`
			import pkg_resources, json, sys, platform, subprocess
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
			
			runtime = dbutils.widgets.get("runtime")
			dbutils.notebook.exit(json.dumps({
				'name': runtime,
				'spark_version': __version__[0:5],
				'python_version': platform.python_version(),
				'pypi': python_packages,
				'jars': jars,
			}))`))),
			Language:  workspace.ImportLanguagePython,
			Format:    workspace.ImportFormatSource,
			Overwrite: true,
		})
		if err != nil {
			return err
		}
		// defer wsc.Workspace.Delete(ctx, workspace.Delete{
		// 	Path: notebookPath,
		// })
		runtimes, err := wsc.Clusters.SparkVersions(ctx)
		if err != nil {
			return err
		}
		for i, r := range runtimes.Versions {
			if strings.Contains(r.Key, "-aarch64") {
				continue
			}
			if strings.Contains(r.Key, "apache-") {
				continue
			}
			if strings.Contains(r.Key, "-gpu") {
				continue
			}
			tasks = append(tasks, jobs.RunSubmitTaskSettings{
				TaskKey: fmt.Sprintf("task-%d", i),
				NotebookTask: &jobs.NotebookTask{
					NotebookPath: notebookPath,
					BaseParameters: map[string]any{
						"runtime": r.Key,
					},
				},
				NewCluster: &jobs.NewCluster{
					SparkVersion: r.Key,
					NumWorkers:   1,
					NodeTypeId:   smallestVM,
				},
			})
		}
		run, err := wsc.Jobs.SubmitAndWait(cmd.Context(), jobs.SubmitRun{
			RunName: "Get Python packages",
			Tasks:   tasks,
		})
		if err != nil {
			return err
		}
		var infos []RuntimeInfo
		for _, t := range run.Tasks {
			output, err := wsc.Jobs.GetRunOutputByRunId(ctx, t.RunId)
			if err != nil {
				return err
			}
			var info RuntimeInfo // todo: fails on cancelled job
			err = json.Unmarshal([]byte(output.NotebookOutput.Result), &info)
			if err != nil {
				return err
			}
			infos = append(infos, info)
		}

		for _, v := range infos {
			raw, _ := json.Marshal(v)
			fmt.Printf("%s\n", string(raw))
		}
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(infoCmd)
	infoCmd.AddCommand(pyPkgsCmd)
}
