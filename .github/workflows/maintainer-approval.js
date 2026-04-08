const path = require("path");
const {
  parseOwnersFile,
  findOwners,
  getMaintainers,
  getOwnershipGroups,
} = require("../scripts/owners");

/**
 * Check if an approver is a member of a GitHub team.
 * Requires org read access on the token; falls back to false if unavailable.
 */
async function isTeamMember(github, org, teamSlug, login) {
  try {
    await github.rest.teams.getMembershipForUserInOrg({
      org,
      team_slug: teamSlug,
      username: login,
    });
    return true;
  } catch {
    return false;
  }
}

/**
 * Check if any approver matches an owner entry.
 * Owner can be a plain login or an org/team ref (containing "/").
 */
async function ownerHasApproval(owner, approverSet, github, org) {
  if (owner.includes("/")) {
    const teamSlug = owner.split("/")[1];
    for (const approver of approverSet) {
      if (await isTeamMember(github, org, teamSlug, approver)) {
        return true;
      }
    }
    return false;
  }
  return approverSet.has(owner.toLowerCase());
}

/**
 * Per-path approval check. Each ownership group needs at least one
 * approval from its owners. Files matching only "*" require a maintainer.
 */
async function checkPerPathApproval(files, rules, approverLogins, github, org) {
  const rulesWithTeams = parseOwnersFile(
    path.join(process.env.GITHUB_WORKSPACE, ".github", "OWNERS"),
    { includeTeams: true }
  );
  const groups = getOwnershipGroups(files.map(f => f.filename), rulesWithTeams);
  const approverSet = new Set(approverLogins.map(l => l.toLowerCase()));

  if (groups.has("*")) {
    return {
      allCovered: false,
      hasWildcardFiles: true,
      wildcardFiles: groups.get("*").files,
    };
  }

  const uncovered = [];
  for (const [pattern, { owners }] of groups) {
    let hasApproval = false;
    for (const owner of owners) {
      if (await ownerHasApproval(owner, approverSet, github, org)) {
        hasApproval = true;
        break;
      }
    }
    if (!hasApproval) {
      uncovered.push({ pattern, owners });
    }
  }
  return { allCovered: uncovered.length === 0, uncovered };
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

  // Gather approved logins (excluding the PR author, since GitHub prevents
  // self-approval, but we filter defensively in case of API edge cases).
  const { pull_request: pr } = context.payload;
  const authorLogin = pr?.user?.login;

  const approverLogins = reviews
    .filter(
      ({ state, user }) =>
        state === "APPROVED" && user && user.login !== authorLogin
    )
    .map(({ user }) => user.login);

  const files = await github.paginate(github.rest.pulls.listFiles, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number,
  });

  const result = await checkPerPathApproval(
    files,
    rules,
    approverLogins,
    github,
    context.repo.owner
  );

  if (result.allCovered && approverLogins.length > 0) {
    core.info("All ownership groups have per-path approval.");
    return;
  }

  if (result.hasWildcardFiles) {
    core.setFailed(
      `Some files only match the wildcard (*) rule and require a maintainer: ` +
        `${result.wildcardFiles.join(", ")}. ` +
        `Maintainers: ${maintainers.join(", ")}.`
    );
    return;
  }

  if (result.uncovered && result.uncovered.length > 0) {
    const groupList = result.uncovered
      .map(({ pattern, owners }) => `${pattern} (needs: ${owners.join(", ")})`)
      .join("; ");
    core.setFailed(
      `Missing per-path approval. Uncovered groups: ${groupList}. ` +
        `Alternatively, any maintainer can approve: ${maintainers.join(", ")}.`
    );
    return;
  }

  core.setFailed(
    `Requires approval from a maintainer: ${maintainers.join(", ")}.`
  );
};
