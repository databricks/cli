const { describe, it, before, after } = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const os = require("os");
const path = require("path");

const runModule = require("./maintainer-approval");

// --- Test helpers ---

function makeTmpOwners(content) {
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "approval-test-"));
  const ghDir = path.join(tmpDir, ".github");
  fs.mkdirSync(ghDir);
  fs.writeFileSync(path.join(ghDir, "OWNERS"), content);
  return tmpDir;
}

const OWNERS_CONTENT = [
  "* @maintainer1 @maintainer2",
  "/cmd/pipelines/ @jefferycheng1 @kanterov",
  "/cmd/apps/ @databricks/eng-apps-devex",
  "/bundle/ @bundleowner",
].join("\n");

function makeContext({ author = "someuser", sha = "abc123", prNumber = 42 } = {}) {
  return {
    repo: { owner: "databricks", repo: "cli" },
    issue: { number: prNumber },
    payload: {
      pull_request: {
        number: prNumber,
        user: { login: author },
        head: { sha },
      },
    },
  };
}

function makeCore() {
  const log = { info: [], warning: [], failed: [] };
  return {
    info: (msg) => log.info.push(msg),
    warning: (msg) => log.warning.push(msg),
    setFailed: (msg) => log.failed.push(msg),
    _log: log,
  };
}

/**
 * Build a mock GitHub API object.
 *
 * @param {Object} opts
 * @param {Array} opts.reviews - PR reviews to return
 * @param {Array} opts.files - PR files to return (objects with .filename)
 * @param {Object} opts.teamMembers - { teamSlug: [logins] }
 * @param {Array} opts.existingComments - Existing PR comments to return
 */
function makeGithub({ reviews = [], files = [], teamMembers = {}, existingComments = [] } = {}) {
  const listReviews = Symbol("listReviews");
  const listFiles = Symbol("listFiles");
  const listComments = Symbol("listComments");
  const statuses = [];
  const createdComments = [];
  const deletedCommentIds = [];

  const github = {
    paginate: async (endpoint, _opts) => {
      if (endpoint === listReviews) return reviews;
      if (endpoint === listFiles) return files;
      if (endpoint === listComments) return existingComments;
      return [];
    },
    rest: {
      pulls: {
        listReviews,
        listFiles,
      },
      repos: {
        createCommitStatus: async (params) => {
          statuses.push(params);
        },
      },
      issues: {
        listComments,
        deleteComment: async (params) => {
          deletedCommentIds.push(params.comment_id);
        },
        createComment: async (params) => {
          createdComments.push(params);
        },
      },
      teams: {
        getMembershipForUserInOrg: async ({ team_slug, username }) => {
          if (teamMembers[team_slug]?.includes(username)) {
            return { data: { state: "active" } };
          }
          const err = new Error("Not found");
          err.status = 404;
          throw err;
        },
      },
    },
    _statuses: statuses,
    _comments: createdComments,
    _deletedCommentIds: deletedCommentIds,
  };
  return github;
}

// --- Tests ---

