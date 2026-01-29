package legacytemplates

import "text/template"

// getTemplateFuncs returns template functions for databricks.yml generation.
func getTemplateFuncs(resources *ResourceValues, appConfig *AppConfig) template.FuncMap {
	registry := GetGlobalRegistry()

	return template.FuncMap{
		// resourceTypes returns the ordered list of resource types to iterate over
		"resourceTypes": func() []ResourceType {
			return []ResourceType{
				ResourceTypeSQLWarehouse,
				ResourceTypeServingEndpoint,
				ResourceTypeExperiment,
				ResourceTypeDatabase,
				ResourceTypeUCVolume,
			}
		},

		// hasResource checks if a resource type has values
		"hasResource": func(rt ResourceType) bool {
			return resources.Has(rt)
		},

		// getValues returns the values for a resource type
		"getValues": func(rt ResourceType) []string {
			val := resources.Get(rt)
			if val == nil {
				return nil
			}
			return val.Values
		},

		// getValue returns the first value for a resource type
		"getValue": func(rt ResourceType) string {
			val := resources.Get(rt)
			if val == nil {
				return ""
			}
			return val.SingleValue()
		},

		// getMetadata returns metadata for a resource type
		"getMetadata": func(rt ResourceType) *ResourceMetadata {
			handler, ok := registry.Get(rt)
			if !ok {
				return nil
			}
			return handler.Metadata()
		},

		// getVariableName returns the primary variable name for a resource
		"getVariableName": func(rt ResourceType) string {
			handler, ok := registry.Get(rt)
			if !ok {
				return ""
			}
			meta := handler.Metadata()
			if len(meta.VariableNames) == 0 {
				return ""
			}
			return meta.VariableNames[0]
		},

		// getBindingLines returns binding lines for a resource
		"getBindingLines": func(rt ResourceType) []string {
			handler, ok := registry.Get(rt)
			if !ok {
				return nil
			}
			meta := handler.Metadata()
			if meta.BindingLines == nil {
				return nil
			}
			val := resources.Get(rt)
			if val == nil {
				return nil
			}
			return meta.BindingLines(val.Values)
		},

		// hasBindings checks if a resource type has binding lines
		"hasBindings": func(rt ResourceType) bool {
			handler, ok := registry.Get(rt)
			if !ok {
				return false
			}
			return handler.Metadata().BindingLines != nil
		},

		// App config functions
		"hasAppConfig": func() bool {
			return appConfig != nil
		},

		"hasAppCommand": func() bool {
			return appConfig != nil && len(appConfig.Command) > 0
		},

		"hasAppEnv": func() bool {
			return appConfig != nil && len(appConfig.Env) > 0
		},

		"getAppCommand": func() []string {
			if appConfig == nil {
				return nil
			}
			return appConfig.Command
		},

		"getAppEnv": func() []EnvVar {
			if appConfig == nil {
				return nil
			}
			return appConfig.Env
		},

		"hasAppResources": func() bool {
			return appConfig != nil && appConfig.ResourcesYAML != ""
		},

		"getAppResources": func() string {
			if appConfig == nil {
				return ""
			}
			return appConfig.ResourcesYAML
		},
	}
}
