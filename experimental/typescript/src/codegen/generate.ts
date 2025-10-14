#!/usr/bin/env tsx
/**
 * Code generator for Databricks Asset Bundles resource types
 *
 * Generates TypeScript types from JSON Schema located at:
 * ../../bundle/schema/jsonschema.json
 */

import { readFileSync, writeFileSync, mkdirSync, existsSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));

// These are the main resource types that users define in bundles
// Other resources.* types are helper types (permissions, lifecycle, etc.)
const MAIN_RESOURCES = [
  "resources.App",
  "resources.Cluster",
  "resources.Dashboard",
  "resources.Job",
  "resources.MlflowExperiment",
  "resources.MlflowModel",
  "resources.ModelServingEndpoint",
  "resources.Pipeline",
  "resources.QualityMonitor",
  "resources.RegisteredModel",
  "resources.Schema",
  "resources.Volume",
  "resources.DatabaseInstance",
  "resources.DatabaseCatalog",
  "resources.SqlWarehouse",
];

// Namespaces to load from the schema
const LOADED_NAMESPACES = [
  "compute",
  "jobs",
  "pipelines",
  "resources",
  "catalog",
  "ml",
  "serving",
  "dashboards",
  "apps",
  "database",
  "database_catalogs",
  "sql_warehouses",
  "sql",
];

// Type renames
const TYPE_RENAMES: Record<string, string> = {
  string: "string",
  boolean: "boolean",
  integer: "number",
  number: "number",
  int: "number",
  int64: "number",
  float64: "number",
  bool: "boolean",
};

const PRIMITIVES = ["string", "boolean", "integer", "number", "bool", "int", "int64", "float64"];

// TypeScript reserved words that need to be escaped
const RESERVED_WORDS = new Set([
  "interface",
  "type",
  "enum",
  "class",
  "function",
  "const",
  "let",
  "var",
  "return",
  "if",
  "else",
  "for",
  "while",
  "do",
  "switch",
  "case",
  "break",
  "continue",
  "try",
  "catch",
  "finally",
  "throw",
  "new",
  "this",
  "super",
  "extends",
  "implements",
  "import",
  "export",
  "default",
  "as",
  "from",
  "typeof",
  "instanceof",
  "void",
  "null",
  "undefined",
  "true",
  "false",
]);

interface Property {
  ref: string;
  description?: string;
  deprecated?: boolean;
  stage?: string;
}

interface Schema {
  type: "object" | "string";
  enum?: string[];
  properties?: Record<string, Property>;
  required?: string[];
  description?: string;
  deprecated?: boolean;
  stage?: string;
}

/**
 * Load JSON schema from file
 */
function loadSchema(): any {
  const schemaPath = join(__dirname, "..", "..", "..", "..", "bundle", "schema", "jsonschema.json");
  return JSON.parse(readFileSync(schemaPath, "utf-8"));
}

/**
 * Navigate to a path in the schema
 */
function getSchemaPath(spec: any, path: string[]): any {
  let current = spec;
  for (const key of path) {
    current = current[key];
  }
  return current;
}

/**
 * Check if a reference should be loaded
 */
function shouldLoadRef(ref: string): boolean {
  const name = ref.split("/").pop()!;

  for (const namespace of LOADED_NAMESPACES) {
    if (name.startsWith(`${namespace}.`)) {
      return true;
    }
  }

  return PRIMITIVES.includes(name);
}

/**
 * Unwrap variable pattern from schema
 * Schema fields can be either the actual type or a variable reference string
 */
function unwrapVariable(schema: any): any | null {
  const anyOf = schema.anyOf || schema.oneOf;

  if (!anyOf || anyOf.length !== 2) {
    return null;
  }

  const [primary, variable] = anyOf;
  const pattern = variable.pattern || "";
  const type = variable.type || "";

  if (type === "string" && pattern.startsWith("\\$\\{") && pattern.endsWith("\\}")) {
    return primary;
  }

  return null;
}

/**
 * Parse a schema definition
 */
function parseSchema(schema: any): Schema {
  const unwrapped = unwrapVariable(schema) || schema;
  const properties: Record<string, Property> = {};

  for (const [key, value] of Object.entries(unwrapped.properties || {})) {
    const v = value as any;

    // Properties should only have $ref
    if (!v.$ref) {
      continue;
    }

    properties[key] = {
      ref: v.$ref,
      description: v.description,
      deprecated: v.deprecated,
      stage: v["x-databricks-preview"],
    };
  }

  return {
    type: unwrapped.type,
    enum: unwrapped.enum,
    properties,
    required: unwrapped.required || [],
    description: unwrapped.description,
    deprecated: unwrapped.deprecated,
    stage: unwrapped["x-databricks-preview"],
  };
}

