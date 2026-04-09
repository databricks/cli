const fs = require("fs");

/**
 * Parse an OWNERS file (same format as CODEOWNERS).
 * Returns array of { pattern, owners } rules.
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
  const lines = fs.readFileSync(filePath, "utf-8").split("\n");
  const rules = [];
  for (const raw of lines) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    const parts = line.split(/\s+/);
    if (parts.length < 2) continue;
    const pattern = parts[0];
    const owners = parts
      .slice(1)
      .filter((p) => p.startsWith("@") && (includeTeams || !p.includes("/")))
      .map((p) => p.slice(1));
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

module.exports = { parseOwnersFile, ownersMatch, findOwners, getMaintainers, getOwnershipGroups };
