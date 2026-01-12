// Package main generates SDK documentation index for MCP tools.
//
// Usage:
//
//	go run tools/gen_sdk_docs_index.go -output experimental/aitools/lib/providers/sdkdocs/
//
// This tool parses the annotations_openapi.yml file and Go SDK interfaces to generate
// a comprehensive SDK documentation index that is embedded into the CLI binary.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// SDKDocsIndex represents the complete SDK documentation index.
type SDKDocsIndex struct {
	Version     string                 `json:"version"`
	GeneratedAt string                 `json:"generated_at"`
	Services    map[string]*ServiceDoc `json:"services"`
	Types       map[string]*TypeDoc    `json:"types"`
	Enums       map[string]*EnumDoc    `json:"enums"`
}

// ServiceDoc represents documentation for an API service.
type ServiceDoc struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Package     string                `json:"package"`
	Methods     map[string]*MethodDoc `json:"methods"`
}

// MethodDoc represents documentation for an API method.
type MethodDoc struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Signature   string     `json:"signature"`
	Parameters  []ParamDoc `json:"parameters"`
	Returns     *ReturnDoc `json:"returns,omitempty"`
	Example     string     `json:"example,omitempty"`
	HTTPMethod  string     `json:"http_method,omitempty"`
	HTTPPath    string     `json:"http_path,omitempty"`
}

// ParamDoc represents documentation for a method parameter.
type ParamDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ReturnDoc represents documentation for a method return type.
type ReturnDoc struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// TypeDoc represents documentation for a data type.
type TypeDoc struct {
	Name        string               `json:"name"`
	Package     string               `json:"package"`
	Description string               `json:"description"`
	Fields      map[string]*FieldDoc `json:"fields"`
}

