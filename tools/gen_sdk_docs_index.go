// Package main generates SDK documentation index for MCP tools.
//
// Usage:
//
//	go run tools/gen_sdk_docs_index.go -output experimental/aitools/lib/providers/sdkdocs/
//
// This tool parses the Go SDK source code to generate a comprehensive SDK documentation
// index that is embedded into the CLI binary. It extracts service interfaces, method
// signatures, type definitions, and enums directly from the SDK.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
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
	SDKVersion  string                 `json:"sdk_version"`
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
	Description string `yaml:"description"`
}

// Priority services to include (most commonly used)
var priorityServices = map[string]bool{
	"apps":      true,
	"catalog":   true,
	"compute":   true,
	"files":     true,
	"iam":       true,
	"jobs":      true,
	"ml":        true,
	"pipelines": true,
	"serving":   true,
	"sql":       true,
	"workspace": true,
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

	// Find SDK path
	sdkPath, sdkVersion, err := findSDKPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding SDK path: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found SDK at: %s (version %s)\n", sdkPath, sdkVersion)

	// Load annotations for additional type descriptions
	annotations, err := loadAnnotations(filepath.Join(projectRoot, *annotationsPath))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load annotations: %v\n", err)
		annotations = make(AnnotationsFile)
	}

	// Generate index from SDK
	index, err := generateIndexFromSDK(sdkPath, sdkVersion, annotations)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating index: %v\n", err)
		os.Exit(1)
	}

	// Write output
	var outputPath string
	if filepath.IsAbs(*outputDir) {
		outputPath = filepath.Join(*outputDir, "sdk_docs_index.json")
	} else {
		outputPath = filepath.Join(projectRoot, *outputDir, "sdk_docs_index.json")
	}
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

func findSDKPath() (string, string, error) {
	// Use go list to find the SDK module path
	cmd := exec.Command("go", "list", "-m", "-json", "github.com/databricks/databricks-sdk-go")
	output, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to find SDK: %w", err)
	}

	var modInfo struct {
		Dir     string `json:"Dir"`
		Version string `json:"Version"`
	}
	if err := json.Unmarshal(output, &modInfo); err != nil {
		return "", "", fmt.Errorf("failed to parse module info: %w", err)
	}

	return modInfo.Dir, modInfo.Version, nil
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

func generateIndexFromSDK(sdkPath, sdkVersion string, annotations AnnotationsFile) (*SDKDocsIndex, error) {
	index := &SDKDocsIndex{
		Version:     "1.0",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		SDKVersion:  sdkVersion,
		Services:    make(map[string]*ServiceDoc),
		Types:       make(map[string]*TypeDoc),
		Enums:       make(map[string]*EnumDoc),
	}

	servicePath := filepath.Join(sdkPath, "service")
	entries, err := os.ReadDir(servicePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgName := entry.Name()

		// Skip non-priority services to keep index manageable
		if !priorityServices[pkgName] {
			continue
		}

		pkgPath := filepath.Join(servicePath, pkgName)

		// Parse the package
		serviceDoc, types, enums, err := parseServicePackage(pkgPath, pkgName)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to parse %s: %v\n", pkgName, err)
			continue
		}

		if serviceDoc != nil && len(serviceDoc.Methods) > 0 {
			index.Services[pkgName] = serviceDoc
		}

		// Add types
		for typeName, typeDoc := range types {
			index.Types[pkgName+"."+typeName] = typeDoc
		}

		// Add enums
		for enumName, enumDoc := range enums {
			index.Enums[pkgName+"."+enumName] = enumDoc
		}
	}

	// Enrich with annotations
	enrichWithAnnotations(index, annotations)

	return index, nil
}

