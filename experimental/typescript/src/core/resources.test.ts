import { describe, it, expect } from "@jest/globals";
import { Resources } from "./resources.js";
import { Resource } from "./resource.js";
import { Location } from "./location.js";

// Mock resource for testing
interface MockParams {
  name: string;
  value?: number;
}

class MockResource extends Resource<MockParams> {
  constructor(name: string, params: MockParams, type: "apps" | "jobs" = "apps") {
    super(name, params, type);
  }
}

describe("Resources", () => {
  describe("constructor", () => {
    it("should create an empty resources container", async () => {
      const resources = new Resources();
      expect(await resources.toDabsResources()).toEqual({});
    });

    it("should initialize diagnostics", () => {
      const resources = new Resources();
      expect(resources.diagnostics.items).toEqual([]);
    });
  });

  describe("addResource", () => {
    it("should add a resource", async () => {
      const resources = new Resources();
      const resource = new MockResource("test_resource", { name: "test" });

      resources.addResource(resource);

      const json = await resources.toDabsResources();
      expect(json.apps).toEqual({
        test_resource: { name: "test" },
      });
    });

    it("should add multiple resources of the same type", async () => {
      const resources = new Resources();
      const resource1 = new MockResource("resource1", { name: "test1" });
      const resource2 = new MockResource("resource2", { name: "test2" });

      resources.addResource(resource1);
      resources.addResource(resource2);

      const json = await resources.toDabsResources();
      expect(json.apps).toEqual({
        resource1: { name: "test1" },
        resource2: { name: "test2" },
      });
    });

    it("should add resources of different types", async () => {
      const resources = new Resources();
      const appResource = new MockResource("app1", { name: "app" }, "apps");
      const jobResource = new MockResource("job1", { name: "job" }, "jobs");

      resources.addResource(appResource);
      resources.addResource(jobResource);

      const json = await resources.toDabsResources();
      expect(json.apps).toEqual({ app1: { name: "app" } });
      expect(json.jobs).toEqual({ job1: { name: "job" } });
    });

    it("should warn on duplicate resource names", () => {
      const resources = new Resources();
      const resource1 = new MockResource("duplicate", { name: "first" });
      const resource2 = new MockResource("duplicate", { name: "second" });

      resources.addResource(resource1);
      resources.addResource(resource2);

      expect(resources.diagnostics.hasWarning()).toBe(true);
      expect(resources.diagnostics.getWarnings()[0]?.summary).toContain("Duplicate");
    });

    it("should store location if provided", () => {
      const resources = new Resources();
      const resource = new MockResource("test", { name: "test" });
      const location = new Location({ file: "test.ts", line: 10 });

      resources.addResource(resource, location);

      expect(resources._locations.get("resources.apps.test")).toBe(location);
    });
  });

  describe("getResources", () => {
    it("should return resources of a specific type", () => {
      const resources = new Resources();
      const resource1 = new MockResource("app1", { name: "app1" });
      const resource2 = new MockResource("app2", { name: "app2" });

      resources.addResource(resource1);
      resources.addResource(resource2);

      const apps = resources.getResources("apps");
      expect(apps.size).toBe(2);
      expect(apps.get("app1")).toBe(resource1);
      expect(apps.get("app2")).toBe(resource2);
    });

    it("should return empty map for types with no resources", () => {
      const resources = new Resources();
      const jobs = resources.getResources("jobs");
      expect(jobs.size).toBe(0);
    });
  });

  describe("addResources", () => {
    it("should merge resources from another Resources instance", async () => {
      const resources1 = new Resources();
      const resources2 = new Resources();

      resources1.addResource(new MockResource("app1", { name: "app1" }));
      resources2.addResource(new MockResource("app2", { name: "app2" }));

      resources1.addResources(resources2);

      const json = await resources1.toDabsResources();
      expect(json.apps).toEqual({
        app1: { name: "app1" },
        app2: { name: "app2" },
      });
    });

    it("should merge diagnostics", () => {
      const resources1 = new Resources();
      const resources2 = new Resources();

      resources2.addDiagnosticWarning("Test warning");

      resources1.addResources(resources2);

      expect(resources1.diagnostics.hasWarning()).toBe(true);
    });
  });

  describe("diagnostics", () => {
    it("should add diagnostic errors", () => {
      const resources = new Resources();
      resources.addDiagnosticError("Test error", {
        detail: "Error details",
        path: ["resources", "apps", "test"],
      });

      expect(resources.diagnostics.hasError()).toBe(true);
      expect(resources.diagnostics.getErrors()[0]?.summary).toBe("Test error");
    });

    it("should add diagnostic warnings", () => {
      const resources = new Resources();
      resources.addDiagnosticWarning("Test warning", {
        detail: "Warning details",
      });

      expect(resources.diagnostics.hasWarning()).toBe(true);
      expect(resources.diagnostics.getWarnings()[0]?.summary).toBe("Test warning");
    });
  });

  describe("addLocation", () => {
    it("should store location for a path", () => {
      const resources = new Resources();
      const location = new Location({ file: "test.ts", line: 10 });

      resources.addLocation(["resources", "apps", "test"], location);

      expect(resources._locations.get("resources.apps.test")).toBe(location);
    });
  });

  describe("toJSON", () => {
    it("should return empty object when no resources", () => {
      const resources = new Resources();
      expect(resources.toDabsResources()).toEqual({});
    });

    it("should serialize all resources", async () => {
      const resources = new Resources();
      resources.addResource(new MockResource("app1", { name: "app1", value: 1 }));
      resources.addResource(new MockResource("job1", { name: "job1" }, "jobs"));

      const json = await resources.toDabsResources();
      expect(json).toEqual({
        apps: { app1: { name: "app1", value: 1 } },
        jobs: { job1: { name: "job1" } },
      });
    });

    it("should not include resource types with no resources", async () => {
      const resources = new Resources();
      resources.addResource(new MockResource("app1", { name: "app1" }));

      const json = await resources.toDabsResources();
      expect(json.apps).toBeDefined();
      expect(json.jobs).toBeUndefined();
    });
  });
});
