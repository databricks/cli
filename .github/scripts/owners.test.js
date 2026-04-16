const { describe, it, before, after } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");

const {
  parseOwnerTeams,
  ownersMatch,
  parseOwnersFile,
  findOwners,
  getMaintainers,
  getOwnershipGroups,
} = require("./owners");

// --- ownersMatch ---

describe("ownersMatch", () => {
  it("* matches everything", () => {
    assert.ok(ownersMatch("*", "any/file/path.go"));
    assert.ok(ownersMatch("*", "README.md"));
    assert.ok(ownersMatch("*", ""));
  });

  it("/dir/ prefix matches files under that directory", () => {
    assert.ok(ownersMatch("/cmd/pipelines/", "cmd/pipelines/foo.go"));
    assert.ok(ownersMatch("/cmd/pipelines/", "cmd/pipelines/sub/bar.go"));
  });

  it("/dir/ does NOT match files in other directories", () => {
    assert.ok(!ownersMatch("/cmd/pipelines/", "cmd/other/foo.go"));
    assert.ok(!ownersMatch("/cmd/pipelines/", "cmd/pipeline/foo.go"));
    assert.ok(!ownersMatch("/cmd/pipelines/", "bundle/pipelines/foo.go"));
  });

  it("exact file match", () => {
    assert.ok(ownersMatch("/some/file.go", "some/file.go"));
    assert.ok(!ownersMatch("/some/file.go", "some/other.go"));
    assert.ok(!ownersMatch("/some/file.go", "some/file.go/extra"));
  });

  it("leading / is stripped for matching", () => {
    assert.ok(ownersMatch("/bundle/", "bundle/config.go"));
    assert.ok(ownersMatch("/README.md", "README.md"));
  });
});

// --- parseOwnersFile ---

describe("parseOwnersFile", () => {
  let tmpDir;
  let ownersPath;

  before(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "owners-test-"));
    ownersPath = path.join(tmpDir, "OWNERS");
  });

  after(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });

  it("parses rules with owners", () => {
    fs.writeFileSync(
      ownersPath,
      [
        "* @alice @bob",
        "/cmd/pipelines/ @carol",
      ].join("\n")
    );
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 2);
    assert.equal(rules[0].pattern, "*");
    assert.deepEqual(rules[0].owners, ["alice", "bob"]);
    assert.equal(rules[1].pattern, "/cmd/pipelines/");
    assert.deepEqual(rules[1].owners, ["carol"]);
  });

  it("filters out team refs by default", () => {
    fs.writeFileSync(
      ownersPath,
      "/cmd/apps/ @databricks/eng-apps-devex @alice\n"
    );
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 1);
    assert.deepEqual(rules[0].owners, ["alice"]);
  });

  it("includes team refs with includeTeams option", () => {
    fs.writeFileSync(
      ownersPath,
      "/cmd/apps/ @databricks/eng-apps-devex @alice\n"
    );
    const rules = parseOwnersFile(ownersPath, { includeTeams: true });
    assert.equal(rules.length, 1);
    assert.deepEqual(rules[0].owners, ["databricks/eng-apps-devex", "alice"]);
  });

  it("skips comments and blank lines", () => {
    fs.writeFileSync(
      ownersPath,
      [
        "# This is a comment",
        "",
        "  # indented comment",
        "* @alice",
        "",
        "/cmd/ @bob",
      ].join("\n")
    );
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 2);
  });

  it("strips @ prefix from owners", () => {
    fs.writeFileSync(ownersPath, "* @alice @bob\n");
    const rules = parseOwnersFile(ownersPath);
    assert.deepEqual(rules[0].owners, ["alice", "bob"]);
  });

  it("skips lines with only a pattern and no owners", () => {
    fs.writeFileSync(ownersPath, "/lonely/\n* @alice\n");
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 1);
    assert.equal(rules[0].pattern, "*");
  });
});

// --- parseOwnerTeams ---

