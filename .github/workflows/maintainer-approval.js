const path = require("path");
const { execSync } = require("child_process");
const {
  parseOwnersFile,
  findOwners,
  getMaintainers,
  getOwnershipGroups,
} = require("../scripts/owners");

// --- Approval helpers ---

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
      return false;
    }
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
 * Find which approver (if any) satisfies a group's ownership requirement.
 * Returns the login of the first matching approver, or null.
 */
async function findGroupApprover(owners, approverLogins, github, org, core) {
  const approverSet = new Set(approverLogins.map(l => l.toLowerCase()));
  for (const owner of owners) {
    if (owner.includes("/")) {
      const teamSlug = owner.split("/")[1];
      for (const approver of approverLogins) {
        if (await isTeamMember(github, org, teamSlug, approver, core)) {
          return approver;
        }
      }
    } else if (approverSet.has(owner.toLowerCase())) {
      return owner;
    }
  }
  return null;
}

/**
 * Per-path approval check. Each ownership group needs at least one
 * approval from its owners. Files matching only "*" require a maintainer.
 * Returns groups, approvedBy map, and coverage info.
 */
async function checkPerPathApproval(files, rulesWithTeams, approverLogins, github, org, core) {
  const groups = getOwnershipGroups(files.map(f => f.filename), rulesWithTeams);
  const approvedBy = new Map();

  if (groups.has("*")) {
    // Still check non-wildcard groups for comment building
    for (const [pattern, { owners }] of groups) {
      if (pattern === "*") continue;
      const approver = await findGroupApprover(owners, approverLogins, github, org, core);
      if (approver) approvedBy.set(pattern, approver);
    }
    return {
      allCovered: false,
      hasWildcardFiles: true,
      wildcardFiles: groups.get("*").files,
      groups,
      approvedBy,
    };
  }

  const uncovered = [];
  for (const [pattern, { owners }] of groups) {
    const approver = await findGroupApprover(owners, approverLogins, github, org, core);
    if (approver) {
      approvedBy.set(pattern, approver);
    } else {
      uncovered.push({ pattern, owners });
    }
  }
  return { allCovered: uncovered.length === 0, uncovered, groups, approvedBy };
}

// --- Git history & scoring helpers ---

const MENTION_REVIEWERS = true;
const OWNERS_LINK = "[OWNERS](.github/OWNERS)";
const MARKER = "<!-- MAINTAINER_APPROVAL -->";
const STATUS_CONTEXT = "maintainer-approval";

const loginCache = {};

function classifyFile(filepath, totalFiles) {
  const base = path.basename(filepath);
  if (base.startsWith("out.") || base === "output.txt") {
    return 0.01 / Math.max(totalFiles, 1);
  }
  if (filepath.startsWith("acceptance/") || filepath.startsWith("integration/")) {
    return 0.2;
  }
  if (filepath.endsWith("_test.go")) return 0.3;
  return filepath.endsWith(".go") ? 1.0 : 0.5;
}

function gitLog(filepath) {
  try {
    const out = execSync(
      `git log -50 --no-merges --since="12 months ago" --format="%H|%an|%aI" -- "${filepath}"`,
      { encoding: "utf-8" }
    );
    const entries = [];
    for (const line of out.split("\n")) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      const parts = trimmed.split("|", 3);
      if (parts.length !== 3) continue;
      const date = new Date(parts[2]);
      if (isNaN(date.getTime())) continue;
      entries.push({ sha: parts[0], name: parts[1], date });
    }
    return entries;
  } catch {
    return [];
  }
}

async function resolveLogin(github, owner, repo, sha, authorName) {
  if (authorName in loginCache) return loginCache[authorName];
  try {
    const { data } = await github.rest.repos.getCommit({ owner, repo, ref: sha });
    const login = data.author?.login || null;
    loginCache[authorName] = login;
    return login;
  } catch {
    loginCache[authorName] = null;
    return null;
  }
}

function parseOwnersForFiles(changedFiles, ownersPath) {
  const rules = parseOwnersFile(ownersPath, { includeTeams: true });
  const allOwners = new Set();
  for (const filepath of changedFiles) {
    for (const o of findOwners(filepath, rules)) allOwners.add(o);
  }
  return Array.from(allOwners).sort();
}

