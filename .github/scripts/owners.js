const fs = require("fs");
const path = require("path");

/**
 * Read a file and return non-empty, non-comment lines split by whitespace.
 * Returns [] if the file does not exist.
 *
 * @param {string} filePath
 * @returns {string[][]} array of whitespace-split tokens per line
 */
function readDataLines(filePath) {
  let content;
  try {
    content = fs.readFileSync(filePath, "utf-8");
  } catch (e) {
    if (e.code === "ENOENT") return [];
    throw e;
  }
  const result = [];
  for (const raw of content.split("\n")) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const parts = line.split(/\s+/);
    if (parts.length >= 2) result.push(parts);
  }
  return result;
}

/**
 * Parse an OWNERTEAMS file into a map of team aliases.
 * Format: "team:<name>  @member1 @member2 ..."
 * Returns Map<string, string[]> where key is "team:<name>" and value is member logins.
 *
 * @param {string} filePath - absolute path to the OWNERTEAMS file
 * @returns {Map<string, string[]>}
 */
function parseOwnerTeams(filePath) {
  const teams = new Map();
  for (const parts of readDataLines(filePath)) {
    if (!parts[0].startsWith("team:")) continue;
    const members = parts.slice(1).filter((p) => p.startsWith("@")).map((p) => p.slice(1));
    teams.set(parts[0], members);
  }
  return teams;
}

/**
 * Parse an OWNERS file (same format as CODEOWNERS).
 * Returns array of { pattern, owners } rules.
 *
 * If an OWNERTEAMS file exists alongside the OWNERS file, "team:<name>"
 * tokens are expanded to their member lists.
 *
 * By default, team refs (org/team) are filtered out and @ is stripped.
 * Pass { includeTeams: true } to keep team refs (with @ stripped).
 *
 * @param {string} filePath - absolute path to the OWNERS file
 * @param {{ includeTeams?: boolean }} [opts]
 * @returns {Array<{pattern: string, owners: string[]}>}
 */
function parseOwnersFile(filePath, opts) {
  const includeTeams = opts && opts.includeTeams;
  const teamsPath = path.join(path.dirname(filePath), "OWNERTEAMS");
  const teams = parseOwnerTeams(teamsPath);
  const rules = [];
  for (const parts of readDataLines(filePath)) {
    const pattern = parts[0];
    const owners = [];
    for (const p of parts.slice(1)) {
      if (p.startsWith("team:") && teams.has(p)) {
        owners.push(...teams.get(p));
      } else if (p.startsWith("@") && (includeTeams || !p.includes("/"))) {
        owners.push(p.slice(1));
      }
    }
    rules.push({ pattern, owners });
  }
  return rules;
}

/**
 * Match a filepath against an OWNERS pattern.
 * Supports: "*" (catch-all), "/dir/" (prefix), "/path/file" (exact).
 */
function ownersMatch(pattern, filepath) {
  if (pattern === "*") return true;
  let p = pattern;
  if (p.startsWith("/")) p = p.slice(1);
  if (p.endsWith("/")) return filepath.startsWith(p);
  return filepath === p;
}

/**
 * Find owners for a file. Last match wins, like CODEOWNERS.
 * @returns {string[]} owner logins
 */
function findOwners(filepath, rules) {
  let matched = [];
  for (const rule of rules) {
    if (ownersMatch(rule.pattern, filepath)) {
      matched = rule.owners;
    }
  }
  return matched;
}

/**
 * Get maintainers from the * catch-all rule.
 * @returns {string[]} logins
 */
function getMaintainers(rules) {
  const catchAll = rules.find((r) => r.pattern === "*");
  return catchAll ? catchAll.owners : [];
}

/**
 * Group files by their matched OWNERS rule (last-match-wins).
 * Returns Map<pattern, { owners: string[], files: string[] }>
 */
function getOwnershipGroups(filenames, rules) {
  const groups = new Map();
  for (const filepath of filenames) {
    let matchedPattern = null;
    let matchedOwners = [];
    for (const rule of rules) {
      if (ownersMatch(rule.pattern, filepath)) {
        matchedPattern = rule.pattern;
        matchedOwners = rule.owners;
      }
    }
    if (!matchedPattern) continue;
    if (!groups.has(matchedPattern)) {
      groups.set(matchedPattern, { owners: [...matchedOwners], files: [] });
    }
    groups.get(matchedPattern).files.push(filepath);
  }
  return groups;
}

/**
 * Parse OWNERS into raw rules WITHOUT expanding team aliases.
 * Unlike parseOwnersFile, this keeps the original tokens ("team:x", "@user")
 * so the validator can tell defined from undefined teams and count owners.
 *
 * @param {string} filePath - absolute path to the OWNERS file
 * @returns {Array<{pattern: string, tokens: string[]}>}
 */
