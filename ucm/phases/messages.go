package phases

// Messages for ucm deploy.
const (
	deleteOrRecreateCatalogMessage = `
This action will result in the deletion or recreation of the following UC catalogs.
Recreating a catalog destroys all schemas, tables, and volumes it contains, and any
underlying data may be lost:`

	deleteOrRecreateSchemaMessage = `
This action will result in the deletion or recreation of the following UC schemas. Any underlying data may be lost:`

	deleteOrRecreateVolumeMessage = `
This action will result in the deletion or recreation of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:`
)

// Messages for ucm destroy.
const (
	deleteCatalogMessage = `This action will result in the deletion of the following UC catalogs along with all schemas, tables, and volumes they contain. Any underlying data may be lost:`

	deleteSchemaMessage = `This action will result in the deletion of the following UC schemas. Any underlying data may be lost:`

	deleteVolumeMessage = `This action will result in the deletion of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:`
)
