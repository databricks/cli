import { Diagnostics, Severity } from "./diagnostics.js";
import { Location } from "./location.js";

describe("Diagnostics", () => {
  describe("constructor", () => {
    it("should create empty diagnostics", () => {
      const diag = new Diagnostics();
      expect(diag.items).toEqual([]);
    });

    it("should create diagnostics with items", () => {
      const diag = new Diagnostics([
        {
          severity: Severity.ERROR,
          summary: "Test error",
        },
      ]);

      expect(diag.items).toHaveLength(1);
      expect(diag.items[0]?.severity).toBe(Severity.ERROR);
      expect(diag.items[0]?.summary).toBe("Test error");
    });
  });

  describe("extend()", () => {
    it("should combine diagnostics", () => {
      const diag1 = new Diagnostics([{ severity: Severity.ERROR, summary: "Error 1" }]);
      const diag2 = new Diagnostics([{ severity: Severity.WARNING, summary: "Warning 1" }]);

      const combined = diag1.extend(diag2);

      expect(combined.items).toHaveLength(2);
      expect(combined.items[0]?.summary).toBe("Error 1");
      expect(combined.items[1]?.summary).toBe("Warning 1");
    });
  });

  describe("extendTuple()", () => {
    it("should extract value and extend diagnostics", () => {
      const diag1 = new Diagnostics();
      const diag2 = new Diagnostics([{ severity: Severity.ERROR, summary: "Error" }]);

      const [value, combined] = diag1.extendTuple([42, diag2]);

      expect(value).toBe(42);
      expect(combined.items).toHaveLength(1);
    });
  });

  describe("hasError()", () => {
    it("should return true if diagnostics contains errors", () => {
      const diag = new Diagnostics([{ severity: Severity.ERROR, summary: "Error" }]);

      expect(diag.hasError()).toBe(true);
    });

    it("should return false if no errors", () => {
      const diag = new Diagnostics([{ severity: Severity.WARNING, summary: "Warning" }]);

      expect(diag.hasError()).toBe(false);
    });
  });

  describe("hasWarning()", () => {
    it("should return true if diagnostics contains warnings", () => {
      const diag = new Diagnostics([{ severity: Severity.WARNING, summary: "Warning" }]);

      expect(diag.hasWarning()).toBe(true);
    });

    it("should return false if no warnings", () => {
      const diag = new Diagnostics([{ severity: Severity.ERROR, summary: "Error" }]);

      expect(diag.hasWarning()).toBe(false);
    });
  });

  describe("getErrors()", () => {
    it("should return only errors", () => {
      const diag = new Diagnostics([
        { severity: Severity.ERROR, summary: "Error 1" },
        { severity: Severity.WARNING, summary: "Warning" },
        { severity: Severity.ERROR, summary: "Error 2" },
      ]);

      const errors = diag.getErrors();
      expect(errors).toHaveLength(2);
      expect(errors[0]?.summary).toBe("Error 1");
      expect(errors[1]?.summary).toBe("Error 2");
    });
  });

  describe("getWarnings()", () => {
    it("should return only warnings", () => {
      const diag = new Diagnostics([
        { severity: Severity.ERROR, summary: "Error" },
        { severity: Severity.WARNING, summary: "Warning 1" },
        { severity: Severity.WARNING, summary: "Warning 2" },
      ]);

      const warnings = diag.getWarnings();
      expect(warnings).toHaveLength(2);
      expect(warnings[0]?.summary).toBe("Warning 1");
      expect(warnings[1]?.summary).toBe("Warning 2");
    });
  });

  describe("createError()", () => {
    it("should create an error diagnostic", () => {
      const diag = Diagnostics.createError("Test error");

      expect(diag.items).toHaveLength(1);
      expect(diag.items[0]?.severity).toBe(Severity.ERROR);
      expect(diag.items[0]?.summary).toBe("Test error");
    });

    it("should include optional fields", () => {
      const location = new Location({ file: "test.ts", line: 10 });
      const diag = Diagnostics.createError("Test error", {
        detail: "Detailed message",
        location,
        path: ["resources", "jobs", "my_job"],
      });

      expect(diag.items[0]?.detail).toBe("Detailed message");
      expect(diag.items[0]?.location).toBe(location);
      expect(diag.items[0]?.path).toEqual(["resources", "jobs", "my_job"]);
    });
  });

  describe("createWarning()", () => {
    it("should create a warning diagnostic", () => {
      const diag = Diagnostics.createWarning("Test warning");

      expect(diag.items).toHaveLength(1);
      expect(diag.items[0]?.severity).toBe(Severity.WARNING);
      expect(diag.items[0]?.summary).toBe("Test warning");
    });
  });

  describe("fromException()", () => {
    it("should create diagnostics from an error", () => {
      const error = new Error("Something went wrong");
      const diag = Diagnostics.fromException(error, "Failed to process");

      expect(diag.items).toHaveLength(1);
      expect(diag.items[0]?.severity).toBe(Severity.ERROR);
      expect(diag.items[0]?.summary).toBe("Failed to process");
      expect(diag.items[0]?.detail).toContain("Something went wrong");
    });

    it("should include explanation", () => {
      const error = new Error("Test error");
      const diag = Diagnostics.fromException(error, "Failed", {
        explanation: "This is why it failed",
      });

      expect(diag.items[0]?.detail).toContain("This is why it failed");
    });
  });

  describe("toJSON()", () => {
    it("should serialize diagnostics to JSON", () => {
      const location = new Location({ file: "test.ts", line: 1 });
      const diag = new Diagnostics([
        {
          severity: Severity.ERROR,
          summary: "Error",
          detail: "Details",
          path: ["resources", "jobs"],
          location,
        },
      ]);

      const json = diag.toJSON();
      expect(json).toHaveLength(1);
      expect(json[0]).toMatchObject({
        severity: "error",
        summary: "Error",
        detail: "Details",
        path: ["resources", "jobs"],
      });
    });
  });
});
