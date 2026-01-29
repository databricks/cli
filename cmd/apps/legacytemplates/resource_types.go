package legacytemplates

// ResourceType represents a specific type of Databricks resource.
type ResourceType string

const (
	ResourceTypeSQLWarehouse    ResourceType = "sql_warehouse"
	ResourceTypeServingEndpoint ResourceType = "serving_endpoint"
	ResourceTypeExperiment      ResourceType = "experiment"
	ResourceTypeDatabase        ResourceType = "database"
	ResourceTypeUCVolume        ResourceType = "uc_volume"
)

// ResourceSpecKey represents the JSON key in resourceSpec for a specific resource type.
type ResourceSpecKey string

const (
	SpecKeySQLWarehouse    ResourceSpecKey = "SQLWarehouse"
	SpecKeyServingEndpoint ResourceSpecKey = "ServingEndpoint"
	SpecKeyExperiment      ResourceSpecKey = "Experiment"
	SpecKeyDatabase        ResourceSpecKey = "Database"
	SpecKeyUCVolume        ResourceSpecKey = "UCSecurable"
)

// ResourceValue holds the collected value(s) for a resource.
// Most resources have a single value, but some (like database) have multiple.
type ResourceValue struct {
	Type   ResourceType
	Values []string
}

// SingleValue returns the first value, or empty string if none.
// Used for resources that only have one value.
func (r *ResourceValue) SingleValue() string {
	if len(r.Values) == 0 {
		return ""
	}
	return r.Values[0]
}

// IsEmpty returns true if the resource has no values.
func (r *ResourceValue) IsEmpty() bool {
	return len(r.Values) == 0
}

// ResourceValues holds all collected resource values.
type ResourceValues struct {
	values map[ResourceType]*ResourceValue
}

// NewResourceValues creates a new ResourceValues instance.
func NewResourceValues() *ResourceValues {
	return &ResourceValues{
		values: make(map[ResourceType]*ResourceValue),
	}
}

// Set sets the value for a resource type.
func (r *ResourceValues) Set(resourceType ResourceType, values ...string) {
	if len(values) == 0 {
		return
	}
	// Filter out empty strings
	filtered := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		return
	}
	r.values[resourceType] = &ResourceValue{
		Type:   resourceType,
		Values: filtered,
	}
}

// Get retrieves the value for a resource type.
// Returns nil if the resource type is not set.
func (r *ResourceValues) Get(resourceType ResourceType) *ResourceValue {
	return r.values[resourceType]
}

// Has checks if a resource type has been set (with non-empty values).
func (r *ResourceValues) Has(resourceType ResourceType) bool {
	val := r.values[resourceType]
	return val != nil && !val.IsEmpty()
}

// All returns all set resource values.
func (r *ResourceValues) All() map[ResourceType]*ResourceValue {
	return r.values
}
