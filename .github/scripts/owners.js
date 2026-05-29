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

module.exports = { parseOwnerTeams, parseOwnersFile, ownersMatch, findOwners, getMaintainers, getOwnershipGroups };
