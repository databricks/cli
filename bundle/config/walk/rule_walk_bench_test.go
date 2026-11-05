package walk

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func benchConfig(n int) *config.Root {
	r := &config.Root{Bundle: config.Bundle{Name: "b"}}
	r.Resources.Jobs = map[string]*resources.Job{}
	for i := range n {
		var tasks []jobs.Task
		for j := range 5 {
			t := jobs.Task{
				TaskKey:      fmt.Sprintf("t%d", j),
				NotebookTask: &jobs.NotebookTask{NotebookPath: "/a/b"},
				NewCluster: &compute.ClusterSpec{
					SparkVersion: "15.4.x-scala2.12", NodeTypeId: "i3.xlarge", NumWorkers: 2,
					SparkConf: map[string]string{"a": "b"},
				},
			}
			if j == 3 {
				t.NotebookTask = &jobs.NotebookTask{}
			}
			tasks = append(tasks, t)
		}
		r.Resources.Jobs[fmt.Sprintf("job%d", i)] = &resources.Job{
			Name:     fmt.Sprintf("job%d", i),
			Tasks:    tasks,
			Schedule: &jobs.CronSchedule{QuartzCronExpression: "0 0 * * * ?"},
			Tags:     map[string]string{"x": "y"},
		}
	}
	return r
}

func BenchmarkRequired(b *testing.B) {
	r := benchConfig(1000)
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		Required(r, func(*Node, string) { n++ })
	}
}

func BenchmarkEnum(b *testing.B) {
	r := benchConfig(1000)
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		Enums(r, func(*Node, string, []string) { n++ })
	}
}

func BenchmarkStringsNonEmpty(b *testing.B) {
	r := benchConfig(1000)
	b.ReportAllocs()
	n := 0
	for b.Loop() {
		Strings(r, NonEmpty, func(*Node, string) (string, bool) { n++; return "", false })
	}
}

func BenchmarkStringsRefs(b *testing.B) {
	r := benchConfig(1000)
	b.ReportAllocs()
	for b.Loop() {
		Strings(r, WithReferences, func(*Node, string) (string, bool) { return "", false })
	}
}