/**
 * Get all schemas for generation
 */
function getSchemas(): Map<string, Schema> {
  const spec = loadSchema();
  const output = new Map<string, Schema>();

  // Get SDK types and resource types
  const sdkTypesSpec = getSchemaPath(spec, [
    "$defs",
    "github.com",
    "databricks",
    "databricks-sdk-go",
    "service",
  ]);

  const resourceTypesSpec = getSchemaPath(spec, [
    "$defs",
    "github.com",
    "databricks",
    "cli",
    "bundle",
    "config",
  ]);

  // Merge specs
  const flatSpec = { ...sdkTypesSpec, ...resourceTypesSpec };

  // Filter to only supported types
  const filteredSpec = Object.fromEntries(
    Object.entries(flatSpec).filter(([key]) => shouldLoadRef(key))
  );

  // Parse schemas
  for (const [name, schema] of Object.entries(filteredSpec)) {
    try {
      output.set(name, parseSchema(schema));
    } catch (error) {
      console.error(`Failed to parse schema for ${name}:`, error);
    }
  }

  return output;
}

/**
 * Get TypeScript type name from ref
 */
function getTypeName(ref: string): string {
  const name = ref.split("/").pop()!.split(".").pop()!;
  const renamed = TYPE_RENAMES[name] || name;

  // If the type name is a reserved word, convert it to "any" for simplicity
  if (RESERVED_WORDS.has(renamed)) {
    return "any";
  }

  return renamed;
}

/**
 * Convert resource name to namespace
 * E.g., "resources.Job" -> "jobs"
 * E.g., "resources.MlflowExperiment" -> "mlflow_experiments"
 */
function getResourceNamespace(resourceName: string): string {
  const typeName = resourceName.split(".").pop()!;

  // Convert CamelCase to snake_case
  const snakeCase = typeName
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1_$2")
    .replace(/([a-z\d])([A-Z])/g, "$1_$2")
    .toLowerCase();

  // Pluralize (simple rules)
  if (snakeCase.endsWith("s") || snakeCase.endsWith("x")) {
    return snakeCase + "es";
  } else if (snakeCase.endsWith("y")) {
    return snakeCase.slice(0, -1) + "ies";
  } else {
    return snakeCase + "s";
  }
}

/**
 * Convert snake_case to camelCase
 */
function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_match, char) => char.toUpperCase());
}

/**
 * Convert ref to import path and type name
 */
function resolveRef(
  ref: string,
  schemas: Map<string, Schema>
): { typeName: string; import?: string } {
  // Handle $defs references
  if (ref.startsWith("#/$defs/")) {
    const refName = ref.slice(8); // Remove "#/$defs/"

    // Check if it's a primitive
    if (PRIMITIVES.includes(refName)) {
      return { typeName: TYPE_RENAMES[refName] || refName };
    }

    // Check for slice/ or map/ prefixes
    if (refName.startsWith("slice/")) {
      const innerRef = refName.slice(6);
      const inner = resolveRef(`#/$defs/${innerRef}`, schemas);
      return { typeName: `${inner.typeName}[]`, import: inner.import };
    }

    if (refName.startsWith("map/")) {
      const innerRef = refName.slice(4);
      const inner = resolveRef(`#/$defs/${innerRef}`, schemas);
      return { typeName: `Record<string, ${inner.typeName}>`, import: inner.import };
    }

    // Regular type reference
    const typeName = getTypeName(ref);
    return { typeName };
  }

  return { typeName: "unknown" };
}

/**
 * Generate TypeScript interface from schema
 * @param isMainResource - If true, generates params interface + class; otherwise generates regular interface
 */