describe("parseOwnerTeams", () => {
  let tmpDir;

  before(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "ownerteams-test-"));
  });

  after(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });

  it("parses team definitions", () => {
    const teamsPath = path.join(tmpDir, "OWNERTEAMS");
    fs.writeFileSync(teamsPath, "team:platform @alice @bob @carol\n");
    const teams = parseOwnerTeams(teamsPath);
    assert.equal(teams.size, 1);
    assert.deepEqual(teams.get("team:platform"), ["alice", "bob", "carol"]);
  });

  it("parses multiple teams", () => {
    const teamsPath = path.join(tmpDir, "OWNERTEAMS");
    fs.writeFileSync(teamsPath, "team:platform @alice @bob\nteam:bundle @carol @dave\n");
    const teams = parseOwnerTeams(teamsPath);
    assert.equal(teams.size, 2);
    assert.deepEqual(teams.get("team:platform"), ["alice", "bob"]);
    assert.deepEqual(teams.get("team:bundle"), ["carol", "dave"]);
  });

  it("skips comments and blank lines", () => {
    const teamsPath = path.join(tmpDir, "OWNERTEAMS");
    fs.writeFileSync(teamsPath, "# comment\n\nteam:platform @alice\n");
    const teams = parseOwnerTeams(teamsPath);
    assert.equal(teams.size, 1);
  });

  it("returns empty map if file does not exist", () => {
    const teams = parseOwnerTeams(path.join(tmpDir, "NONEXISTENT"));
    assert.equal(teams.size, 0);
  });
});

// --- parseOwnersFile with team aliases ---

describe("parseOwnersFile with OWNERTEAMS", () => {
  let tmpDir;
  let ownersPath;
  let teamsPath;

  before(() => {
    tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "owners-teams-test-"));
    ownersPath = path.join(tmpDir, "OWNERS");
    teamsPath = path.join(tmpDir, "OWNERTEAMS");
  });

  after(() => {
    fs.rmSync(tmpDir, { recursive: true });
  });

  it("expands team aliases to members", () => {
    fs.writeFileSync(teamsPath, "team:platform @alice @bob\n");
    fs.writeFileSync(ownersPath, "/cmd/auth/ team:platform\n");
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 1);
    assert.deepEqual(rules[0].owners, ["alice", "bob"]);
  });

  it("mixes team aliases with individual owners", () => {
    fs.writeFileSync(teamsPath, "team:platform @alice @bob\n");
    fs.writeFileSync(ownersPath, "/cmd/auth/ team:platform @carol\n");
    const rules = parseOwnersFile(ownersPath);
    assert.equal(rules.length, 1);
    assert.deepEqual(rules[0].owners, ["alice", "bob", "carol"]);
  });

  it("unknown team alias is ignored", () => {
    fs.writeFileSync(teamsPath, "team:platform @alice\n");
    fs.writeFileSync(ownersPath, "/cmd/auth/ team:unknown @bob\n");
    const rules = parseOwnersFile(ownersPath);
    assert.deepEqual(rules[0].owners, ["bob"]);
  });

  it("works without OWNERTEAMS file", () => {
    const tmpDir2 = fs.mkdtempSync(path.join(os.tmpdir(), "owners-noteams-"));
    const ownersPath2 = path.join(tmpDir2, "OWNERS");
    fs.writeFileSync(ownersPath2, "* @alice\n");
    const rules = parseOwnersFile(ownersPath2);
    assert.deepEqual(rules[0].owners, ["alice"]);
    fs.rmSync(tmpDir2, { recursive: true });
  });
});

// --- findOwners ---

describe("findOwners", () => {
  const rules = [
    { pattern: "*", owners: ["maintainer1", "maintainer2"] },
    { pattern: "/cmd/pipelines/", owners: ["pipelinesOwner"] },
    { pattern: "/cmd/apps/", owners: ["appsOwner"] },
  ];

  it("last match wins", () => {
    const owners = findOwners("cmd/pipelines/foo.go", rules);
    assert.deepEqual(owners, ["pipelinesOwner"]);
  });

  it("file matching only * returns catch-all owners", () => {
    const owners = findOwners("README.md", rules);
    assert.deepEqual(owners, ["maintainer1", "maintainer2"]);
  });

  it("file matching specific rule returns that rule's owners", () => {
    const owners = findOwners("cmd/apps/main.go", rules);
    assert.deepEqual(owners, ["appsOwner"]);
  });

  it("returns empty array when no rules match", () => {
    const noWildcard = [{ pattern: "/cmd/pipelines/", owners: ["owner1"] }];
    const owners = findOwners("bundle/config.go", noWildcard);
    assert.deepEqual(owners, []);
  });
});

