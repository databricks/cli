/**
 * Build system for CLI integration
 *
 * This module provides the main entry point for the Databricks CLI to execute
 * TypeScript-based bundle definitions. It handles:
 * - JSON input/output for CLI communication
 * - Dynamic module loading
 * - Phase execution (load_resources, apply_mutators)
 * - Diagnostics reporting
 */

import { readFile, writeFile } from "fs/promises";
import { pathToFileURL } from "url";
import { relative, isAbsolute } from "path";
import type { Bundle } from "../core/bundle.js";
import { Diagnostics } from "../core/diagnostics.js";
import { Location } from "../core/location.js";
import { Resources } from "../core/resources.js";
import type { ResourceMutator } from "./resource-mutator.js";

/**
 * Command line arguments
 */
export interface BuildArgs {
  phase: "load_resources" | "apply_mutators";
  input: string;
  output: string;
  diagnostics: string;
  locations?: string;
}

/**
 * Configuration from databricks.yml
 */
export interface BuildConfig {
  resources?: string[];
  mutators?: string[];
}

/**
 * Bundle input JSON from CLI
 */
export interface BundleInput {
  bundle?: {
    target?: string;
    [key: string]: unknown;
  };
  variables?: Record<string, { value?: unknown; [key: string]: unknown }>;
  resources?: Record<string, Record<string, unknown>>;
  python?: BuildConfig;
  experimental?: {
    python?: BuildConfig;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

/**
 * Location output format for CLI
 */
export interface LocationOutput {
  path: string;
  file: string;
  line?: number;
  column?: number;
}

/**
 * Parse command line arguments
 */
export function parseArgs(args: string[]): BuildArgs | "help" {
  // Check for help flag
  if (args.includes("--help") || args.includes("-h")) {
    return "help";
  }

  const parsed: Partial<BuildArgs> = {};

  for (let i = 0; i < args.length; i += 2) {
    const key = args[i];
    const value = args[i + 1];

    if (key === "--phase") {
      if (value !== "load_resources" && value !== "apply_mutators") {
        throw new Error(`Invalid phase: ${value}`);
      }
      parsed.phase = value;
    } else if (key === "--input") {
      parsed.input = value;
    } else if (key === "--output") {
      parsed.output = value;
    } else if (key === "--diagnostics") {
      parsed.diagnostics = value;
    } else if (key === "--locations") {
      parsed.locations = value;
    }
  }

  if (!parsed.phase) {
    throw new Error("Missing required argument --phase");
  }
  if (!parsed.input) {
    throw new Error("Missing required argument --input");
  }
  if (!parsed.output) {
    throw new Error("Missing required argument --output");
  }
  if (!parsed.diagnostics) {
    throw new Error("Missing required argument --diagnostics");
  }

  return parsed as BuildArgs;
}

/**
 * Read configuration from bundle input
 */
export function readConfig(input: BundleInput): [BuildConfig, Diagnostics] {
  const experimental = input.experimental || {};

  const experimentalConfig = experimental.javascript || {};
  const config = input.javascript || {};

  const hasConfig = Object.keys(config).length > 0;
  const hasExperimentalConfig = Object.keys(experimentalConfig).length > 0;

  if (hasConfig && hasExperimentalConfig) {
    // Check if configs are equal
    if (JSON.stringify(config) !== JSON.stringify(experimentalConfig)) {
      return [
        {},
        Diagnostics.createError(
          "Both 'javascript' and 'experimental/javascript' sections are present, use 'javascript' section only",
          { path: ["experimental", "javascript"] }
        ),
      ];
    }
  }

  // Prefer non-experimental config
  const finalConfig = hasConfig ? config : experimentalConfig;

  return [finalConfig, new Diagnostics()];
}

/**
 * Parse bundle information from input
 */
export function parseBundleInfo(input: BundleInput): Bundle {
  const bundleInfo = input.bundle || {};
  const variables: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(input.variables || {})) {
    if (typeof value === "object" && value !== null && "value" in value) {
      variables[key] = value.value;
    }
  }

  return {
    target: bundleInfo.target || "default",
    variables,
  } as Bundle;
}

/**
 * Load a function or object from a module
 */
export async function loadObject(qualifiedName: string): Promise<[unknown, Diagnostics]> {
  const parts = qualifiedName.split(":");
  if (parts.length !== 2) {
    return [
      null,
      Diagnostics.createError(
        `Invalid qualified name '${qualifiedName}', expected format 'module:name'`
      ),
    ];
  }

  const modulePath = parts[0]!;
  const name = parts[1]!;

  try {
    // Convert relative path to absolute file URL for ESM import
    const moduleUrl = modulePath.startsWith("./") || modulePath.startsWith("../")
      ? pathToFileURL(modulePath).href
      : modulePath;

    const module = await import(moduleUrl);

    if (!(name in module)) {
      return [
        null,
        Diagnostics.createError(`Name '${name}' not found in module '${modulePath}'`),
      ];
    }

    return [module[name], new Diagnostics()];
  } catch (error) {
    if (error instanceof Error && error.message.includes("Cannot find module")) {
      if (qualifiedName === "resources:loadResources") {
        return [null, explainCommonImportError(error)];
      }
    }

    const errorMessage = error instanceof Error ? error.message : String(error);
    return [
      null,
      Diagnostics.createError(`Failed to import module '${modulePath}': ${errorMessage}`),
    ];
  }
}

/**
 * Provide helpful error message for common import errors
 */
function explainCommonImportError(_error: Error): Diagnostics {
  const explanation = `Make sure to create a new TypeScript file at resources/index.ts with contents:

import { defineResources } from "@databricks/bundles";

export const loadResources = defineResources((bundle) => {
  // Define your resources here
  return new Resources();
});
`;

  return Diagnostics.createError(
    "Can't find function 'loadResources' in module 'resources'",
    { detail: explanation }
  );
}

/**
 * Load resource functions
 */
export async function loadFunctions(names: string[]): Promise<[((bundle: Bundle) => Promise<Resources>)[], Diagnostics]> {
  let diagnostics = new Diagnostics();
  const functions: ((bundle: Bundle) => Promise<Resources>)[] = [];

  for (const name of names) {
    const [obj, diag] = await loadObject(name);
    diagnostics = diagnostics.extend(diag);

    if (diag.hasError()) {
      continue;
    }

    if (typeof obj !== "function") {
      const [, funcName] = name.split(":");
      diagnostics = diagnostics.extend(
        Diagnostics.createError(`'${funcName}' in module is not a function`)
      );
      continue;
    }

    functions.push(obj as ((bundle: Bundle) => Promise<Resources>));
  }

  return [functions, diagnostics];
}

/**
 * Load resource mutators
 */
export async function loadResourceMutators(
  names: string[]
): Promise<[ResourceMutator[], Diagnostics]> {
  let diagnostics = new Diagnostics();
  const mutators: ResourceMutator[] = [];

  for (const name of names) {
    const [obj, diag] = await loadObject(name);
    diagnostics = diagnostics.extend(diag);

    if (diag.hasError()) {
      continue;
    }

    if (!isResourceMutator(obj)) {
      const [, funcName] = name.split(":");
      diagnostics = diagnostics.extend(
        Diagnostics.createError(
          `'${funcName}' is not a ResourceMutator, did you decorate it with @jobMutator, @pipelineMutator, etc.?`
        )
      );
      continue;
    }

    mutators.push(obj as ResourceMutator);
  }

  return [mutators, diagnostics];
}

/**
 * Type guard for ResourceMutator
 */
function isResourceMutator(obj: unknown): obj is ResourceMutator {
  return (
    typeof obj === "object" &&
    obj !== null &&
    "resourceType" in obj &&
    "function" in obj &&
    typeof (obj as ResourceMutator).function === "function"
  );
}

/**
 * Load resources from functions
 */
export async function loadResources(
  bundle: Bundle,
  functions: ((bundle: Bundle) => Promise<Resources>)[]
): Promise<[Resources, Diagnostics]> {
  let diagnostics = new Diagnostics();
  const resources = new Resources();

  for (const func of functions) {
    try {
      const funcResources = await func(bundle);
      resources.addResources(funcResources);
      console.log("funcResources", JSON.stringify(funcResources, null, 2));
      console.log("resources", resources.toJSON());
    } catch (error) {
      console.error("Failed to load resources", error);
      const errorMessage = error instanceof Error ? error.message : String(error);
      diagnostics = diagnostics.extend(
        Diagnostics.createError("Failed to load resources", { detail: errorMessage })
      );
    }
  }
  return [resources, diagnostics];
}

/**
 * Append resources to bundle input
 */
export function appendResources(input: BundleInput, resources: Resources): BundleInput {
  const output = { ...input };
    
  const resourcesJSON = resources.toJSON();

  if (Object.keys(resourcesJSON).length > 0) {
    output.resources = output.resources || {};

    for (const [resourceType, resourceMap] of Object.entries(resourcesJSON)) {
      output.resources[resourceType] = {
        ...(output.resources[resourceType] || {}),
        ...resourceMap,
      };
    }
  }

  return output;
}

/**
 * Relativize file paths in locations
 */
export function relativizeLocations(
  locations: Map<string, Location>
): Map<string, Location> {
  const result = new Map<string, Location>();

  for (const [path, location] of locations) {
    result.set(path, relativizeLocation(location));
  }

  return result;
}

/**
 * Relativize a single location
 */
function relativizeLocation(location: Location): Location {
  return new Location({
    file: relativizePath(location.file),
    line: location.line,
    column: location.column,
  });
}

/**
 * Relativize a file path
 */
function relativizePath(path: string): string {
  if (!isAbsolute(path)) {
    return path;
  }

  const cwd = process.cwd();

  try {
    return relative(cwd, path);
  } catch {
    return path;
  }
}

/**
 * Write diagnostics to file
 */
export async function writeDiagnostics(path: string, diagnostics: Diagnostics): Promise<void> {
  const lines: string[] = [];

  for (const diagnostic of diagnostics.items) {
    const obj: Record<string, unknown> = {
      severity: diagnostic.severity,
      summary: diagnostic.summary,
    };

    if (diagnostic.detail) {
      obj.detail = diagnostic.detail;
    }

    if (diagnostic.location) {
      obj.location = {
        file: diagnostic.location.file,
        ...(diagnostic.location.line !== undefined && { line: diagnostic.location.line }),
        ...(diagnostic.location.column !== undefined && { column: diagnostic.location.column }),
      };
    }

    if (diagnostic.path) {
      obj.path = diagnostic.path.join(".");
    }

    lines.push(JSON.stringify(obj));
  }

  await writeFile(path, lines.join("\n"), "utf-8");
}

/**
 * Write locations to file
 */
export async function writeLocations(
  path: string,
  locations: Map<string, Location>
): Promise<void> {
  const lines: string[] = [];

  for (const [locPath, location] of locations) {
    const obj: LocationOutput = {
      path: locPath,
      file: location.file,
    };

    if (location.line !== undefined) {
      obj.line = location.line;
    }

    if (location.column !== undefined) {
      obj.column = location.column;
    }

    lines.push(JSON.stringify(obj));
  }

  await writeFile(path, lines.join("\n"), "utf-8");
}

/**
 * Main entry point for the build system
 */
export async function jsMutator(
  args: BuildArgs
): Promise<[BundleInput, Map<string, Location>, Diagnostics]> {
  
  const inputContent = await readFile(args.input, "utf-8");
  const input: BundleInput = JSON.parse(inputContent);

  let diagnostics = new Diagnostics();

  const [config, configDiag] = readConfig(input);
  diagnostics = diagnostics.extend(configDiag);

  if (diagnostics.hasError()) {
    return [input, new Map(), diagnostics];
  }

  const bundle = parseBundleInfo(input);
  if (args.phase === "load_resources") {    
    const [functions, funcDiag] = await loadFunctions(config.resources || []);
    diagnostics = diagnostics.extend(funcDiag);

    if (diagnostics.hasError()) {
      return [input, new Map(), diagnostics];
    }

    const [resources, resDiag] = await loadResources(bundle, functions);    
    diagnostics = diagnostics.extend(resDiag).extend(resources.diagnostics);

    if (diagnostics.hasError()) {
      return [input, new Map(), diagnostics];
    }

    const output = appendResources(input, resources);
    const locations = relativizeLocations(resources._locations);

    return [output, locations, diagnostics];
  } else {
    // apply_mutators phase
    return [input, new Map(), Diagnostics.createError("apply_mutators phase not yet implemented")];
  }
}

/**
 * Print help message
 */
function printHelp(): void {
  console.log(`
Databricks TypeScript Bundle Builder

Usage:
  databricks-ts-builder [OPTIONS]

Options:
  --phase <phase>           Phase to execute: load_resources or apply_mutators
  --input <path>            Path to input JSON file containing bundle configuration
  --output <path>           Path to output JSON file for processed bundle
  --diagnostics <path>      Path to diagnostics output file (newline-delimited JSON)
  --locations <path>        Path to locations output file (newline-delimited JSON, optional)
  --help, -h                Show this help message

Phases:
  load_resources            Load resources from TypeScript modules and append to bundle
  apply_mutators            Apply resource mutators to transform existing resources

Examples:
  # Load resources phase
  databricks-ts-builder --phase load_resources \\
    --input bundle.json \\
    --output bundle-with-resources.json \\
    --diagnostics diagnostics.txt

  # With locations tracking
  databricks-ts-builder --phase load_resources \\
    --input bundle.json \\
    --output bundle-with-resources.json \\
    --diagnostics diagnostics.txt \\
    --locations locations.txt

For more information, visit: https://docs.databricks.com/dev-tools/bundles/
`);
}

/**
 * CLI entry point
 */
export async function main(argv: string[]): Promise<number> {
  try {
    const args = parseArgs(argv.slice(2));

    if (args === "help") {
      printHelp();
      return 0;
    }

    const [output, locations, diagnostics] = await jsMutator(args);

    // Write diagnostics
    await writeDiagnostics(args.diagnostics, diagnostics);

    // Write locations if specified
    if (args.locations) {
      await writeLocations(args.locations, locations);
    }

    // Write output
    await writeFile(args.output, JSON.stringify(output), "utf-8");

    return diagnostics.hasError() ? 1 : 0;
  } catch (error) {
    console.error("Fatal error:", error);
    return 1;
  }
}
