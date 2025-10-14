import {
  parseArgs,
  readConfig,
  parseBundleInfo,
  appendResources,
  type BundleInput,
} from "./index.js";
import { Resources } from "../core/resources.js";
import { Resource } from "../core/resource.js";

// Mock resource for testing
class MockResource extends Resource<{ name: string }> {
  constructor(name: string, params: { name: string }) {
    super(name, params, "apps");
  }
}

describe("build/index", () => {
  describe("parseArgs", () => {
    it("should parse valid arguments", () => {
      const args = [
        "--phase",
        "load_resources",
        "--input",
        "input.json",
        "--output",
        "output.json",
        "--diagnostics",
        "diag.txt",
      ];

      const result = parseArgs(args);
      expect(result).toEqual({
        phase: "load_resources",
        input: "input.json",
        output: "output.json",
        diagnostics: "diag.txt",
      });
    });

    it("should parse with locations argument", () => {
      const args = [
        "--phase",
        "apply_mutators",
        "--input",
        "input.json",
        "--output",
        "output.json",
        "--diagnostics",
        "diag.txt",
        "--locations",
        "loc.txt",
      ];

      const result = parseArgs(args);
      expect(result).toEqual({
        phase: "apply_mutators",
        input: "input.json",
        output: "output.json",
        diagnostics: "diag.txt",
        locations: "loc.txt",
      });
    });

    it("should return help for --help flag", () => {
      expect(parseArgs(["--help"])).toBe("help");
    });

    it("should return help for -h flag", () => {
      expect(parseArgs(["-h"])).toBe("help");
    });

    it("should throw on invalid phase", () => {
      const args = [
        "--phase",
        "invalid_phase",
        "--input",
        "input.json",
        "--output",
        "output.json",
        "--diagnostics",
        "diag.txt",
      ];

      expect(() => parseArgs(args)).toThrow("Invalid phase");
    });

    it("should throw on missing phase", () => {
      const args = [
        "--input",
        "input.json",
        "--output",
        "output.json",
        "--diagnostics",
        "diag.txt",
      ];

      expect(() => parseArgs(args)).toThrow("Missing required argument --phase");
    });

    it("should throw on missing input", () => {
      const args = [
        "--phase",
        "load_resources",
        "--output",
        "output.json",
        "--diagnostics",
        "diag.txt",
      ];

      expect(() => parseArgs(args)).toThrow("Missing required argument --input");
    });

    it("should throw on missing output", () => {
      const args = [
        "--phase",
        "load_resources",
        "--input",
        "input.json",
        "--diagnostics",
        "diag.txt",
      ];

      expect(() => parseArgs(args)).toThrow("Missing required argument --output");
    });

    it("should throw on missing diagnostics", () => {
      const args = [
        "--phase",
        "load_resources",
        "--input",
        "input.json",
        "--output",
        "output.json",
      ];

      expect(() => parseArgs(args)).toThrow("Missing required argument --diagnostics");
    });
  });

  describe("readConfig", () => {
    it("should read from javascript section", () => {
      const input: BundleInput = {
        javascript: {
          resources: ["resources:loadResources"],
        },
      };

      const [config, diag] = readConfig(input);
      expect(config).toEqual({
        resources: ["resources:loadResources"],
      });
      expect(diag.hasError()).toBe(false);
    });

    it("should read from experimental/javascript section", () => {
      const input: BundleInput = {
        experimental: {
          javascript: {
            resources: ["resources:loadResources"],
          },
        },
      };

      const [config, diag] = readConfig(input);
      expect(config).toEqual({
        resources: ["resources:loadResources"],
      });
      expect(diag.hasError()).toBe(false);
    });

    it("should prefer javascript over experimental/javascript when they match", () => {
      const input: BundleInput = {
        javascript: {
          resources: ["resources:loadResources"],
        },
        experimental: {
          javascript: {
            resources: ["resources:loadResources"],
          },
        },
      };

      const [config, diag] = readConfig(input);
      expect(config).toEqual({
        resources: ["resources:loadResources"],
      });
      expect(diag.hasError()).toBe(false);
    });

    it("should error when both sections differ", () => {
      const input: BundleInput = {
        javascript: {
          resources: ["resources:loadResources"],
        },
        experimental: {
          javascript: {
            resources: ["other:loadResources"],
          },
        },
      };

      const [, diag] = readConfig(input);
      expect(diag.hasError()).toBe(true);
      expect(diag.getErrors()[0]?.summary).toContain("'javascript' and 'experimental/javascript'");
    });

    it("should return empty config when no javascript section", () => {
      const input: BundleInput = {};

      const [config, diag] = readConfig(input);
      expect(config).toEqual({});
      expect(diag.hasError()).toBe(false);
    });
  });

  describe("parseBundleInfo", () => {
    it("should parse bundle with target and variables", () => {
      const input: BundleInput = {
        bundle: {
          target: "development",
          mode: "development",
          name: "test-bundle",
        },
        variables: {
          warehouse_id: { value: "abc123" },
          job_name: { value: "my-job" },
        },
      };

      const bundle = parseBundleInfo(input);
      expect(bundle.target).toBe("development");
      expect(bundle.mode).toBe("development");
      expect(bundle.name).toBe("test-bundle");
      expect(bundle.variables).toEqual({
        warehouse_id: "abc123",
        job_name: "my-job",
      });
    });

    it("should use defaults when bundle section missing", () => {
      const input: BundleInput = {};

      const bundle = parseBundleInfo(input);
      expect(bundle.target).toBe("default");
      expect(bundle.name).toBe("unknown");
      expect(bundle.variables).toEqual({});
    });

    it("should handle variables without value property", () => {
      const input: BundleInput = {
        bundle: {
          name: "test",
        },
        variables: {
          // eslint-disable-next-line @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-explicit-any
          foo: "bar" as any, // Not the expected format
        },
      };

      const bundle = parseBundleInfo(input);
      expect(bundle.variables).toEqual({});
    });
  });

  describe("appendResources", () => {
    it("should append resources to empty input", () => {
      const input: BundleInput = {};
      const resources = new Resources();
      resources.addResource(new MockResource("app1", { name: "app1" }));

      const result = appendResources(input, resources);

      expect(result.resources).toEqual({
        apps: {
          app1: { name: "app1" },
        },
      });
    });

    it("should merge with existing resources", () => {
      const input: BundleInput = {
        resources: {
          jobs: {
            job1: { name: "job1" },
          },
        },
      };

      const resources = new Resources();
      resources.addResource(new MockResource("app1", { name: "app1" }));

      const result = appendResources(input, resources);

      expect(result.resources).toEqual({
        jobs: {
          job1: { name: "job1" },
        },
        apps: {
          app1: { name: "app1" },
        },
      });
    });

    it("should merge resources of the same type", () => {
      const input: BundleInput = {
        resources: {
          apps: {
            app1: { name: "app1" },
          },
        },
      };

      const resources = new Resources();
      resources.addResource(new MockResource("app2", { name: "app2" }));

      const result = appendResources(input, resources);

      expect(result.resources).toEqual({
        apps: {
          app1: { name: "app1" },
          app2: { name: "app2" },
        },
      });
    });

    it("should not modify input when resources empty", () => {
      const input: BundleInput = {
        resources: {
          jobs: {
            job1: { name: "job1" },
          },
        },
      };

      const resources = new Resources();
      const result = appendResources(input, resources);

      expect(result.resources).toEqual({
        jobs: {
          job1: { name: "job1" },
        },
      });
    });
  });
});