func parseServicePackage(pkgPath, pkgName string) (*ServiceDoc, map[string]*TypeDoc, map[string]*EnumDoc, error) {
	fset := token.NewFileSet()

	// Parse all Go files in the package
	pkgs, err := parser.ParseDir(fset, pkgPath, func(fi os.FileInfo) bool {
		// Skip test files
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to parse package: %w", err)
	}

	var serviceDoc *ServiceDoc
	types := make(map[string]*TypeDoc)
	enums := make(map[string]*EnumDoc)

	for _, pkg := range pkgs {
		// Create doc package for better comment extraction
		docPkg := doc.New(pkg, pkgPath, doc.AllDecls)

		// Find service interface
		for _, typ := range docPkg.Types {
			if strings.HasSuffix(typ.Name, "Service") && !strings.HasSuffix(typ.Name, "Interface") {
				// This is a service interface
				if serviceDoc == nil {
					serviceDoc = parseServiceInterface(typ, pkgName, fset, pkg)
				}
			} else if isEnumType(typ, pkg) {
				// This is an enum
				enumDoc := parseEnumType(typ, pkgName, pkg)
				if enumDoc != nil && len(enumDoc.Values) > 0 {
					enums[typ.Name] = enumDoc
				}
			} else if isStructType(typ, pkg) {
				// This is a struct type
				typeDoc := parseStructType(typ, pkgName, fset, pkg)
				if typeDoc != nil {
					types[typ.Name] = typeDoc
				}
			}
		}
	}

	return serviceDoc, types, enums, nil
}

func parseServiceInterface(typ *doc.Type, pkgName string, fset *token.FileSet, pkg *ast.Package) *ServiceDoc {
	serviceDoc := &ServiceDoc{
		Name:        strings.TrimSuffix(typ.Name, "Service"),
		Description: cleanDescription(typ.Doc),
		Package:     fmt.Sprintf("github.com/databricks/databricks-sdk-go/service/%s", pkgName),
		Methods:     make(map[string]*MethodDoc),
	}

	// Find the interface declaration
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typ.Name {
					continue
				}

				ifaceType, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				// Parse methods
				for _, method := range ifaceType.Methods.List {
					if len(method.Names) == 0 {
						continue
					}

					methodName := method.Names[0].Name
					funcType, ok := method.Type.(*ast.FuncType)
					if !ok {
						continue
					}

					methodDoc := parseMethod(methodName, funcType, method.Doc, fset)
					if methodDoc != nil {
						serviceDoc.Methods[methodName] = methodDoc
					}
				}
			}
		}
	}

	return serviceDoc
}

func parseMethod(name string, funcType *ast.FuncType, doc *ast.CommentGroup, fset *token.FileSet) *MethodDoc {
	methodDoc := &MethodDoc{
		Name:        name,
		Description: cleanDescription(extractCommentText(doc)),
		Parameters:  []ParamDoc{},
	}

	// Build signature
	var sig bytes.Buffer
	sig.WriteString(name)
	sig.WriteString("(")

	// Parse parameters
	if funcType.Params != nil {
		params := []string{}
		for _, field := range funcType.Params.List {
			typeStr := typeToString(field.Type, fset)

			for _, name := range field.Names {
				params = append(params, fmt.Sprintf("%s %s", name.Name, typeStr))

				// Skip context parameter
				if name.Name == "ctx" {
					continue
				}

				methodDoc.Parameters = append(methodDoc.Parameters, ParamDoc{
					Name:     name.Name,
					Type:     typeStr,
					Required: true,
				})
			}

			// Handle unnamed parameters
			if len(field.Names) == 0 {
				params = append(params, typeStr)
			}
		}
		sig.WriteString(strings.Join(params, ", "))
	}
	sig.WriteString(")")

	// Parse return type
	if funcType.Results != nil && len(funcType.Results.List) > 0 {
		returns := []string{}
		for _, field := range funcType.Results.List {
			typeStr := typeToString(field.Type, fset)
			returns = append(returns, typeStr)
		}

		if len(returns) == 1 {
			sig.WriteString(" ")
			sig.WriteString(returns[0])
			if returns[0] != "error" {
				methodDoc.Returns = &ReturnDoc{Type: returns[0]}
			}
		} else {
			sig.WriteString(" (")
			sig.WriteString(strings.Join(returns, ", "))
			sig.WriteString(")")
			// Find non-error return type
			for _, ret := range returns {
				if ret != "error" {
					methodDoc.Returns = &ReturnDoc{Type: ret}
					break
				}
			}
		}
	}

	methodDoc.Signature = sig.String()
	return methodDoc
}

func isEnumType(typ *doc.Type, pkg *ast.Package) bool {
	// Check if type has constants defined (enum pattern)
	return len(typ.Consts) > 0
}

func isStructType(typ *doc.Type, pkg *ast.Package) bool {
	// Check if this is a struct type
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typ.Name {
					continue
				}

				_, isStruct := typeSpec.Type.(*ast.StructType)
				return isStruct
			}
		}
	}
	return false
}

