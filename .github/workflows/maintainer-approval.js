const path = require("path");
const {
  parseOwnersFile,
  getMaintainers,
  getOwnershipGroups,
} = require("../scripts/owners");

/**
 * Check if an approver is a member of a GitHub team.
 * Requires org read access on the token; falls back to false if unavailable.
 */
async function isTeamMember(github, org, teamSlug, login, core) {
  try {
    const { data } = await github.rest.teams.getMembershipForUserInOrg({
      org,
      team_slug: teamSlug,
      username: login,
    });
    return data.state === "active";
  } catch (err) {
    if (err.status === 404) {
      // User is genuinely not a member of this team.
      return false;
    }
    // Permission denied or other error. Log it so it's visible.
    if (core) {
      core.warning(
        `Could not verify team membership for ${login} in ${org}/${teamSlug} ` +
        `(HTTP ${err.status || "unknown"}). Team-based approval may not work ` +
        `without a token with org:read scope.`
      );
    }
    return false;
  }
}

/**
 * Check if any approver matches an owner entry.
 * Owner can be a plain login or an org/team ref (containing "/").
 */
async function ownerHasApproval(owner, approverSet, github, org, core) {
  if (owner.includes("/")) {
    const teamSlug = owner.split("/")[1];
    for (const approver of approverSet) {
      if (await isTeamMember(github, org, teamSlug, approver, core)) {
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
async function checkPerPathApproval(files, rulesWithTeams, approverLogins, github, org, core) {
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
      if (await ownerHasApproval(owner, approverSet, github, org, core)) {
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

const STATUS_CONTEXT = "maintainer-approval";

module.exports = async ({ github, context, core }) => {
  const ownersPath = path.join(
    process.env.GITHUB_WORKSPACE,
    ".github",
    "OWNERS"
  );
  const rulesWithTeams = parseOwnersFile(ownersPath, { includeTeams: true });
  const maintainers = getMaintainers(rulesWithTeams);

  if (maintainers.length === 0) {
    core.setFailed(
      "Could not determine maintainers from .github/OWNERS (no * rule found)."
    );
    return;
  }

  const { pull_request: pr } = context.payload;
  const authorLogin = pr?.user?.login;
  const sha = pr.head.sha;
  const statusParams = {
    owner: context.repo.owner,
    repo: context.repo.repo,
    sha,
    context: STATUS_CONTEXT,
  };

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
    await github.rest.repos.createCommitStatus({
      ...statusParams,
      state: "success",
      description: `Approved by @${approver.user.login}`,
    });
    return;
  }

  // If the PR author is a maintainer, any approval suffices (second pair of eyes).
  if (authorLogin && maintainers.includes(authorLogin)) {
    const hasAnyApproval = reviews.some(
      ({ state, user }) =>
        state === "APPROVED" && user && user.login !== authorLogin
    );
    if (hasAnyApproval) {
      core.info(`Maintainer-authored PR approved by a reviewer.`);
      await github.rest.repos.createCommitStatus({
        ...statusParams,
        state: "success",
        description: "Approved (maintainer-authored PR)",
      });
      return;
    }
  }

  // Gather approved logins (excluding the PR author, since GitHub prevents
  // self-approval, but we filter defensively in case of API edge cases).
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
    rulesWithTeams,
    approverLogins,
    github,
    context.repo.owner,
    core
  );

  if (result.allCovered && approverLogins.length > 0) {
    core.info("All ownership groups have per-path approval.");
    await github.rest.repos.createCommitStatus({
      ...statusParams,
      state: "success",
      description: "All ownership groups approved",
    });
    return;
  }

  if (result.hasWildcardFiles) {
    const fileList = result.wildcardFiles.join(", ");
    const msg =
      `Files need maintainer review: ${fileList}. ` +
      `Maintainers: ${maintainers.join(", ")}`;
    core.info(msg);
    await github.rest.repos.createCommitStatus({
      ...statusParams,
      state: "pending",
      description: msg.length > 140 ? msg.slice(0, 137) + "..." : msg,
    });
    return;
  }

  if (result.uncovered && result.uncovered.length > 0) {
    const groupList = result.uncovered
      .map(({ pattern, owners }) => `${pattern} (needs: ${owners.join(", ")})`)
      .join("; ");
    const msg = `Needs approval: ${groupList}`;
    core.info(
      `${msg}. Alternatively, any maintainer can approve: ${maintainers.join(", ")}.`
    );
    await github.rest.repos.createCommitStatus({
      ...statusParams,
      state: "pending",
      description: msg.length > 140 ? msg.slice(0, 137) + "..." : msg,
    });
    return;
  }

  const msg = `Waiting for maintainer approval: ${maintainers.join(", ")}`;
  core.info(msg);
  await github.rest.repos.createCommitStatus({
    ...statusParams,
    state: "pending",
    description: msg.length > 140 ? msg.slice(0, 137) + "..." : msg,
  });
};
