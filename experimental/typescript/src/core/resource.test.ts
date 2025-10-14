import { describe, it, expect } from "@jest/globals";
import { Resource } from "./resource.js";

interface TestParams {
  name: string;
  value: number;
  nested?: {
    foo: string;
  };
}

class TestResource extends Resource<TestParams> {
  constructor(name: string, params: TestParams) {
    super(name, params, "apps");
  }
}

describe("Resource", () => {
  describe("constructor", () => {
    it("should create a resource with name, data, and type", () => {
      const params: TestParams = { name: "test", value: 42 };
      const resource = new TestResource("my_resource", params);

      expect(resource.dabsName).toBe("my_resource");
      expect(resource.data).toEqual(params);
      expect(resource.type).toBe("apps");
    });

    it("should store data as readonly", () => {
      const params: TestParams = { name: "test", value: 42 };
      const resource = new TestResource("my_resource", params);

      expect(resource.data).toBe(params);
    });
  });

  describe("toJSON()", () => {
    it("should serialize resource data", () => {
      const params: TestParams = { name: "test", value: 42 };
      const resource = new TestResource("my_resource", params);

      const json = resource.toJSON();
      expect(json).toEqual({ name: "test", value: 42 });
    });

    it("should handle nested objects", () => {
      const params: TestParams = {
        name: "test",
        value: 42,
        nested: { foo: "bar" },
      };
      const resource = new TestResource("my_resource", params);

      const json = resource.toJSON();
      expect(json).toEqual({
        name: "test",
        nested: { foo: "bar" },
        value: 42,
      });
    });

    it("should omit undefined values", () => {
      const params: TestParams = {
        name: "test",
        value: 42,
        nested: undefined,
      };
      const resource = new TestResource("my_resource", params);

      const json = resource.toJSON();
      expect(json).toEqual({ name: "test", value: 42 });
    });
  });
});
