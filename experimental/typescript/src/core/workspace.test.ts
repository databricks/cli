import { describe, it, expect } from "@jest/globals";
import { Workspace } from "./workspace.js";
import { Variable } from "./variable.js";

describe("Workspace", () => {
  describe("host", () => {
    it("should be a Variable with correct path", () => {
      expect(Workspace.host).toBeInstanceOf(Variable);
      expect(Workspace.host.path).toBe("workspace.host");
      expect(Workspace.host.value).toBe("${workspace.host}");
    });
  });

  describe("currentUser", () => {
    it("should have domainFriendlyName Variable", () => {
      expect(Workspace.currentUser.domainFriendlyName).toBeInstanceOf(Variable);
      expect(Workspace.currentUser.domainFriendlyName.path).toBe(
        "workspace.current_user.domain_friendly_name"
      );
      expect(Workspace.currentUser.domainFriendlyName.value).toBe(
        "${workspace.current_user.domain_friendly_name}"
      );
    });

    it("should have userName Variable", () => {
      expect(Workspace.currentUser.userName).toBeInstanceOf(Variable);
      expect(Workspace.currentUser.userName.path).toBe("workspace.current_user.user_name");
      expect(Workspace.currentUser.userName.value).toBe("${workspace.current_user.user_name}");
    });

    it("should have shortName Variable", () => {
      expect(Workspace.currentUser.shortName).toBeInstanceOf(Variable);
      expect(Workspace.currentUser.shortName.path).toBe("workspace.current_user.short_name");
      expect(Workspace.currentUser.shortName.value).toBe("${workspace.current_user.short_name}");
    });
  });

  describe("filePath", () => {
    it("should be a Variable with correct path", () => {
      expect(Workspace.filePath).toBeInstanceOf(Variable);
      expect(Workspace.filePath.path).toBe("workspace.file_path");
      expect(Workspace.filePath.value).toBe("${workspace.file_path}");
    });
  });

  describe("rootPath", () => {
    it("should be a Variable with correct path", () => {
      expect(Workspace.rootPath).toBeInstanceOf(Variable);
      expect(Workspace.rootPath.path).toBe("workspace.root_path");
      expect(Workspace.rootPath.value).toBe("${workspace.root_path}");
    });
  });
});