async function scoreContributors(files, prAuthor, now, github, owner, repo) {
  const scores = {};
  const dirScores = {};
  let scoredCount = 0;
  const authorLogin = (prAuthor || "").toLowerCase();
  const totalFiles = files.length;

  for (const filepath of files) {
    const weight = classifyFile(filepath, totalFiles);
    let history = gitLog(filepath);
    if (history.length === 0) {
      const parent = path.dirname(filepath);
      if (parent && parent !== ".") {
        history = gitLog(parent);
      }
    }
    if (history.length === 0) continue;

    const topDir = path.dirname(filepath) || ".";
    let fileContributed = false;
    for (const { sha, name, date } of history) {
      if (name.endsWith("[bot]")) continue;
      const login = await resolveLogin(github, owner, repo, sha, name);
      if (!login || login.toLowerCase() === authorLogin) continue;
      const daysAgo = Math.max(0, (now - date) / 86400000);
      const s = weight * Math.pow(0.5, daysAgo / 150);
      scores[login] = (scores[login] || 0) + s;
      if (!dirScores[login]) dirScores[login] = {};
      dirScores[login][topDir] = (dirScores[login][topDir] || 0) + s;
      fileContributed = true;
    }
    if (fileContributed) scoredCount++;
  }
  return { scores, dirScores, scoredCount };
}

