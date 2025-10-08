import { Bundle } from "./bundle.js";
import { Variable } from "./variable.js";

describe("Bundle", () => {
  describe("constructor", () => {
    it("should create a bundle with target and variables", () => {
      const bundle = new Bundle({
        target: "development",
        variables: {
          warehouse_id: "abc123",
          job_name: "my-job",
        },
      });

      expect(bundle.target).toBe("development");
      expect(bundle.variables).toEqual({
        warehouse_id: "abc123",
        job_name: "my-job",
      });
    });

    it("should freeze variables object", () => {
      const bundle = new Bundle({
        target: "production",
        variables: { foo: "bar" },
      });

      expect(Object.isFrozen(bundle.variables)).toBe(true);
    });
  });

  describe("resolveVariable()", () => {
    it("should return concrete values as-is", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {},
      });

      expect(bundle.resolveVariable("concrete")).toBe("concrete");
      expect(bundle.resolveVariable(123)).toBe(123);
      expect(bundle.resolveVariable(true)).toBe(true);
    });

    it("should resolve variable references", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {
          warehouse_id: "abc123",
          job_name: "my-job",
        },
      });

      const warehouseVar = new Variable<string>("var.warehouse_id");
      expect(bundle.resolveVariable(warehouseVar)).toBe("abc123");

      const jobNameVar = new Variable<string>("var.job_name");
      expect(bundle.resolveVariable(jobNameVar)).toBe("my-job");
    });

    it("should throw error for non-var prefix", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {},
      });

      const invalidVar = new Variable<string>("bundle.something");
      expect(() => bundle.resolveVariable(invalidVar)).toThrow(
        "You can only get values of variables starting with 'var.*'"
      );
    });

    it("should throw error for undefined variable", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {},
      });

      const missingVar = new Variable<string>("var.missing");
      expect(() => bundle.resolveVariable(missingVar)).toThrow(
        "Can't find 'missing' variable. Did you define it in databricks.yml?"
      );
    });

    it("should throw error for nested variable references", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {
          foo: "${var.bar}",
        },
      });

      const fooVar = new Variable<string>("var.foo");
      expect(() => bundle.resolveVariable(fooVar)).toThrow(
        /refers to another variable/
      );
    });
  });

  describe("resolveVariableList()", () => {
    it("should resolve a list variable", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {
          my_list: ["a", "b", "c"],
        },
      });

      const listVar = new Variable<string[]>("var.my_list");
      expect(bundle.resolveVariableList(listVar)).toEqual(["a", "b", "c"]);
    });

    it("should resolve variables within a list", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {
          item1: "value1",
          my_list: ["a", "b"],
        },
      });

      const result = bundle.resolveVariableList(["a", "b"]);
      expect(result).toEqual(["a", "b"]);
    });

    it("should throw error if value is not a list", () => {
      const bundle = new Bundle({
        target: "dev",
        variables: {
          not_a_list: "string",
        },
      });

      const notListVar = new Variable<string[]>("var.not_a_list");
      expect(() => bundle.resolveVariableList(notListVar)).toThrow("Expected a list value");
    });
  });
});
