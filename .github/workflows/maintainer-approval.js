const path = require("path");
const {
  parseOwnersFile,
  findOwners,
  getMaintainers,
} = require("../scripts/owners");

// Check if the PR author is exempted.
// If ALL changed files are owned by non-maintainer owners that include the
// author, the PR can merge with any approval (not necessarily a maintainer).
function isExempted(authorLogin, files, rules, maintainers) {
  if (files.length === 0) return false;
  const maintainerSet = new Set(maintainers);
  for (const { filename } of files) {
    const owners = findOwners(filename, rules);
    const nonMaintainers = owners.filter((o) => !maintainerSet.has(o));
    if (nonMaintainers.length === 0 || !nonMaintainers.includes(authorLogin)) {
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
  const maintainers = getMaintainers(rules);

  if (maintainers.length === 0) {
    core.setFailed(
      "Could not determine maintainers from .github/OWNERS (no * rule found)."
    );
    return;
  }

  const reviews = await github.paginate(github.rest.pulls.listReviews, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number,
  });

  const maintainerApproved = reviews.some(
    ({ state, user }) =>
      state === "APPROVED" && user && maintainers.includes(user.login)
  );

  if (maintainerApproved) {
    const approver = reviews.find(
      ({ state, user }) =>
        state === "APPROVED" && user && maintainers.includes(user.login)
    );
    core.info(`Maintainer approval from @${approver.user.login}`);
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

  if (authorLogin && isExempted(authorLogin, files, rules, maintainers)) {
    const hasAnyApproval = reviews.some(({ state }) => state === "APPROVED");
    if (!hasAnyApproval) {
      core.setFailed(
        "PR from exempted author still needs at least one approval."
      );
    }
    return;
  }

  core.setFailed(
    `Requires approval from a maintainer: ${maintainers.join(", ")}.`
  );
};