// --- getMaintainers ---

describe("getMaintainers", () => {
  it("returns owners from * rule", () => {
    const rules = [
      { pattern: "*", owners: ["alice", "bob"] },
      { pattern: "/cmd/", owners: ["carol"] },
    ];
    assert.deepEqual(getMaintainers(rules), ["alice", "bob"]);
  });

  it("returns empty array if no * rule", () => {
    const rules = [{ pattern: "/cmd/", owners: ["carol"] }];
    assert.deepEqual(getMaintainers(rules), []);
  });
});

// --- getOwnershipGroups ---

describe("getOwnershipGroups", () => {
  const rules = [
    { pattern: "*", owners: ["maintainer"] },
    { pattern: "/cmd/pipelines/", owners: ["pipelinesOwner"] },
    { pattern: "/cmd/apps/", owners: ["appsOwner"] },
    { pattern: "/bundle/", owners: ["bundleOwner"] },
  ];

  it("single file matching one rule -> one group", () => {
    const groups = getOwnershipGroups(["cmd/pipelines/foo.go"], rules);
    assert.equal(groups.size, 1);
    assert.ok(groups.has("/cmd/pipelines/"));
    assert.deepEqual(groups.get("/cmd/pipelines/").owners, ["pipelinesOwner"]);
    assert.deepEqual(groups.get("/cmd/pipelines/").files, ["cmd/pipelines/foo.go"]);
  });

  it("multiple files matching same rule -> grouped together", () => {
    const groups = getOwnershipGroups(
      ["cmd/pipelines/foo.go", "cmd/pipelines/bar.go"],
      rules
    );
    assert.equal(groups.size, 1);
    assert.deepEqual(groups.get("/cmd/pipelines/").files, [
      "cmd/pipelines/foo.go",
      "cmd/pipelines/bar.go",
    ]);
  });

  it("files matching different rules -> separate groups", () => {
    const groups = getOwnershipGroups(
      ["cmd/pipelines/foo.go", "cmd/apps/bar.go"],
      rules
    );
    assert.equal(groups.size, 2);
    assert.ok(groups.has("/cmd/pipelines/"));
    assert.ok(groups.has("/cmd/apps/"));
  });

  it("file matching only * -> group with * key", () => {
    const groups = getOwnershipGroups(["README.md"], rules);
    assert.equal(groups.size, 1);
    assert.ok(groups.has("*"));
    assert.deepEqual(groups.get("*").owners, ["maintainer"]);
    assert.deepEqual(groups.get("*").files, ["README.md"]);
  });

  it("file matching no rule -> skipped", () => {
    const noWildcard = [{ pattern: "/cmd/pipelines/", owners: ["owner1"] }];
    const groups = getOwnershipGroups(["unrelated/file.go"], noWildcard);
    assert.equal(groups.size, 0);
  });

  it("cross-domain: /cmd/pipelines/ and /cmd/apps/ -> two groups", () => {
    const groups = getOwnershipGroups(
      [
        "cmd/pipelines/a.go",
        "cmd/pipelines/b.go",
        "cmd/apps/c.go",
      ],
      rules
    );
    assert.equal(groups.size, 2);
    assert.deepEqual(groups.get("/cmd/pipelines/").files, [
      "cmd/pipelines/a.go",
      "cmd/pipelines/b.go",
    ]);
    assert.deepEqual(groups.get("/cmd/apps/").files, ["cmd/apps/c.go"]);
  });

  it("mixed: domain files + *-only files -> both groups present", () => {
    const groups = getOwnershipGroups(
      ["cmd/pipelines/a.go", "README.md"],
      rules
    );
    assert.equal(groups.size, 2);
    assert.ok(groups.has("/cmd/pipelines/"));
    assert.ok(groups.has("*"));
  });
});
