import { describe, it, expect } from "@jest/globals";
import { transformToJSON } from "./transform.js";
import { Variable } from "./variable.js";

describe("transformToJSON", () => {
  describe("primitives", () => {
    it("should handle strings", () => {
      expect(transformToJSON("hello")).toBe("hello");
      expect(transformToJSON("")).toBe("");
    });

    it("should handle numbers", () => {
      expect(transformToJSON(42)).toBe(42);
      expect(transformToJSON(0)).toBe(0);
      expect(transformToJSON(-1)).toBe(-1);
      expect(transformToJSON(3.14)).toBe(3.14);
    });

    it("should handle booleans", () => {
      expect(transformToJSON(true)).toBe(true);
      expect(transformToJSON(false)).toBe(false);
    });

    it("should handle null and undefined", () => {
      expect(transformToJSON(null)).toBe(null);
      expect(transformToJSON(undefined)).toBe(null);
    });
  });

  describe("Variable instances", () => {
    it("should transform Variable to its value", () => {
      const v = new Variable<string>("var.test");
      expect(transformToJSON(v)).toBe("${var.test}");
    });

    it("should handle Variables in objects", () => {
      const obj = {
        name: "test",
        id: new Variable<string>("var.warehouse_id"),
      };

      const result = transformToJSON(obj);
      expect(result).toEqual({
        id: "${var.warehouse_id}",
        name: "test",
      });
    });

    it("should handle Variables in arrays", () => {
      const arr = ["test", new Variable<string>("var.value")];

      const result = transformToJSON(arr);
      expect(result).toEqual(["test", "${var.value}"]);
    });
  });

  describe("arrays", () => {
    it("should transform arrays", () => {
      const arr = [1, 2, 3];
      expect(transformToJSON(arr)).toEqual([1, 2, 3]);
    });

    it("should transform nested arrays", () => {
      const arr = [
        [1, 2],
        [3, 4],
      ];
      expect(transformToJSON(arr)).toEqual([
        [1, 2],
        [3, 4],
      ]);
    });

    it("should transform arrays with mixed types", () => {
      const arr = [1, "test", true, null];
      expect(transformToJSON(arr)).toEqual([1, "test", true, null]);
    });
  });

  describe("objects", () => {
    it("should transform plain objects", () => {
      const obj = { name: "test", value: 42 };
      expect(transformToJSON(obj)).toEqual({ name: "test", value: 42 });
    });

    it("should transform nested objects", () => {
      const obj = {
        name: "test",
        nested: {
          foo: "bar",
          baz: 123,
        },
      };

      expect(transformToJSON(obj)).toEqual({
        name: "test",
        nested: { baz: 123, foo: "bar" },
      });
    });

    it("should sort object keys", () => {
      const obj = { z: 1, a: 2, m: 3 };
      const result = transformToJSON(obj) as Record<string, unknown>;

      expect(Object.keys(result)).toEqual(["a", "m", "z"]);
    });
  });

  describe("omitempty semantics", () => {
    it("should omit null values", () => {
      const obj = { name: "test", value: null };
      expect(transformToJSON(obj)).toEqual({ name: "test" });
    });

    it("should omit undefined values", () => {
      const obj = { name: "test", value: undefined };
      expect(transformToJSON(obj)).toEqual({ name: "test" });
    });

    it("should omit empty arrays", () => {
      const obj = { name: "test", items: [] };
      expect(transformToJSON(obj)).toEqual({ name: "test" });
    });

    it("should omit empty objects", () => {
      const obj = { name: "test", config: {} };
      expect(transformToJSON(obj)).toEqual({ name: "test" });
    });

    it("should include non-empty arrays", () => {
      const obj = { name: "test", items: [1, 2, 3] };
      expect(transformToJSON(obj)).toEqual({ name: "test", items: [1, 2, 3] });
    });

    it("should include non-empty objects", () => {
      const obj = { name: "test", config: { foo: "bar" } };
      expect(transformToJSON(obj)).toEqual({ name: "test", config: { foo: "bar" } });
    });

    it("should include zero and false", () => {
      const obj = { name: "test", count: 0, enabled: false };
      expect(transformToJSON(obj)).toEqual({ count: 0, enabled: false, name: "test" });
    });

    it("should include empty strings", () => {
      const obj = { name: "", value: "test" };
      expect(transformToJSON(obj)).toEqual({ name: "", value: "test" });
    });
  });

  describe("complex scenarios", () => {
    it("should handle deeply nested structures", () => {
      const obj = {
        name: "test",
        config: {
          nested: {
            deep: {
              value: 42,
            },
          },
        },
      };

      expect(transformToJSON(obj)).toEqual({
        config: { nested: { deep: { value: 42 } } },
        name: "test",
      });
    });

    it("should handle mixed arrays and objects", () => {
      const obj = {
        items: [
          { name: "item1", value: 1 },
          { name: "item2", value: 2 },
        ],
      };

      expect(transformToJSON(obj)).toEqual({
        items: [
          { name: "item1", value: 1 },
          { name: "item2", value: 2 },
        ],
      });
    });

    it("should handle Variables mixed with regular values", () => {
      const obj = {
        name: "test",
        warehouse_id: new Variable<string>("var.warehouse"),
        tasks: [
          {
            task_key: "task1",
            cluster_id: new Variable<string>("var.cluster"),
          },
        ],
      };

      const result = transformToJSON(obj);
      expect(result).toEqual({
        name: "test",
        tasks: [{ cluster_id: "${var.cluster}", task_key: "task1" }],
        warehouse_id: "${var.warehouse}",
      });
    });
  });
});