func parseEnumType(typ *doc.Type, pkgName string, pkg *ast.Package) *EnumDoc {
	enumDoc := &EnumDoc{
		Name:        typ.Name,
		Package:     pkgName,
		Description: cleanDescription(typ.Doc),
		Values:      []string{},
	}

	// Extract enum values from constants
	for _, c := range typ.Consts {
		for _, name := range c.Names {
			// Extract the value part after the type prefix
			value := strings.TrimPrefix(name, typ.Name)
			if value != name && value != "" {
				enumDoc.Values = append(enumDoc.Values, value)
			}
		}
	}

	return enumDoc
}

func parseStructType(typ *doc.Type, pkgName string, fset *token.FileSet, pkg *ast.Package) *TypeDoc {
	typeDoc := &TypeDoc{
		Name:        typ.Name,
		Package:     pkgName,
		Description: cleanDescription(typ.Doc),
		Fields:      make(map[string]*FieldDoc),
	}

	// Find the struct declaration
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}

			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name != typ.Name {
					continue
				}

				structType, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}

				// Parse fields
				for _, field := range structType.Fields.List {
					if len(field.Names) == 0 {
						continue
					}

					fieldName := field.Names[0].Name

					// Skip internal fields
					if fieldName == "ForceSendFields" {
						continue
					}

					// Get JSON name from tag
					jsonName := fieldName
					if field.Tag != nil {
						jsonName = extractJSONName(field.Tag.Value)
						if jsonName == "" || jsonName == "-" {
							continue
						}
					}

					fieldDoc := &FieldDoc{
						Name:        jsonName,
						Type:        typeToString(field.Type, fset),
						Description: cleanDescription(extractCommentText(field.Doc)),
					}

					typeDoc.Fields[jsonName] = fieldDoc
				}
			}
		}
	}

	// Only return if we have fields
	if len(typeDoc.Fields) == 0 {
		return nil
	}

	return typeDoc
}

func typeToString(expr ast.Expr, fset *token.FileSet) string {
	var buf bytes.Buffer
	printer.Fprint(&buf, fset, expr)
	return buf.String()
}

func extractCommentText(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	return cg.Text()
}

func extractJSONName(tag string) string {
	// Parse struct tag to get JSON name
	// Tag format: `json:"name,omitempty"`
	re := regexp.MustCompile(`json:"([^",]+)`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func cleanDescription(s string) string {
	// Clean up description text
	s = strings.TrimSpace(s)

	// Remove "Deprecated:" notices for cleaner output
	if idx := strings.Index(s, "\n\nDeprecated:"); idx > 0 {
		s = s[:idx]
	}

	// Truncate very long descriptions
	if len(s) > 500 {
		s = s[:497] + "..."
	}

	return s
}

func enrichWithAnnotations(index *SDKDocsIndex, annotations AnnotationsFile) {
	// Add type descriptions from annotations
	for fullTypeName, fields := range annotations {
		typeName := extractTypeName(fullTypeName)
		packageName := extractPackageName(fullTypeName)

		if typeName == "" {
			continue
		}

		service := inferServiceFromPackage(packageName)
		typePath := service + "." + typeName

		// Check if type already exists
		if existing, ok := index.Types[typePath]; ok {
			// Enrich existing type with annotation descriptions
			for fieldName, annotation := range fields {
				if fieldName == "_" && annotation.Description != "" {
					existing.Description = annotation.Description
					continue
				}
				if field, ok := existing.Fields[fieldName]; ok && field.Description == "" {
					field.Description = annotation.Description
				}
			}
		} else {
			// Create new type from annotations
			typeDoc := &TypeDoc{
				Name:        typeName,
				Package:     packageName,
				Description: "",
				Fields:      make(map[string]*FieldDoc),
			}

			for fieldName, annotation := range fields {
				if fieldName == "_" {
					typeDoc.Description = annotation.Description
					continue
				}
				typeDoc.Fields[fieldName] = &FieldDoc{
					Name:        fieldName,
					Type:        "any",
					Description: annotation.Description,
				}
			}

			if len(typeDoc.Fields) > 0 || typeDoc.Description != "" {
				index.Types[typePath] = typeDoc
			}
		}
	}
}

func extractTypeName(fullPath string) string {
	parts := strings.Split(fullPath, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractPackageName(fullPath string) string {
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
		"files":     "files",
	}

	if service, ok := serviceMap[packageName]; ok {
		return service
	}
	return packageName
}

func writeIndex(index *SDKDocsIndex, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Sort for deterministic output
	sortIndex(index)

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
	for _, enumDoc := range index.Enums {
		sort.Strings(enumDoc.Values)
	}
}
