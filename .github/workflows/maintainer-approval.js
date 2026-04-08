const fs = require("fs");
const path = require("path");

// Parse .github/OWNERS (same format as CODEOWNERS).
// Returns array of { pattern, owners } rules.
function parseOwners() {
  const ownersPath = path.join(
    process.env.GITHUB_WORKSPACE,
    ".github",
    "OWNERS"
  );
  const lines = fs.readFileSync(ownersPath, "utf-8").split("\n");
  const rules = [];
  for (const raw of lines) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const parts = line.split(/\s+/);
    if (parts.length < 2) continue;
    const pattern = parts[0];
    // Strip @ prefix, skip team refs (org/team).
    const owners = parts
      .slice(1)
      .filter((p) => p.startsWith("@") && !p.includes("/"))
      .map((p) => p.slice(1));
    rules.push({ pattern, owners });
  }
  return rules;
}

// Get core team from the * catch-all rule.
function getCoreTeam(rules) {
  const catchAll = rules.find((r) => r.pattern === "*");
  return catchAll ? catchAll.owners : [];
}

// Match a filepath against an OWNERS pattern.
// Supports: "*" (catch-all), "/dir/" (prefix), "/path/file" (exact).
function ownersMatch(pattern, filepath) {
  if (pattern === "*") return true;
  let p = pattern;
  if (p.startsWith("/")) p = p.slice(1);
  if (p.endsWith("/")) return filepath.startsWith(p);
  return filepath === p;
}

// Find which owners match a given file (last match wins, like CODEOWNERS).
function findOwners(filepath, rules) {
  let matched = [];
  for (const rule of rules) {
    if (ownersMatch(rule.pattern, filepath)) {
      matched = rule.owners;
    }
  }
  return matched;
}

// Check if the PR author is exempted.
// If ALL changed files are owned by non-core-team owners that include the
// author, the PR can merge with any approval (not necessarily core team).
function isExempted(authorLogin, files, rules, coreTeam) {
  const coreSet = new Set(coreTeam);
  for (const { filename } of files) {
    const owners = findOwners(filename, rules);
    const nonCoreOwners = owners.filter((o) => !coreSet.has(o));
    if (nonCoreOwners.length === 0 || !nonCoreOwners.includes(authorLogin)) {
      return false;
    }
  }
  return true;
}

module.exports = async ({ github, context, core }) => {
  const rules = parseOwners();
  const coreTeam = getCoreTeam(rules);

  if (coreTeam.length === 0) {
    core.setFailed(
      "Could not determine core team from .github/OWNERS (no * rule found)."
    );
    return;
  }

  const reviews = await github.paginate(github.rest.pulls.listReviews, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number,
  });

  const coreTeamApproved = reviews.some(
    ({ state, user: { login } }) =>
      state === "APPROVED" && coreTeam.includes(login)
  );

  if (coreTeamApproved) {
    return;
  }

  // Check exemption rules based on file ownership.
  const { pull_request: pr } = context.payload;
  const authorLogin = pr?.user?.login;

  const files = await github.paginate(github.rest.pulls.listFiles, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number,
  });

  if (authorLogin && isExempted(authorLogin, files, rules, coreTeam)) {
    const hasAnyApproval = reviews.some(({ state }) => state === "APPROVED");
    if (!hasAnyApproval) {
      core.setFailed(
        "PR from exempted author still needs at least one approval."
      );
    }
    return;
  }

  core.setFailed(
    `Requires approval from a core team member: ${coreTeam.join(", ")}.`
  );
};