function generateInterface(
  name: string,
  namespace: string,
  schema: Schema,
  schemas: Map<string, Schema>,
  isMainResource: boolean = false
): string {
  const lines: string[] = [];

  // Add description if present
  if (schema.description) {
    lines.push("/**");
    lines.push(` * ${schema.description.replace(/\n/g, "\n * ")}`);
    if (schema.deprecated) {
      lines.push(" * @deprecated");
    }
    lines.push(" */");
  }

  // Handle enums
  if (schema.type === "string" && schema.enum && schema.enum.length > 0) {
    lines.push(`export type ${name} =`);
    for (let i = 0; i < schema.enum.length; i++) {
      const isLast = i === schema.enum.length - 1;
      lines.push(`  | "${schema.enum[i]}"${isLast ? ";" : ""}`);
    }
    lines.push("");
    return lines.join("\n");
  }

  if (isMainResource) {
    // For main resources: generate params interface + class
    const paramsName = `${name}Params`;

    // Generate params interface
    lines.push(`export interface ${paramsName} {`);

    if (schema.properties) {
      for (const [propName, prop] of Object.entries(schema.properties)) {
        const isRequired = schema.required?.includes(propName);
        const optional = isRequired ? "" : "?";

        // Add property description
        if (prop.description) {
          lines.push("  /**");
          lines.push(`   * ${prop.description.replace(/\n/g, "\n   * ")}`);
          if (prop.deprecated) {
            lines.push("   * @deprecated");
          }
          lines.push("   */");
        }

        const resolved = resolveRef(prop.ref, schemas);
        lines.push(`  ${propName}${optional}: VariableOr<${resolved.typeName}>;`);
      }
    }

    lines.push("}");
    lines.push("");

    // Generate class
    lines.push(`export class ${name} extends Resource<${paramsName}> {`);
    lines.push(`  constructor(name: string, params: ${paramsName}) {`);
    lines.push(`    super(name, params, "${namespace}");`);
    lines.push(`  }`);
    lines.push("}");
    lines.push("");
  } else {
    // For supporting types: generate regular interface
    lines.push(`export interface ${name} {`);

    if (schema.properties) {
      for (const [propName, prop] of Object.entries(schema.properties)) {
        const isRequired = schema.required?.includes(propName);
        const optional = isRequired ? "" : "?";

        // Add property description
        if (prop.description) {
          lines.push("  /**");
          lines.push(`   * ${prop.description.replace(/\n/g, "\n   * ")}`);
          if (prop.deprecated) {
            lines.push("   * @deprecated");
          }
          lines.push("   */");
        }

        const resolved = resolveRef(prop.ref, schemas);
        lines.push(`  ${propName}${optional}: VariableOr<${resolved.typeName}>;`);
      }
    }

    lines.push("}");
    lines.push("");
  }

  return lines.join("\n");
}

/**
 * Recursively collect all referenced types
 */
function collectReferencedTypes(
  schema: Schema,
  schemas: Map<string, Schema>,
  visited: Set<string> = new Set()
): Set<string> {
  const types = new Set<string>();

  if (!schema.properties) {
    return types;
  }

  for (const prop of Object.values(schema.properties)) {
    const refName = prop.ref.split("/").pop()!;

    // Skip primitives and already visited
    if (PRIMITIVES.includes(refName) || visited.has(refName)) {
      continue;
    }

    // Handle slice/ and map/ prefixes
    if (refName.startsWith("slice/")) {
      const innerRef = refName.slice(6);
      if (!PRIMITIVES.includes(innerRef) && !visited.has(innerRef)) {
        types.add(innerRef);
        visited.add(innerRef);

        const innerSchema = schemas.get(innerRef);
        if (innerSchema) {
          const nested = collectReferencedTypes(innerSchema, schemas, visited);
          nested.forEach((t) => types.add(t));
        }
      }
      continue;
    }

    if (refName.startsWith("map/")) {
      const innerRef = refName.slice(4);
      if (!PRIMITIVES.includes(innerRef) && !visited.has(innerRef)) {
        types.add(innerRef);
        visited.add(innerRef);

        const innerSchema = schemas.get(innerRef);
        if (innerSchema) {
          const nested = collectReferencedTypes(innerSchema, schemas, visited);
          nested.forEach((t) => types.add(t));
        }
      }
      continue;
    }

    // Regular type reference
    types.add(refName);
    visited.add(refName);

    // Recursively collect types from the referenced schema
    const refSchema = schemas.get(refName);
    if (refSchema) {
      const nested = collectReferencedTypes(refSchema, schemas, visited);
      nested.forEach((t) => types.add(t));
    }
  }

  return types;
}

/**
 * Generate TypeScript file for a resource namespace
 */
