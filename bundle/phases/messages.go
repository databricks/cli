package phases

// Messages for bundle deploy.
const (
	deleteOrRecreateSchemaMessage = `
This action will result in the deletion or recreation of the following UC schemas. Any underlying data may be lost:`

	deleteOrRecreatePipelineMessage = `
This action will result in the deletion or recreation of the following Lakeflow Spark Declarative Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the pipelines will
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

	deleteOrRecreateDatabaseInstanceMessage = `
This action will result in the deletion or recreation of the following Lakebase database instances.
All data stored in them will be permanently lost:`

	deleteOrRecreateSyncedDatabaseTableMessage = `
This action will result in the deletion or recreation of the following synced database tables.
The synced data in the destination database will be lost (the source table is preserved):`

	deleteOrRecreatePostgresProjectMessage = `
This action will result in the deletion or recreation of the following Lakebase projects along with
all their branches, databases, and endpoints. All data stored in them will be permanently lost:`

	deleteOrRecreatePostgresBranchMessage = `
This action will result in the deletion or recreation of the following Lakebase branches.
All data stored in them will be permanently lost:`
)

// Messages for bundle destroy.
const (
	deleteSchemaMessage = `This action will result in the deletion of the following UC schemas. Any underlying data may be lost:`

	deletePipelineMessage = `This action will result in the deletion of the following Lakeflow Spark Declarative Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them:`

	deleteVolumeMessage = `This action will result in the deletion of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:`

	deleteDatabaseInstanceMessage = `This action will result in the deletion of the following Lakebase database instances.
All data stored in them will be permanently lost:`

	deleteSyncedDatabaseTableMessage = `This action will result in the deletion of the following synced database tables.
The synced data in the destination database will be lost (the source table is preserved):`

	deletePostgresProjectMessage = `This action will result in the deletion of the following Lakebase projects along with
all their branches, databases, and endpoints. All data stored in them will be permanently lost:`

	deletePostgresBranchMessage = `This action will result in the deletion of the following Lakebase branches.
All data stored in them will be permanently lost:`
)
