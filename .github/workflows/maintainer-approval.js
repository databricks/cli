const path = require("path");
const {
  parseOwnersFile,
  findOwners,
  getCoreTeam,
} = require("../scripts/owners");

// Check if the PR author is exempted.
// If ALL changed files are owned by non-core-team owners that include the
// author, the PR can merge with any approval (not necessarily core team).
function isExempted(authorLogin, files, rules, coreTeam) {
  if (files.length === 0) return false;
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
  const ownersPath = path.join(
    process.env.GITHUB_WORKSPACE,
    ".github",
    "OWNERS"
  );
  const rules = parseOwnersFile(ownersPath);
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
    ({ state, user }) =>
      state === "APPROVED" && user && coreTeam.includes(user.login)
  );

  if (coreTeamApproved) {
    const approver = reviews.find(
      ({ state, user }) =>
        state === "APPROVED" && user && coreTeam.includes(user.login)
    );
    core.info(`Core team approval from @${approver.user.login}`);
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