function generateResourceFile(
  resourceName: string,
  namespace: string,
  schemas: Map<string, Schema>
): void {
  const schema = schemas.get(resourceName);
  if (!schema) {
    console.error(`Schema not found for ${resourceName}`);
    return;
  }

  const typeName = resourceName.split(".").pop()!;
  const lines: string[] = [];

  // File header
  lines.push("/**");
  lines.push(` * ${typeName} resource types for Databricks Asset Bundles`);
  lines.push(" *");
  lines.push(" * Auto-generated from JSON Schema. Do not edit manually.");
  lines.push(" */");
  lines.push("");

  // Imports - need Resource as a value for class extension
  lines.push('import { Resource } from "../../src/core/resource.js";');
  lines.push('import type { VariableOr } from "../../src/core/variable.js";');
  lines.push("");

  // Collect all referenced types recursively
  const referencedTypes = collectReferencedTypes(schema, schemas);

  // Generate main resource class (with params interface)
  lines.push(generateInterface(typeName, namespace, schema, schemas, true));

  // Generate referenced types in sorted order (regular interfaces)
  for (const refName of Array.from(referencedTypes).sort()) {
    const refSchema = schemas.get(refName);
    if (refSchema) {
      lines.push(
        generateInterface(getTypeName(`#/$defs/${refName}`), namespace, refSchema, schemas, false)
      );
    }
  }

  // Write file
  const outputDir = join(__dirname, "..", "..", "generated", namespace);
  if (!existsSync(outputDir)) {
    mkdirSync(outputDir, { recursive: true });
  }

  const outputPath = join(outputDir, "index.ts");
  writeFileSync(outputPath, lines.join("\n"), "utf-8");
  console.log(`Generated ${outputPath}`);
}

/**
 * Generate manifest file with resource metadata
 */
function generateManifest(): void {
  const resourceTypes: Array<{ name: string; namespace: string; typeName: string }> = [];

  for (const resourceName of MAIN_RESOURCES) {
    const namespace = getResourceNamespace(resourceName);
    const typeName = resourceName.split(".").pop()!;
    resourceTypes.push({ name: resourceName, namespace, typeName });
  }

  const lines: string[] = [];
  lines.push("/**");
  lines.push(" * Auto-generated manifest of all Databricks resource types");
  lines.push(" *");
  lines.push(" * DO NOT EDIT MANUALLY - Generated by src/codegen/generate.ts");
  lines.push(" */");
  lines.push("");

  // Export resource type info
  lines.push("export interface ResourceTypeInfo {");
  lines.push("  name: string;");
  lines.push("  namespace: string;");
  lines.push("  typeName: string;");
  lines.push("}");
  lines.push("");

  lines.push("export const RESOURCE_TYPES: readonly ResourceTypeInfo[] = [");
  for (const { name, namespace, typeName } of resourceTypes) {
    lines.push(`  { name: "${name}", namespace: "${namespace}", typeName: "${typeName}" },`);
  }
  lines.push("] as const;");
  lines.push("");

  lines.push("export const RESOURCE_NAMESPACES = [");
  for (const { namespace } of resourceTypes) {
    lines.push(`  "${namespace}",`);
  }
  lines.push("] as const;");
  lines.push("");

  lines.push("export const RESOURCE_TYPE_NAMES = [");
  for (const { typeName } of resourceTypes) {
    lines.push(`  "${typeName}",`);
  }
  lines.push("] as const;");
  lines.push("");

  const outputPath = join(__dirname, "..", "..", "generated", "manifest.ts");
  const outputDir = dirname(outputPath);
  if (!existsSync(outputDir)) {
    mkdirSync(outputDir, { recursive: true });
  }
  writeFileSync(outputPath, lines.join("\n"), "utf-8");
  console.log(`Generated ${outputPath}`);
}

/**
 * Main generation function
 */
function main() {
  console.log("Loading JSON schema...");
  const schemas = getSchemas();
  console.log(`Loaded ${schemas.size} schemas`);

  console.log("\nGenerating resource types...");
  for (const resourceName of MAIN_RESOURCES) {
    const namespace = getResourceNamespace(resourceName);
    console.log(`\nGenerating ${resourceName} -> ${namespace}`);
    generateResourceFile(resourceName, namespace, schemas);
  }

  console.log("\nGenerating manifest...");
  generateManifest();
  console.log("\n Code generation complete!");
}

// Run generator
main();