function topDirs(ds, n = 3) {
  return Object.entries(ds || {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, n)
    .map(([d]) => d);
}

function fmtReviewer(login, dirs) {
  const mention = MENTION_REVIEWERS ? `@${login}` : login;
  const dirList = dirs.map((d) => `\`${d}/\``).join(", ");
  return `- ${mention} -- recent work in ${dirList}`;
}

function selectReviewers(ss) {
  if (ss.length === 0) return [];
  const out = [ss[0]];
  if (ss.length >= 2 && ss[0][1] < 1.5 * ss[1][1]) {
    out.push(ss[1]);
    if (ss.length >= 3 && ss[1][1] < 1.5 * ss[2][1]) {
      out.push(ss[2]);
    }
  }
  return out;
}

function fmtEligible(owners) {
  if (MENTION_REVIEWERS) return owners.map((o) => `@${o}`).join(", ");
  return owners.join(", ");
}

async function countRecentReviews(github, owner, repo, logins, days = 30) {
  const since = new Date(Date.now() - days * 86400000)
    .toISOString()
    .slice(0, 10);
  const counts = {};
  for (const login of logins) {
    try {
      const { data } = await github.rest.search.issuesAndPullRequests({
        q: `repo:${owner}/${repo} reviewed-by:${login} is:pr created:>${since}`,
      });
      counts[login] = data.total_count;
    } catch {
      // skip on error
    }
  }
  return counts;
}

async function selectRoundRobin(github, owner, repo, eligibleOwners, prAuthor) {
  const candidates = eligibleOwners
    .filter((o) => !o.includes("/") && o.toLowerCase() !== (prAuthor || "").toLowerCase());
  if (candidates.length === 0) return null;
  const counts = await countRecentReviews(github, owner, repo, candidates);
  if (Object.keys(counts).length === 0) {
    return candidates[Math.floor(Math.random() * candidates.length)];
  }
  return candidates.reduce((best, c) =>
    (counts[c] || 0) < (counts[best] || 0) ? c : best
  );
}

// --- Comment builders ---

function buildPendingPerGroupComment(groups, scores, dirScores, approvedBy, maintainers, prAuthor) {
  const authorLower = (prAuthor || "").toLowerCase();
  const lines = [MARKER, "## Approval status: pending", ""];

  for (const [pattern, { owners, files }] of groups) {
    if (pattern === "*") continue;

    const approver = approvedBy.get(pattern);
    if (approver) {
      lines.push(`### \`${pattern}\` - approved by @${approver}`);
    } else {
      lines.push(`### \`${pattern}\` - needs approval`);
    }
    lines.push(`Files: ${files.map(f => `\`${f}\``).join(", ")}`);

    const teams = owners.filter(o => o.includes("/"));
    const individuals = owners.filter(o => !o.includes("/") && o.toLowerCase() !== authorLower);

    if (teams.length > 0) {
      lines.push(`Teams: ${teams.map(t => `@${t}`).join(", ")}`);
    }

    if (!approver && individuals.length > 0) {
      const scored = individuals.map(o => [o, scores[o] || 0]).sort((a, b) => b[1] - a[1]);
      if (scored[0][1] > 0) {
        lines.push(`Suggested: @${scored[0][0]}`);
        const rest = scored.slice(1).map(([o]) => o);
        if (rest.length > 0) {
          lines.push(`Also eligible: ${fmtEligible(rest)}`);
        }
      } else {
        lines.push(`Eligible: ${fmtEligible(individuals)}`);
      }
    }
    lines.push("");
  }

  const starGroup = groups.get("*");
  if (starGroup) {
    lines.push("### General files (require maintainer)");
    lines.push(`Files: ${starGroup.files.map(f => `\`${f}\``).join(", ")}`);

    const maintainerSet = new Set(maintainers.map(m => m.toLowerCase()));
    const maintainerScores = Object.entries(scores)
      .filter(([login]) =>
        login.toLowerCase() !== authorLower && maintainerSet.has(login.toLowerCase())
      )
      .sort((a, b) => b[1] - a[1]);

    if (maintainerScores.length > 0 && maintainerScores[0][1] > 0) {
      const [login] = maintainerScores[0];
      const dirs = topDirs(dirScores[login]);
      lines.push("Based on git history:");
      lines.push(fmtReviewer(login, dirs));
    } else {
      lines.push(`Pick a maintainer from ${OWNERS_LINK}.`);
    }
    lines.push("");
  }

  const maintainerList = maintainers
    .filter(m => m.toLowerCase() !== authorLower)
    .map(m => `@${m}`)
    .join(", ");

  lines.push(
    `<sub>Any maintainer (${maintainerList}) can approve all areas.`,
    `See ${OWNERS_LINK} for ownership rules.</sub>`
  );

  return lines.join("\n") + "\n";
}

function buildSingleDomainPendingComment(sortedScores, dirScores, scoredCount, eligibleOwners, prAuthor, roundRobinReviewer) {
  const reviewers = selectReviewers(sortedScores);
  const suggestedLogins = new Set(reviewers.map(([login]) => login.toLowerCase()));
  const eligible = eligibleOwners.filter(
    o => o.toLowerCase() !== (prAuthor || "").toLowerCase() && !suggestedLogins.has(o.toLowerCase())
  );

  const lines = [MARKER, "## Waiting for approval", ""];

  if (reviewers.length > 0) {
    lines.push("Based on git history, these people are best suited to review:", "");
    for (const [login] of reviewers) {
      lines.push(fmtReviewer(login, topDirs(dirScores[login])));
    }
    lines.push("");
  } else if (roundRobinReviewer) {
    lines.push(
      "Could not determine reviewers from git history.",
      `Round-robin suggestion: @${roundRobinReviewer}`,
      ""
    );
  }

  if (eligible.length > 0) {
    lines.push(`Eligible reviewers: ${fmtEligible(eligible)}`, "");
  }

  lines.push(`<sub>Suggestions based on git history. See ${OWNERS_LINK} for ownership rules.</sub>`);
  return lines.join("\n") + "\n";
}

// --- Comment management ---

const LEGACY_MARKER = "<!-- REVIEWER_SUGGESTION -->";

/**
 * Delete all marker and legacy marker comments from the PR.
 * Used on success paths to clean up stale pending comments.
 */
async function deleteMarkerComments(github, owner, repo, prNumber) {
  const comments = await github.paginate(github.rest.issues.listComments, {
    owner, repo, issue_number: prNumber,
  });
  for (const c of comments) {
    if (c.body && (c.body.includes(MARKER) || c.body.includes(LEGACY_MARKER))) {
      await github.rest.issues.deleteComment({
        owner, repo, comment_id: c.id,
      });
    }
  }
}

/**
 * Create or edit the marker comment. Skips the edit if the body is unchanged.
 * Cleans up duplicate or legacy marker comments, keeping only the first one.
 */
async function upsertComment(github, owner, repo, prNumber, newBody) {
  const comments = await github.paginate(github.rest.issues.listComments, {
    owner, repo, issue_number: prNumber,
  });
  const markerComments = comments.filter(c =>
    c.body && (c.body.includes(MARKER) || c.body.includes(LEGACY_MARKER))
  );

  if (markerComments.length > 0) {
    const existing = markerComments[0];

    // Clean up duplicates (legacy or accidental), keep the first.
    for (const c of markerComments.slice(1)) {
      await github.rest.issues.deleteComment({
        owner, repo, comment_id: c.id,
      });
    }

    // Skip if body is unchanged.
    if (existing.body === newBody) return;

    await github.rest.issues.updateComment({
      owner, repo, comment_id: existing.id, body: newBody,
    });
    return;
  }

  await github.rest.issues.createComment({
    owner, repo, issue_number: prNumber, body: newBody,
  });
}

// --- Main ---

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
  const owner = context.repo.owner;
  const repo = context.repo.repo;
  const prNumber = context.issue.number;
  const authorLogin = pr?.user?.login;
  const sha = pr.head.sha;
  const checkParams = {
    owner: context.repo.owner,
    repo: context.repo.repo,
    head_sha: sha,
    name: STATUS_CONTEXT,
  };

  const reviews = await github.paginate(github.rest.pulls.listReviews, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: context.issue.number,
  });

  // Maintainer approval -> success with simple comment
  const maintainerApproval = reviews.find(
    ({ state, user }) =>
      state === "APPROVED" && user && maintainers.includes(user.login)
  );
  if (maintainerApproval) {
    const approver = maintainerApproval.user.login;
    core.info(`Maintainer approval from @${approver}`);
    await github.rest.checks.create({
      ...checkParams,
      status: "completed",
      conclusion: "success",
      output: { title: STATUS_CONTEXT, summary: `Approved by @${approver}` },
    });
    await deleteMarkerComments(github, owner, repo, prNumber);
    return;
  }

  // Maintainer-authored PR with any approval -> success
  if (authorLogin && maintainers.includes(authorLogin)) {
    const hasAnyApproval = reviews.some(
      ({ state, user }) =>
        state === "APPROVED" && user && user.login !== authorLogin
    );
    if (hasAnyApproval) {
      core.info(`Maintainer-authored PR approved by a reviewer.`);
      await github.rest.checks.create({
        ...checkParams,
        status: "completed",
        conclusion: "success",
        output: { title: STATUS_CONTEXT, summary: "Approved (maintainer-authored PR)" },
      });
      await deleteMarkerComments(github, owner, repo, prNumber);
      return;
    }
  }

  // Gather approved logins (excluding the PR author).
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

  // Approved PRs get a success check run and return early.
  // Pending PRs intentionally create NO check run or status. The required
  // status check "maintainer-approval" stays as "Expected" (yellow dot) in
  // the GitHub UI, which blocks the merge until approval is granted.
  if (result.allCovered && approverLogins.length > 0) {
    core.info("All ownership groups have per-path approval.");
    await github.rest.checks.create({
      ...checkParams,
      status: "completed",
      conclusion: "success",
      output: { title: STATUS_CONTEXT, summary: "All ownership groups approved" },
    });
    await deleteMarkerComments(github, owner, repo, prNumber);
    return;
  }

  if (result.hasWildcardFiles) {
    const fileList = result.wildcardFiles.join(", ");
    core.info(
      `Files need maintainer review: ${fileList}. ` +
      `Maintainers: ${maintainers.join(", ")}`
    );
  } else if (result.uncovered && result.uncovered.length > 0) {
    const groupList = result.uncovered
      .map(({ pattern, owners }) => `${pattern} (needs: ${owners.join(", ")})`)
      .join("; ");
    core.info(
      `Needs approval: ${groupList}. ` +
      `Alternatively, any maintainer can approve: ${maintainers.join(", ")}.`
    );
  } else {
    core.info(`Waiting for maintainer approval: ${maintainers.join(", ")}`);
  }

  // Score contributors via git history
  const fileNames = files.map(f => f.filename);
  const now = new Date();
  const { scores, dirScores, scoredCount } = await scoreContributors(
    fileNames,
    authorLogin,
    now,
    github,
    owner,
    repo
  );
  const sortedScores = Object.entries(scores).sort((a, b) => b[1] - a[1]);

  // Build pending comment with reviewer suggestions.
  let comment;
  const groups = result.groups;

  if (groups.size >= 2) {
    comment = buildPendingPerGroupComment(
      groups, scores, dirScores, result.approvedBy, maintainers, authorLogin
    );
  } else {
    const eligible = parseOwnersForFiles(fileNames, ownersPath);
    let roundRobin = null;
    if (selectReviewers(sortedScores).length === 0 && eligible.length > 0) {
      roundRobin = await selectRoundRobin(github, owner, repo, eligible, authorLogin);
    }
    comment = buildSingleDomainPendingComment(
      sortedScores, dirScores, scoredCount, eligible, authorLogin, roundRobin
    );
  }

  core.info(comment);
  await upsertComment(github, owner, repo, prNumber, comment);
};
