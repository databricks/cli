package fuzz

import (
	"fmt"
	"math/rand/v2"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// Value pools are intentionally small and valid-looking: the goal is to exercise
// config->payload translation across many field combinations, not to stress the
// API with invalid values the testserver would reject.
var (
	sparkVersions = []string{"13.3.x-scala2.12", "14.3.x-scala2.12", "15.4.x-scala2.12", "16.4.x-scala2.12"}
	nodeTypeIDs   = []string{"i3.xlarge", "m5.large", "r5.xlarge", "Standard_DS3_v2"}
	timezones     = []string{"UTC", "America/Los_Angeles", "Europe/Amsterdam"}
	cronExprs     = []string{"0 0 12 * * ?", "0 15 10 ? * MON-FRI", "0 0/30 * * * ?"}
	pauseStatuses = []jobs.PauseStatus{jobs.PauseStatusPaused, jobs.PauseStatusUnpaused}
	performance   = []jobs.PerformanceTarget{jobs.PerformanceTargetPerformanceOptimized, jobs.PerformanceTargetStandard}
	timeUnits     = []string{"HOURS", "DAYS", "WEEKS"}
	healthMetrics = []string{"RUN_DURATION_SECONDS", "STREAMING_BACKLOG_BYTES", "STREAMING_BACKLOG_RECORDS"}
	conditionOps  = []string{"EQUAL_TO", "NOT_EQUAL", "GREATER_THAN", "LESS_THAN_OR_EQUAL"}
	runIfs        = []string{"ALL_SUCCESS", "AT_LEAST_ONE_SUCCESS", "NONE_FAILED", "ALL_DONE"}
	gitProviders  = []jobs.GitProvider{jobs.GitProviderGitHub, jobs.GitProviderGitLab, jobs.GitProviderAzureDevOpsServices}
)

// generateJob builds a random, well-formed job config driven entirely by rng, so
// the same seed always produces the same job. It favors fields whose
// config->payload translation is non-trivial (clusters, scheduling, references).
//
// TODO: generalize the harness across resource kinds.
func generateJob(rng *rand.Rand) *resources.Job {
	job := &resources.Job{}
	job.Name = randName(rng, "job")

	if chance(rng, 0.5) {
		job.Description = randSentence(rng)
	}
	if chance(rng, 0.4) {
		job.MaxConcurrentRuns = rng.IntN(10) + 1
	}
	if chance(rng, 0.4) {
		job.TimeoutSeconds = rng.IntN(7200)
	}
	if chance(rng, 0.3) {
		job.PerformanceTarget = oneOf(rng, performance)
	}
	if chance(rng, 0.5) {
		job.Tags = randTags(rng)
	}
	if chance(rng, 0.3) {
		job.GitSource = randGitSource(rng)
	}

	randScheduling(rng, job)

	if chance(rng, 0.3) {
		job.EmailNotifications = randEmailNotifications(rng)
	}
	if chance(rng, 0.2) {
		job.WebhookNotifications = randWebhookNotifications(rng)
	}
	if chance(rng, 0.3) {
		job.NotificationSettings = &jobs.JobNotificationSettings{
			NoAlertForCanceledRuns: chance(rng, 0.5),
			NoAlertForSkippedRuns:  chance(rng, 0.5),
		}
	}
	if chance(rng, 0.3) {
		job.Health = randHealth(rng)
	}
	if chance(rng, 0.3) {
		job.Parameters = randParameters(rng)
	}
	if chance(rng, 0.3) {
		job.Queue = &jobs.QueueSettings{Enabled: chance(rng, 0.5)}
	}

	// Generate shared job clusters first so tasks can reference them by key.
	var jobClusterKeys []string
	if chance(rng, 0.5) {
		n := rng.IntN(2) + 1
		for i := range n {
			key := fmt.Sprintf("cluster_%d", i)
			jobClusterKeys = append(jobClusterKeys, key)
			job.JobClusters = append(job.JobClusters, jobs.JobCluster{
				JobClusterKey: key,
				NewCluster:    randClusterSpec(rng),
			})
		}
	}

	nTasks := rng.IntN(3) + 1
	var taskKeys []string
	for i := range nTasks {
		task := randTask(rng, i, jobClusterKeys)
		// Randomly chain dependencies onto previously generated tasks.
		if len(taskKeys) > 0 && chance(rng, 0.4) {
			dep := taskKeys[rng.IntN(len(taskKeys))]
			task.DependsOn = []jobs.TaskDependency{{TaskKey: dep}}
			if chance(rng, 0.5) {
				task.RunIf = jobs.RunIf(oneOf(rng, runIfs))
			}
		}
		taskKeys = append(taskKeys, task.TaskKey)
		job.Tasks = append(job.Tasks, task)
	}

	return job
}

// randScheduling sets at most one of schedule/trigger/continuous, which are
// mutually exclusive ways to launch a job.
func randScheduling(rng *rand.Rand, job *resources.Job) {
	switch rng.IntN(5) {
	case 0:
		job.Schedule = &jobs.CronSchedule{
			QuartzCronExpression: oneOf(rng, cronExprs),
			TimezoneId:           oneOf(rng, timezones),
			PauseStatus:          oneOf(rng, pauseStatuses),
		}
	case 1:
		job.Trigger = &jobs.TriggerSettings{
			PauseStatus: oneOf(rng, pauseStatuses),
			Periodic: &jobs.PeriodicTriggerConfiguration{
				Interval: rng.IntN(12) + 1,
				Unit:     jobs.PeriodicTriggerConfigurationTimeUnit(oneOf(rng, timeUnits)),
			},
		}
	case 2:
		job.Trigger = &jobs.TriggerSettings{
			PauseStatus: oneOf(rng, pauseStatuses),
			FileArrival: &jobs.FileArrivalTriggerConfiguration{
				Url: "s3://" + randWord(rng) + "/" + randWord(rng),
			},
		}
	case 3:
		job.Continuous = &jobs.Continuous{PauseStatus: oneOf(rng, pauseStatuses)}
	default:
		// no scheduling
	}
}

func randTask(rng *rand.Rand, idx int, jobClusterKeys []string) jobs.Task {
	task := jobs.Task{TaskKey: fmt.Sprintf("task_%d", idx)}

	// Use absolute workspace paths so deploy never depends on local files.
	// condition_task needs no compute, handled separately below.
	needsCompute := true
	switch rng.IntN(4) {
	case 0:
		task.NotebookTask = &jobs.NotebookTask{
			NotebookPath: "/Workspace/Users/test/" + randName(rng, "nb"),
			Source:       jobs.SourceWorkspace,
		}
	case 1:
		task.SparkPythonTask = &jobs.SparkPythonTask{
			PythonFile: "/Workspace/Users/test/" + randName(rng, "main") + ".py",
			Source:     jobs.SourceWorkspace,
		}
	case 2:
		task.PythonWheelTask = &jobs.PythonWheelTask{
			PackageName: randName(rng, "pkg"),
			EntryPoint:  "main",
		}
	case 3:
		task.ConditionTask = &jobs.ConditionTask{
			Left:  randWord(rng),
			Op:    jobs.ConditionTaskOp(oneOf(rng, conditionOps)),
			Right: randWord(rng),
		}
		needsCompute = false
	}

	if needsCompute {
		assignCompute(rng, &task, jobClusterKeys)
		if chance(rng, 0.4) {
			task.Libraries = randLibraries(rng)
		}
	}

	if chance(rng, 0.3) {
		task.TimeoutSeconds = rng.IntN(3600)
	}
	if chance(rng, 0.3) {
		task.MaxRetries = rng.IntN(5)
		task.MinRetryIntervalMillis = rng.IntN(60000)
		task.RetryOnTimeout = chance(rng, 0.5)
	}
	return task
}

// assignCompute attaches exactly one compute source: a shared job cluster (when
// available), a new cluster, or an existing cluster id.
func assignCompute(rng *rand.Rand, task *jobs.Task, jobClusterKeys []string) {
	const (
		computeNew = iota
		computeExisting
		computeShared
	)
	options := []int{computeNew, computeExisting}
	if len(jobClusterKeys) > 0 {
		options = append(options, computeShared)
	}
	switch oneOf(rng, options) {
	case computeNew:
		spec := randClusterSpec(rng)
		task.NewCluster = &spec
	case computeExisting:
		task.ExistingClusterId = randName(rng, "cluster")
	case computeShared:
		task.JobClusterKey = oneOf(rng, jobClusterKeys)
	}
}

func randClusterSpec(rng *rand.Rand) compute.ClusterSpec {
	spec := compute.ClusterSpec{
		SparkVersion: oneOf(rng, sparkVersions),
		NodeTypeId:   oneOf(rng, nodeTypeIDs),
	}
	if chance(rng, 0.5) {
		spec.NumWorkers = rng.IntN(8)
	} else {
		spec.Autoscale = &compute.AutoScale{
			MinWorkers: 1,
			MaxWorkers: rng.IntN(8) + 2,
		}
	}
	if chance(rng, 0.4) {
		spec.SparkConf = map[string]string{
			"spark.databricks.delta.preview.enabled": "true",
			"spark.speculation":                      strconv.FormatBool(chance(rng, 0.5)),
		}
	}
	if chance(rng, 0.3) {
		spec.CustomTags = randTags(rng)
	}
	if chance(rng, 0.3) {
		spec.SparkEnvVars = map[string]string{"PYSPARK_PYTHON": "/databricks/python3/bin/python3"}
	}
	if chance(rng, 0.3) {
		spec.DriverNodeTypeId = oneOf(rng, nodeTypeIDs)
	}
	return spec
}

func randGitSource(rng *rand.Rand) *jobs.GitSource {
	src := &jobs.GitSource{
		GitProvider: oneOf(rng, gitProviders),
		GitUrl:      "https://example.com/" + randWord(rng) + "/" + randWord(rng) + ".git",
	}
	switch rng.IntN(3) {
	case 0:
		src.GitBranch = oneOf(rng, []string{"main", "develop", "release"})
	case 1:
		src.GitTag = "v" + fmt.Sprintf("%d.%d.0", rng.IntN(5), rng.IntN(10))
	case 2:
		src.GitCommit = fmt.Sprintf("%040x", rng.Int64())
	}
	return src
}

func randEmailNotifications(rng *rand.Rand) *jobs.JobEmailNotifications {
	email := randWord(rng) + "@example.com"
	n := &jobs.JobEmailNotifications{NoAlertForSkippedRuns: chance(rng, 0.5)}
	if chance(rng, 0.6) {
		n.OnFailure = []string{email}
	}
	if chance(rng, 0.4) {
		n.OnSuccess = []string{email}
	}
	if chance(rng, 0.3) {
		n.OnStart = []string{email}
	}
	return n
}

func randWebhookNotifications(rng *rand.Rand) *jobs.WebhookNotifications {
	hook := []jobs.Webhook{{Id: randName(rng, "hook")}}
	n := &jobs.WebhookNotifications{}
	if chance(rng, 0.6) {
		n.OnFailure = hook
	}
	if chance(rng, 0.4) {
		n.OnSuccess = hook
	}
	return n
}

func randHealth(rng *rand.Rand) *jobs.JobsHealthRules {
	return &jobs.JobsHealthRules{
		Rules: []jobs.JobsHealthRule{
			{
				Metric: jobs.JobsHealthMetric(oneOf(rng, healthMetrics)),
				Op:     jobs.JobsHealthOperatorGreaterThan,
				Value:  int64(rng.IntN(3600) + 1),
			},
		},
	}
}

func randLibraries(rng *rand.Rand) []compute.Library {
	n := rng.IntN(2) + 1
	libs := make([]compute.Library, 0, n)
	for range n {
		switch rng.IntN(3) {
		case 0:
			libs = append(libs, compute.Library{Pypi: &compute.PythonPyPiLibrary{Package: randWord(rng)}})
		case 1:
			libs = append(libs, compute.Library{Maven: &compute.MavenLibrary{Coordinates: "org.example:" + randWord(rng) + ":1.0.0"}})
		case 2:
			libs = append(libs, compute.Library{Whl: "/Workspace/Users/test/" + randName(rng, "lib") + ".whl"})
		}
	}
	return libs
}

func randParameters(rng *rand.Rand) []jobs.JobParameterDefinition {
	n := rng.IntN(3) + 1
	params := make([]jobs.JobParameterDefinition, 0, n)
	for i := range n {
		params = append(params, jobs.JobParameterDefinition{
			Name:    fmt.Sprintf("param_%d", i),
			Default: randWord(rng),
		})
	}
	return params
}

func randTags(rng *rand.Rand) map[string]string {
	n := rng.IntN(3) + 1
	tags := make(map[string]string, n)
	for i := range n {
		tags[fmt.Sprintf("tag_%d", i)] = randWord(rng)
	}
	return tags
}
