import { describe, it, expect } from "@jest/globals";
import { Location } from "./location.js";

describe("Location", () => {
  describe("constructor", () => {
    it("should create a location with file only", () => {
      const loc = new Location({ file: "test.ts" });
      expect(loc.file).toBe("test.ts");
      expect(loc.line).toBeUndefined();
      expect(loc.column).toBeUndefined();
    });

    it("should create a location with file and line", () => {
      const loc = new Location({ file: "test.ts", line: 10 });
      expect(loc.file).toBe("test.ts");
      expect(loc.line).toBe(10);
      expect(loc.column).toBeUndefined();
    });

    it("should create a location with file, line, and column", () => {
      const loc = new Location({ file: "test.ts", line: 10, column: 5 });
      expect(loc.file).toBe("test.ts");
      expect(loc.line).toBe(10);
      expect(loc.column).toBe(5);
    });

    it("should throw error for line number less than 1", () => {
      expect(() => {
        new Location({ file: "test.ts", line: 0 });
      }).toThrow("Line number must be greater than 0");

      expect(() => {
        new Location({ file: "test.ts", line: -1 });
      }).toThrow("Line number must be greater than 0");
    });

    it("should throw error for column number less than 1", () => {
      expect(() => {
        new Location({ file: "test.ts", line: 1, column: 0 });
      }).toThrow("Column number must be greater than 0");

      expect(() => {
        new Location({ file: "test.ts", line: 1, column: -1 });
      }).toThrow("Column number must be greater than 0");
    });
  });

  describe("fromFunction()", () => {
    it("should return undefined", () => {
      const loc = Location.fromFunction(() => {});
      expect(loc).toBeUndefined();
    });
  });

  describe("fromStack()", () => {
    it("should capture location from stack trace", () => {
      const loc = Location.fromStack(0);

      // The location should exist and have a file
      expect(loc).toBeDefined();
      if (loc) {
        expect(loc.file).toBeTruthy();
        expect(loc.line).toBeGreaterThan(0);
        expect(loc.column).toBeGreaterThan(0);
      }
    });

    it("should handle different stack depths", () => {
      function nested() {
        return Location.fromStack(1);
      }

      const loc = nested();
      expect(loc).toBeDefined();
      if (loc) {
        expect(loc.file).toBeTruthy();
      }
    });
  });

  describe("toJSON()", () => {
    it("should serialize location with file only", () => {
      const loc = new Location({ file: "test.ts" });
      expect(loc.toJSON()).toEqual({ file: "test.ts" });
    });

    it("should serialize location with file and line", () => {
      const loc = new Location({ file: "test.ts", line: 10 });
      expect(loc.toJSON()).toEqual({ file: "test.ts", line: 10 });
    });

    it("should serialize location with file, line, and column", () => {
      const loc = new Location({ file: "test.ts", line: 10, column: 5 });
      expect(loc.toJSON()).toEqual({ file: "test.ts", line: 10, column: 5 });
    });
  });

  describe("toString()", () => {
    it("should format location with file only", () => {
      const loc = new Location({ file: "test.ts" });
      expect(loc.toString()).toBe("test.ts");
    });

    it("should format location with file and line", () => {
      const loc = new Location({ file: "test.ts", line: 10 });
      expect(loc.toString()).toBe("test.ts:10");
    });

    it("should format location with file, line, and column", () => {
      const loc = new Location({ file: "test.ts", line: 10, column: 5 });
      expect(loc.toString()).toBe("test.ts:10:5");
    });
  });
});
