package phases

// Messages for bundle deploy.
const (
	deleteOrRecreateSchemaMessage = `
	This action will result in the deletion or recreation of the following UC schemas. Any underlying data may be lost:`
	deleteOrRecreateDltMessage = `
This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:`

	deleteOrRecreateVolumeMessage = `
This action will result in the deletion or recreation of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:`

	deleteOrRecreateDashboardMessage = `
This action will result in the deletion or recreation of the following dashboards.
This will result in changed IDs and permanent URLs of the dashboards that will be recreated:`
)

// Messages for bundle destroy.
const (
	deleteSchemaMessage = `This action will result in the deletion of the following UC schemas. Any underlying data may be lost:`

	deleteDltMessage = `This action will result in the deletion of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them:`

	deleteVolumeMessage = `This action will result in the deletion of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:`
)
