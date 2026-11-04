package metrics

// Boolean telemetry keys are declared as `metric` tags on bundle.Telemetry.

// Key under which the snapshot upload duration is reported in
// bundle_mutator_execution_time_ms, next to files.(upload) so the two can be
// compared. Covers creating the snapshot resource (the upload plus recording it in
// state), not building the zip, which is snapshot.(snapshotUpload). Absent when no
// snapshot was uploaded. It lacks the parentheses of a mutator name because it is
// not a mutator.
const SnapshotUploadDuration = "snapshot.upload"