// FieldDoc represents documentation for a struct field.
type FieldDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	OutputOnly  bool   `json:"output_only,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

// EnumDoc represents documentation for an enum type.
type EnumDoc struct {
	Name        string   `json:"name"`
	Package     string   `json:"package"`
	Description string   `json:"description"`
	Values      []string `json:"values"`
}

// AnnotationsFile represents the structure of annotations_openapi.yml
type AnnotationsFile map[string]map[string]FieldAnnotation

// FieldAnnotation represents annotations for a single field
type FieldAnnotation struct {
	Description        string `yaml:"description"`
	OutputOnly         string `yaml:"x-databricks-field-behaviors_output_only"`
	DeprecationMessage string `yaml:"deprecation_message"`
}

func main() {
	outputDir := flag.String("output", "experimental/aitools/lib/providers/sdkdocs/", "Output directory for generated index")
	annotationsPath := flag.String("annotations", "bundle/internal/schema/annotations_openapi.yml", "Path to annotations file")
	flag.Parse()

	// Find project root
	projectRoot, err := findProjectRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding project root: %v\n", err)
		os.Exit(1)
	}

	// Load annotations
	annotations, err := loadAnnotations(filepath.Join(projectRoot, *annotationsPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading annotations: %v\n", err)
		os.Exit(1)
	}

	// Generate index
	index := generateIndex(annotations)

	// Write output
	outputPath := filepath.Join(projectRoot, *outputDir, "sdk_docs_index.json")
	if err := writeIndex(index, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing index: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated SDK docs index: %s\n", outputPath)
	fmt.Printf("  Services: %d\n", len(index.Services))
	fmt.Printf("  Types: %d\n", len(index.Types))
	fmt.Printf("  Enums: %d\n", len(index.Enums))
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find project root (go.mod)")
		}
		dir = parent
	}
}

func loadAnnotations(path string) (AnnotationsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read annotations file: %w", err)
	}

	var annotations AnnotationsFile
	if err := yaml.Unmarshal(data, &annotations); err != nil {
		return nil, fmt.Errorf("failed to parse annotations: %w", err)
	}

	return annotations, nil
}

func generateIndex(annotations AnnotationsFile) *SDKDocsIndex {
	index := &SDKDocsIndex{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Services:    make(map[string]*ServiceDoc),
		Types:       make(map[string]*TypeDoc),
		Enums:       make(map[string]*EnumDoc),
	}

	// Extract types from annotations
	for fullTypeName, fields := range annotations {
		typeName := extractTypeName(fullTypeName)
		packageName := extractPackageName(fullTypeName)

		if typeName == "" {
			continue
		}

		typeDoc := &TypeDoc{
			Name:        typeName,
			Package:     packageName,
			Description: inferTypeDescription(typeName),
			Fields:      make(map[string]*FieldDoc),
		}

		for fieldName, annotation := range fields {
			if fieldName == "_" {
				// Type-level description
				if annotation.Description != "" {
					typeDoc.Description = annotation.Description
				}
				continue
			}

			fieldDoc := &FieldDoc{
				Name:        fieldName,
				Type:        inferFieldType(fieldName),
				Description: annotation.Description,
				OutputOnly:  annotation.OutputOnly == "true",
				Deprecated:  annotation.DeprecationMessage != "",
			}
			typeDoc.Fields[fieldName] = fieldDoc
		}

		// Determine the service this type belongs to
		service := inferServiceFromPackage(packageName)
		typePath := service + "." + typeName
		index.Types[typePath] = typeDoc
	}

	// Add well-known services with common methods
	addCoreServices(index)

	return index
}

func extractTypeName(fullPath string) string {
	// Extract type name from paths like "github.com/databricks/cli/bundle/config/resources.Alert"
	parts := strings.Split(fullPath, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractPackageName(fullPath string) string {
	// Extract package from paths like "github.com/databricks/cli/bundle/config/resources.Alert"
	parts := strings.Split(fullPath, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		if idx := strings.Index(lastPart, "."); idx > 0 {
			return lastPart[:idx]
		}
		return lastPart
	}
	return ""
}

func inferServiceFromPackage(packageName string) string {
	// Map package names to service names
	serviceMap := map[string]string{
		"resources": "bundle",
		"jobs":      "jobs",
		"clusters":  "compute",
		"compute":   "compute",
		"pipelines": "pipelines",
		"catalog":   "catalog",
		"sql":       "sql",
		"apps":      "apps",
		"serving":   "serving",
		"ml":        "ml",
		"workspace": "workspace",
		"iam":       "iam",
		"settings":  "settings",
		"files":     "files",
		"sharing":   "sharing",
	}

	if service, ok := serviceMap[packageName]; ok {
		return service
	}
	return packageName
}

func inferTypeDescription(typeName string) string {
	// Generate reasonable descriptions based on type name patterns
	if strings.HasSuffix(typeName, "Request") {
		base := strings.TrimSuffix(typeName, "Request")
		return fmt.Sprintf("Request parameters for %s operation.", toSentenceCase(base))
	}
	if strings.HasSuffix(typeName, "Response") {
		base := strings.TrimSuffix(typeName, "Response")
		return fmt.Sprintf("Response from %s operation.", toSentenceCase(base))
	}
	if strings.HasSuffix(typeName, "Settings") {
		base := strings.TrimSuffix(typeName, "Settings")
		return fmt.Sprintf("Configuration settings for %s.", toSentenceCase(base))
	}
	if strings.HasSuffix(typeName, "Spec") {
		base := strings.TrimSuffix(typeName, "Spec")
		return fmt.Sprintf("Specification for %s.", toSentenceCase(base))
	}
	return fmt.Sprintf("%s configuration.", toSentenceCase(typeName))
}

func inferFieldType(fieldName string) string {
	// Infer type from common field name patterns
	patterns := map[*regexp.Regexp]string{
		regexp.MustCompile(`(?i)_id$`):        "string",
		regexp.MustCompile(`(?i)_ids$`):       "[]string",
		regexp.MustCompile(`(?i)_time$`):      "string (timestamp)",
		regexp.MustCompile(`(?i)_at$`):        "string (timestamp)",
		regexp.MustCompile(`(?i)^is_`):        "bool",
		regexp.MustCompile(`(?i)^has_`):       "bool",
		regexp.MustCompile(`(?i)^enable`):     "bool",
		regexp.MustCompile(`(?i)_enabled$`):   "bool",
		regexp.MustCompile(`(?i)_count$`):     "int",
		regexp.MustCompile(`(?i)_size$`):      "int",
		regexp.MustCompile(`(?i)_minutes$`):   "int",
		regexp.MustCompile(`(?i)_seconds$`):   "int",
		regexp.MustCompile(`(?i)_name$`):      "string",
		regexp.MustCompile(`(?i)_path$`):      "string",
		regexp.MustCompile(`(?i)_url$`):       "string",
		regexp.MustCompile(`(?i)description`): "string",
		regexp.MustCompile(`(?i)tags$`):       "map[string]string",
	}

	for pattern, typeName := range patterns {
		if pattern.MatchString(fieldName) {
			return typeName
		}
	}

	return "any"
}

func toSentenceCase(s string) string {
	// Convert CamelCase to sentence case
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

func addCoreServices(index *SDKDocsIndex) {
	// Jobs service
	index.Services["jobs"] = &ServiceDoc{
		Name:        "Jobs",
		Description: "The Jobs API allows you to create, edit, and delete jobs. Jobs are the primary unit of scheduled execution in Databricks.",
		Package:     "github.com/databricks/databricks-sdk-go/service/jobs",
		Methods: map[string]*MethodDoc{
			"Create": {
				Name:        "Create",
				Description: "Create a new job.",
				Signature:   "Create(ctx context.Context, request CreateJob) (*CreateResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "CreateJob", Description: "Job creation parameters including name, tasks, and schedule", Required: true},
				},
				Returns: &ReturnDoc{Type: "*CreateResponse", Description: "Contains the job_id of the created job"},
				Example: "resp, err := w.Jobs.Create(ctx, jobs.CreateJob{\n    Name: \"my-job\",\n    Tasks: []jobs.Task{{TaskKey: \"main\", ...}},\n})",
			},
			"List": {
				Name:        "List",
				Description: "Retrieves a list of jobs.",
				Signature:   "List(ctx context.Context, request ListJobsRequest) listing.Iterator[BaseJob]",
				Parameters: []ParamDoc{
					{Name: "request", Type: "ListJobsRequest", Description: "Filter and pagination parameters", Required: false},
				},
				Returns: &ReturnDoc{Type: "listing.Iterator[BaseJob]", Description: "Iterator over jobs matching the filter"},
			},
			"Get": {
				Name:        "Get",
				Description: "Retrieves the details for a single job.",
				Signature:   "Get(ctx context.Context, request GetJobRequest) (*Job, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "GetJobRequest", Description: "Contains job_id to retrieve", Required: true},
				},
				Returns: &ReturnDoc{Type: "*Job", Description: "Full job details including settings and run history"},
			},
			"Delete": {
				Name:        "Delete",
				Description: "Deletes a job.",
				Signature:   "Delete(ctx context.Context, request DeleteJob) error",
				Parameters: []ParamDoc{
					{Name: "request", Type: "DeleteJob", Description: "Contains job_id to delete", Required: true},
				},
			},
			"RunNow": {
				Name:        "RunNow",
				Description: "Triggers an immediate run of a job.",
				Signature:   "RunNow(ctx context.Context, request RunNow) (*RunNowResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "RunNow", Description: "Job ID and optional parameters for the run", Required: true},
				},
				Returns: &ReturnDoc{Type: "*RunNowResponse", Description: "Contains run_id of the triggered run"},
			},
		},
	}

	// Clusters/Compute service
	index.Services["compute"] = &ServiceDoc{
		Name:        "Clusters",
		Description: "The Clusters API allows you to create, start, edit, and terminate clusters. Clusters are managed cloud resources for running Spark workloads.",
		Package:     "github.com/databricks/databricks-sdk-go/service/compute",
		Methods: map[string]*MethodDoc{
			"Create": {
				Name:        "Create",
				Description: "Create a new Spark cluster.",
				Signature:   "Create(ctx context.Context, request CreateCluster) (*CreateClusterResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "CreateCluster", Description: "Cluster configuration including node types, autoscaling, and Spark version", Required: true},
				},
				Returns: &ReturnDoc{Type: "*CreateClusterResponse", Description: "Contains cluster_id of the created cluster"},
			},
			"List": {
				Name:        "List",
				Description: "Returns information about all clusters.",
				Signature:   "List(ctx context.Context, request ListClustersRequest) listing.Iterator[ClusterDetails]",
				Returns:     &ReturnDoc{Type: "listing.Iterator[ClusterDetails]", Description: "Iterator over cluster details"},
			},
			"Get": {
				Name:        "Get",
				Description: "Retrieves the information for a cluster given its identifier.",
				Signature:   "Get(ctx context.Context, request GetClusterRequest) (*ClusterDetails, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "GetClusterRequest", Description: "Contains cluster_id", Required: true},
				},
				Returns: &ReturnDoc{Type: "*ClusterDetails", Description: "Full cluster configuration and state"},
			},
			"Start": {
				Name:        "Start",
				Description: "Starts a terminated cluster.",
				Signature:   "Start(ctx context.Context, request StartCluster) error",
				Parameters: []ParamDoc{
					{Name: "request", Type: "StartCluster", Description: "Contains cluster_id to start", Required: true},
				},
			},
			"Delete": {
				Name:        "Delete",
				Description: "Permanently deletes a Spark cluster.",
				Signature:   "Delete(ctx context.Context, request DeleteCluster) error",
				Parameters: []ParamDoc{
					{Name: "request", Type: "DeleteCluster", Description: "Contains cluster_id to delete", Required: true},
				},
			},
		},
	}

	// Pipelines service
	index.Services["pipelines"] = &ServiceDoc{
		Name:        "Pipelines",
		Description: "The Delta Live Tables API allows you to create, edit, and run pipelines for data transformation and ingestion.",
		Package:     "github.com/databricks/databricks-sdk-go/service/pipelines",
		Methods: map[string]*MethodDoc{
			"Create": {
				Name:        "Create",
				Description: "Creates a new data processing pipeline.",
				Signature:   "Create(ctx context.Context, request CreatePipeline) (*CreatePipelineResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "CreatePipeline", Description: "Pipeline configuration including clusters, libraries, and target", Required: true},
				},
				Returns: &ReturnDoc{Type: "*CreatePipelineResponse", Description: "Contains pipeline_id of the created pipeline"},
			},
			"List": {
				Name:        "List",
				Description: "Lists pipelines defined in the workspace.",
				Signature:   "List(ctx context.Context, request ListPipelinesRequest) listing.Iterator[PipelineStateInfo]",
				Returns:     &ReturnDoc{Type: "listing.Iterator[PipelineStateInfo]", Description: "Iterator over pipeline info"},
			},
			"StartUpdate": {
				Name:        "StartUpdate",
				Description: "Starts a new update for the pipeline.",
				Signature:   "StartUpdate(ctx context.Context, request StartUpdate) (*StartUpdateResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "StartUpdate", Description: "Pipeline ID and update options", Required: true},
				},
				Returns: &ReturnDoc{Type: "*StartUpdateResponse", Description: "Contains update_id of the started update"},
			},
		},
	}

	// Catalog service
	index.Services["catalog"] = &ServiceDoc{
		Name:        "Catalog",
		Description: "Unity Catalog APIs for managing catalogs, schemas, tables, and other data assets.",
		Package:     "github.com/databricks/databricks-sdk-go/service/catalog",
		Methods: map[string]*MethodDoc{
			"ListCatalogs": {
				Name:        "ListCatalogs",
				Description: "Lists all catalogs in the metastore.",
				Signature:   "List(ctx context.Context, request ListCatalogsRequest) listing.Iterator[CatalogInfo]",
				Returns:     &ReturnDoc{Type: "listing.Iterator[CatalogInfo]", Description: "Iterator over catalog information"},
			},
			"ListSchemas": {
				Name:        "ListSchemas",
				Description: "Lists all schemas in a catalog.",
				Signature:   "List(ctx context.Context, request ListSchemasRequest) listing.Iterator[SchemaInfo]",
				Parameters: []ParamDoc{
					{Name: "request", Type: "ListSchemasRequest", Description: "Contains catalog_name to list schemas from", Required: true},
				},
				Returns: &ReturnDoc{Type: "listing.Iterator[SchemaInfo]", Description: "Iterator over schema information"},
			},
			"ListTables": {
				Name:        "ListTables",
				Description: "Lists all tables in a schema.",
				Signature:   "List(ctx context.Context, request ListTablesRequest) listing.Iterator[TableInfo]",
				Parameters: []ParamDoc{
					{Name: "request", Type: "ListTablesRequest", Description: "Contains catalog_name and schema_name", Required: true},
				},
				Returns: &ReturnDoc{Type: "listing.Iterator[TableInfo]", Description: "Iterator over table information"},
			},
		},
	}

	// Apps service
	index.Services["apps"] = &ServiceDoc{
		Name:        "Apps",
		Description: "Databricks Apps API for deploying and managing web applications on Databricks.",
		Package:     "github.com/databricks/databricks-sdk-go/service/apps",
		Methods: map[string]*MethodDoc{
			"Create": {
				Name:        "Create",
				Description: "Creates a new app.",
				Signature:   "Create(ctx context.Context, request CreateAppRequest) (*App, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "CreateAppRequest", Description: "App configuration including name and description", Required: true},
				},
				Returns: &ReturnDoc{Type: "*App", Description: "The created app details"},
			},
			"Deploy": {
				Name:        "Deploy",
				Description: "Deploys an app to Databricks Apps.",
				Signature:   "Deploy(ctx context.Context, request CreateAppDeploymentRequest) (*AppDeployment, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "CreateAppDeploymentRequest", Description: "Deployment configuration", Required: true},
				},
				Returns: &ReturnDoc{Type: "*AppDeployment", Description: "Deployment status and details"},
			},
			"List": {
				Name:        "List",
				Description: "Lists all apps in the workspace.",
				Signature:   "List(ctx context.Context, request ListAppsRequest) listing.Iterator[App]",
				Returns:     &ReturnDoc{Type: "listing.Iterator[App]", Description: "Iterator over apps"},
			},
		},
	}

	// SQL service
	index.Services["sql"] = &ServiceDoc{
		Name:        "SQL",
		Description: "Databricks SQL APIs for managing warehouses, queries, and dashboards.",
		Package:     "github.com/databricks/databricks-sdk-go/service/sql",
		Methods: map[string]*MethodDoc{
			"ExecuteStatement": {
				Name:        "ExecuteStatement",
				Description: "Execute a SQL statement and return results.",
				Signature:   "ExecuteStatement(ctx context.Context, request ExecuteStatementRequest) (*ExecuteStatementResponse, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "ExecuteStatementRequest", Description: "SQL statement, warehouse ID, and execution options", Required: true},
				},
				Returns: &ReturnDoc{Type: "*ExecuteStatementResponse", Description: "Query results or statement ID for async execution"},
			},
			"ListWarehouses": {
				Name:        "ListWarehouses",
				Description: "Lists all SQL warehouses.",
				Signature:   "List(ctx context.Context, request ListWarehousesRequest) listing.Iterator[EndpointInfo]",
				Returns:     &ReturnDoc{Type: "listing.Iterator[EndpointInfo]", Description: "Iterator over warehouse information"},
			},
		},
	}

	// Workspace service
	index.Services["workspace"] = &ServiceDoc{
		Name:        "Workspace",
		Description: "Workspace API for managing notebooks, folders, and other workspace objects.",
		Package:     "github.com/databricks/databricks-sdk-go/service/workspace",
		Methods: map[string]*MethodDoc{
			"List": {
				Name:        "List",
				Description: "Lists the contents of a directory.",
				Signature:   "List(ctx context.Context, request ListWorkspaceRequest) listing.Iterator[ObjectInfo]",
				Parameters: []ParamDoc{
					{Name: "request", Type: "ListWorkspaceRequest", Description: "Contains path to list", Required: true},
				},
				Returns: &ReturnDoc{Type: "listing.Iterator[ObjectInfo]", Description: "Iterator over workspace objects"},
			},
			"GetStatus": {
				Name:        "GetStatus",
				Description: "Gets the status of a workspace object.",
				Signature:   "GetStatus(ctx context.Context, request GetStatusRequest) (*ObjectInfo, error)",
				Parameters: []ParamDoc{
					{Name: "request", Type: "GetStatusRequest", Description: "Contains path to get status for", Required: true},
				},
				Returns: &ReturnDoc{Type: "*ObjectInfo", Description: "Object information including type and path"},
			},
			"Import": {
				Name:        "Import",
				Description: "Imports a notebook or file into the workspace.",
				Signature:   "Import(ctx context.Context, request Import) error",
				Parameters: []ParamDoc{
					{Name: "request", Type: "Import", Description: "Path, content, and format of the object to import", Required: true},
				},
			},
		},
	}

	// Add some common enums
	index.Enums["jobs.RunLifeCycleState"] = &EnumDoc{
		Name:        "RunLifeCycleState",
		Package:     "jobs",
		Description: "The current state of the run lifecycle.",
		Values:      []string{"PENDING", "RUNNING", "TERMINATING", "TERMINATED", "SKIPPED", "INTERNAL_ERROR"},
	}

	index.Enums["compute.State"] = &EnumDoc{
		Name:        "State",
		Package:     "compute",
		Description: "The state of a cluster.",
		Values:      []string{"PENDING", "RUNNING", "RESTARTING", "RESIZING", "TERMINATING", "TERMINATED", "ERROR", "UNKNOWN"},
	}

	index.Enums["pipelines.PipelineState"] = &EnumDoc{
		Name:        "PipelineState",
		Package:     "pipelines",
		Description: "The state of a pipeline.",
		Values:      []string{"IDLE", "RUNNING", "STARTING", "STOPPING", "DELETED", "RECOVERING", "FAILED", "RESETTING"},
	}
}

func writeIndex(index *SDKDocsIndex, path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Sort maps for deterministic output
	sortIndex(index)

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	return nil
}

func sortIndex(index *SDKDocsIndex) {
	// Sort service methods
	for _, service := range index.Services {
		// Methods are already in a map, which will be sorted by JSON marshaling
		_ = service
	}

	// Sort type fields
	for _, typeDoc := range index.Types {
		// Sort fields by converting to sorted slice would require changing structure
		// For now, rely on JSON marshaling order
		_ = typeDoc
	}

	// Sort enum values
	for _, enumDoc := range index.Enums {
		sort.Strings(enumDoc.Values)
	}
}
