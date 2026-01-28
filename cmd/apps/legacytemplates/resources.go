package legacytemplates

import "context"

// LegacyResourceCollector handles gathering all required resources for a legacy template.
type LegacyResourceCollector struct {
	template      *AppTemplateManifest
	isInteractive bool
	// Input values from flags
	warehouseID     string
	servingEndpoint string
	experimentID    string
	instanceName    string
	databaseName    string
	ucVolume        string
}

// NewLegacyResourceCollector creates a new resource collector for a legacy template.
func NewLegacyResourceCollector(template *AppTemplateManifest, isInteractive bool, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume string) *LegacyResourceCollector {
	return &LegacyResourceCollector{
		template:        template,
		isInteractive:   isInteractive,
		warehouseID:     warehouseID,
		servingEndpoint: servingEndpoint,
		experimentID:    experimentID,
		instanceName:    instanceName,
		databaseName:    databaseName,
		ucVolume:        ucVolume,
	}
}

// LegacyResources holds all collected resource values.
type LegacyResources struct {
	WarehouseID     string
	ServingEndpoint string
	ExperimentID    string
	InstanceName    string
	DatabaseName    string
	UCVolume        string
}

// CollectAll gathers all required resources for the template.
func (c *LegacyResourceCollector) CollectAll(ctx context.Context) (*LegacyResources, error) {
	resources := &LegacyResources{}

	// Get warehouse ID if needed
	warehouseID, err := GetWarehouseIDForTemplate(ctx, c.template, c.warehouseID, c.isInteractive)
	if err != nil {
		return nil, err
	}
	resources.WarehouseID = warehouseID

	// Get serving endpoint if needed
	servingEndpoint, err := GetServingEndpointForTemplate(ctx, c.template, c.servingEndpoint, c.isInteractive)
	if err != nil {
		return nil, err
	}
	resources.ServingEndpoint = servingEndpoint

	// Get experiment ID if needed
	experimentID, err := GetExperimentIDForTemplate(ctx, c.template, c.experimentID, c.isInteractive)
	if err != nil {
		return nil, err
	}
	resources.ExperimentID = experimentID

	// Get database resources if needed
	instanceName, databaseName, err := GetDatabaseForTemplate(ctx, c.template, c.instanceName, c.databaseName, c.isInteractive)
	if err != nil {
		return nil, err
	}
	resources.InstanceName = instanceName
	resources.DatabaseName = databaseName

	// Get UC volume if needed
	ucVolume, err := GetUCVolumeForTemplate(ctx, c.template, c.ucVolume, c.isInteractive)
	if err != nil {
		return nil, err
	}
	resources.UCVolume = ucVolume

	return resources, nil
}