describe("maintainer-approval", () => {
  let tmpDir;
  let originalWorkspace;

  before(() => {
    originalWorkspace = process.env.GITHUB_WORKSPACE;
    tmpDir = makeTmpOwners(OWNERS_CONTENT);
    process.env.GITHUB_WORKSPACE = tmpDir;
  });

  after(() => {
    if (originalWorkspace !== undefined) {
      process.env.GITHUB_WORKSPACE = originalWorkspace;
    } else {
      delete process.env.GITHUB_WORKSPACE;
    }
    fs.rmSync(tmpDir, { recursive: true });
  });

  it("maintainer approved -> success", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "maintainer1" } },
      ],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "success");
    assert.ok(github._statuses[0].description.includes("maintainer1"));
  });

  it("maintainer-authored PR with any approval -> success", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "randomreviewer" } },
      ],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext({ author: "maintainer1" });

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "success");
    assert.ok(github._statuses[0].description.includes("maintainer-authored"));
  });

  it("single domain, owner approved -> success", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
      ],
      files: [
        { filename: "cmd/pipelines/foo.go" },
        { filename: "cmd/pipelines/bar.go" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "success");
  });

  it("cross-domain, both approved -> success", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
        { state: "APPROVED", user: { login: "bundleowner" } },
      ],
      files: [
        { filename: "cmd/pipelines/foo.go" },
        { filename: "bundle/config.go" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "success");
  });

  it("cross-domain, one missing -> pending", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
      ],
      files: [
        { filename: "cmd/pipelines/foo.go" },
        { filename: "bundle/config.go" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
    assert.ok(github._statuses[0].description.includes("/bundle/"));
  });

  it("wildcard files present -> pending, mentions maintainer", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "randomreviewer" } },
      ],
      files: [{ filename: "README.md" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
    assert.ok(github._statuses[0].description.includes("maintainer"));
  });

  it("no approvals at all -> pending", async () => {
    const github = makeGithub({
      reviews: [],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
  });

  it("team member approved -> success for team-owned path", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "teamdev1" } },
      ],
      files: [{ filename: "cmd/apps/main.go" }],
      teamMembers: { "eng-apps-devex": ["teamdev1"] },
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "success");
  });

  it("non-team-member approval for team-owned path -> pending", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "outsider" } },
      ],
      files: [{ filename: "cmd/apps/main.go" }],
      teamMembers: { "eng-apps-devex": [] },
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
  });

  it("CHANGES_REQUESTED does not count as approval", async () => {
    const github = makeGithub({
      reviews: [
        { state: "CHANGES_REQUESTED", user: { login: "jefferycheng1" } },
      ],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
  });

  it("self-approval by PR author is excluded", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
      ],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext({ author: "jefferycheng1" });

    await runModule({ github, context, core });

    assert.equal(github._statuses.length, 1);
    assert.equal(github._statuses[0].state, "pending");
  });

  it("no * rule in OWNERS -> setFailed", async () => {
    const noWildcardDir = makeTmpOwners("/cmd/pipelines/ @jefferycheng1\n");
    const oldWorkspace = process.env.GITHUB_WORKSPACE;
    process.env.GITHUB_WORKSPACE = noWildcardDir;

    const github = makeGithub({ reviews: [], files: [] });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(core._log.failed.length, 1);
    assert.ok(core._log.failed[0].includes("maintainers"));

    process.env.GITHUB_WORKSPACE = oldWorkspace;
    fs.rmSync(noWildcardDir, { recursive: true });
  });

  // --- Comment posting tests ---

  it("posts a comment with MARKER on every run", async () => {
    const github = makeGithub({
      reviews: [],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._comments.length, 1);
    assert.ok(github._comments[0].body.includes("<!-- MAINTAINER_APPROVAL -->"));
  });

  it("maintainer approval posts simple approved comment", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "maintainer1" } },
      ],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._comments.length, 1);
    assert.ok(github._comments[0].body.includes("Approved by @maintainer1"));
    assert.ok(github._comments[0].body.includes("<!-- MAINTAINER_APPROVAL -->"));
  });

  it("deletes existing comment before posting new one", async () => {
    const github = makeGithub({
      reviews: [],
      files: [{ filename: "cmd/pipelines/foo.go" }],
      existingComments: [
        { id: 999, body: "<!-- MAINTAINER_APPROVAL -->\nOld comment" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._deletedCommentIds.length, 1);
    assert.equal(github._deletedCommentIds[0], 999);
    assert.equal(github._comments.length, 1);
    assert.ok(github._comments[0].body.includes("<!-- MAINTAINER_APPROVAL -->"));
  });

  it("does not delete comments without the marker", async () => {
    const github = makeGithub({
      reviews: [],
      files: [{ filename: "cmd/pipelines/foo.go" }],
      existingComments: [
        { id: 888, body: "Some unrelated comment" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._deletedCommentIds.length, 0);
    assert.equal(github._comments.length, 1);
  });

  it("pending single-domain comment includes waiting header", async () => {
    const github = makeGithub({
      reviews: [],
      files: [{ filename: "cmd/pipelines/foo.go" }],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._comments.length, 1);
    assert.ok(github._comments[0].body.includes("## Waiting for approval"));
  });

  it("pending cross-domain comment includes per-group sections", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
      ],
      files: [
        { filename: "cmd/pipelines/foo.go" },
        { filename: "bundle/config.go" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._comments.length, 1);
    const body = github._comments[0].body;
    assert.ok(body.includes("## Approval status: pending"));
    assert.ok(body.includes("`/cmd/pipelines/`"));
    assert.ok(body.includes("`/bundle/`"));
    assert.ok(body.includes("approved by @jefferycheng1"));
    assert.ok(body.includes("needs approval"));
  });

  it("all groups approved comment shows per-group detail", async () => {
    const github = makeGithub({
      reviews: [
        { state: "APPROVED", user: { login: "jefferycheng1" } },
        { state: "APPROVED", user: { login: "bundleowner" } },
      ],
      files: [
        { filename: "cmd/pipelines/foo.go" },
        { filename: "bundle/config.go" },
      ],
    });
    const core = makeCore();
    const context = makeContext();

    await runModule({ github, context, core });

    assert.equal(github._comments.length, 1);
    const body = github._comments[0].body;
    assert.ok(body.includes("## All ownership groups approved"));
    assert.ok(body.includes("Approved by: @jefferycheng1"));
    assert.ok(body.includes("Approved by: @bundleowner"));
  });
});
