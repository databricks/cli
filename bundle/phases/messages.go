package phases

// Messages for bundle deploy.
const (
	deleteOrRecreateResourceMessage = `
This action will result in the deletion or recreation of the following resources.
Deleted or recreated resources may result in data loss or other irreversible consequences:`
)

// Messages for bundle destroy.
const (
	deleteResourceMessage = `The following resources may contain data that will be lost upon deletion:`
)
