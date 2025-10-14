import { describe, it, expect } from "@jest/globals";
import { Variable, variable, variables, isVariable, getVariablePath } from "./variable.js";

describe("Variable", () => {
  describe("constructor", () => {
    it("should create a variable with a path", () => {
      const v = new Variable<string>("var.my_var");
      expect(v.path).toBe("var.my_var");
    });
  });

  describe("value property", () => {
    it("should return the variable reference in ${} format", () => {
      const v = new Variable<string>("var.warehouse_id");
      expect(v.value).toBe("${var.warehouse_id}");
    });
  });

  describe("toString()", () => {
    it("should return the variable reference", () => {
      const v = new Variable<string>("var.job_name");
      expect(v.toString()).toBe("${var.job_name}");
    });
  });

  describe("toJSON()", () => {
    it("should serialize to variable reference", () => {
      const v = new Variable<string>("var.cluster_id");
      expect(v.toJSON()).toBe("${var.cluster_id}");
      expect(JSON.stringify(v)).toBe('"${var.cluster_id}"');
    });
  });
});

describe("variable()", () => {
  it("should create a Variable instance", () => {
    const v = variable<string>("var.my_var");
    expect(v).toBeInstanceOf(Variable);
    expect(v.path).toBe("var.my_var");
  });
});

describe("variables()", () => {
  it("should create a proxy that returns Variables", () => {
    interface MyVars extends Record<string, Variable<unknown>> {
      warehouse_id: Variable<string>;
      job_name: Variable<string>;
    }

    const vars = variables<MyVars>();

    expect(vars.warehouse_id).toBeInstanceOf(Variable);
    expect(vars.warehouse_id.path).toBe("var.warehouse_id");
    expect(vars.warehouse_id.value).toBe("${var.warehouse_id}");

    expect(vars.job_name).toBeInstanceOf(Variable);
    expect(vars.job_name.path).toBe("var.job_name");
  });

  it("should support custom prefix", () => {
    interface MyVars extends Record<string, Variable<unknown>> {
      foo: Variable<string>;
    }

    const vars = variables<MyVars>("custom");
    expect(vars.foo.path).toBe("custom.foo");
  });
});

describe("isVariable()", () => {
  it("should return true for Variable instances", () => {
    const v = new Variable<string>("var.test");
    expect(isVariable(v)).toBe(true);
  });

  it("should return false for non-Variable values", () => {
    expect(isVariable("string")).toBe(false);
    expect(isVariable(123)).toBe(false);
    expect(isVariable(null)).toBe(false);
    expect(isVariable(undefined)).toBe(false);
    expect(isVariable({})).toBe(false);
  });
});

describe("getVariablePath()", () => {
  it("should return path for Variable instances", () => {
    const v = new Variable<string>("var.test");
    expect(getVariablePath(v)).toBe("var.test");
  });

  it("should return undefined for non-Variable values", () => {
    expect(getVariablePath("string")).toBeUndefined();
    expect(getVariablePath(123)).toBeUndefined();
  });
});