function parseOwnersRules(filePath) {
  return readDataLines(filePath).map((parts) => ({
    pattern: parts[0],
    tokens: parts.slice(1),
  }));
}

/**
 * Parse the GitHub team-page URLs declared in the OWNERTEAMS header comment.
 * Format: "#   <name>: https://github.com/orgs/databricks/teams/<slug>"
 * Returns the set of "team:<name>" aliases that have a documented page.
 *
 * @param {string} filePath - absolute path to the OWNERTEAMS file
 * @returns {Set<string>}
 */
function parseTeamPageUrls(filePath) {
  let content;
  try {
    content = fs.readFileSync(filePath, "utf-8");
  } catch (e) {
    if (e.code === "ENOENT") return new Set();
    throw e;
  }
  const pages = new Set();
  for (const raw of content.split("\n")) {
    const m = raw.match(/^#\s*([a-z0-9-]+):\s*https?:\/\//);
    if (m) pages.add("team:" + m[1]);
  }
  return pages;
}

/**
 * Validate OWNERS and OWNERTEAMS for internal consistency.
 *
 * Errors (block CI):
 *   - a rule references a "team:" alias not defined in OWNERTEAMS
 *   - a rule resolves to zero owners (only a maintainer could ever approve it)
 *   - a rule maps a path that does not exist in the repository
 *
 * Warnings (reported, non-blocking): a defined team has no team-page URL in the
 * OWNERTEAMS header. A team may legitimately predate its GitHub team page, so
 * this never blocks a merge.
 *
 * fileExists is injectable so tests can validate synthetic rules without a tree.
 *
 * @param {string} ownersPath
 * @param {string} teamsPath
 * @param {{ repoRoot?: string, fileExists?: (p: string) => boolean }} [opts]
 * @returns {{ errors: string[], warnings: string[] }}
 */
function validateOwners(ownersPath, teamsPath, opts) {
  const repoRoot = (opts && opts.repoRoot) || process.cwd();
  const fileExists = (opts && opts.fileExists) || ((p) => fs.existsSync(p));
  const teams = parseOwnerTeams(teamsPath);
  const pageUrls = parseTeamPageUrls(teamsPath);
  const errors = [];
  const warnings = [];

  for (const { pattern, tokens } of parseOwnersRules(ownersPath)) {
    let ownerCount = 0;
    let undefinedTeam = false;
    for (const token of tokens) {
      if (token.startsWith("team:")) {
        if (teams.has(token)) {
          ownerCount += teams.get(token).length;
        } else {
          errors.push(`rule "${pattern}" references undefined team "${token}"; define it in .github/OWNERTEAMS`);
          undefinedTeam = true;
        }
      } else if (token.startsWith("@")) {
        ownerCount += 1;
      }
    }
    // Skip the zero-owner error when an undefined team already explains it.
    if (ownerCount === 0 && !undefinedTeam) {
      errors.push(`rule "${pattern}" resolves to zero owners`);
    }
    if (pattern !== "*") {
      const rel = pattern.replace(/^\//, "").replace(/\/$/, "");
      if (rel && !fileExists(path.join(repoRoot, rel))) {
        errors.push(`rule "${pattern}" maps a path that does not exist in the repo`);
      }
    }
  }

  for (const team of teams.keys()) {
    if (!pageUrls.has(team)) {
      warnings.push(`team "${team}" has no GitHub team-page URL in the .github/OWNERTEAMS header comment`);
    }
  }

  return { errors, warnings };
}

module.exports = {
  parseOwnerTeams,
  parseOwnersFile,
  parseOwnersRules,
  parseTeamPageUrls,
  ownersMatch,
  findOwners,
  getMaintainers,
  getOwnershipGroups,
  validateOwners,
};

// CLI entrypoint: `node .github/scripts/owners.js validate`
if (require.main === module) {
  if (process.argv[2] !== "validate") {
    console.error("usage: node .github/scripts/owners.js validate");
    process.exit(2);
  }
  const root = process.cwd();
  const { errors, warnings } = validateOwners(
    path.join(root, ".github", "OWNERS"),
    path.join(root, ".github", "OWNERTEAMS"),
    { repoRoot: root },
  );
  for (const w of warnings) console.warn(`warning: ${w}`);
  for (const e of errors) console.error(`error: ${e}`);
  if (errors.length > 0) {
    console.error(`OWNERS validation failed: ${errors.length} error(s), ${warnings.length} warning(s)`);
    process.exit(1);
  }
  console.log(`OWNERS validation passed: ${warnings.length} warning(s)`);
}
