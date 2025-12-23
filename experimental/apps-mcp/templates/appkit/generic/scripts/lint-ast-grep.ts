/**
 * AST-based linting using ast-grep.
 * Catches patterns that ESLint/TypeScript miss or handle poorly.
 * Usage: npx tsx scripts/lint-ast-grep.ts
 */

import { parse, Lang } from '@ast-grep/napi';
import fs from 'node:fs';
import path from 'node:path';

interface LintViolation {
  file: string;
  line: number;
  column: number;
  rule: string;
  message: string;
  code: string;
}

interface LintRule {
  id: string;
  pattern: string;
  message: string;
  filter?: (code: string) => boolean;
  includeTests?: boolean; // default true - set false to skip test files
}

const rules: LintRule[] = [
  {
    id: 'no-double-type-assertion',
    pattern: '$X as unknown as $Y',
    message: 'Avoid double type assertion (as unknown as). Use proper type guards or fix the source type.',
  },
  {
    id: 'no-as-any',
    pattern: '$X as any',
    message: 'Avoid "as any" type assertion. Use proper typing or unknown with type guards.',
    includeTests: false, // acceptable in test mocks
  },
  {
    id: 'no-array-index-key',
    pattern: 'key={$IDX}',
    message: 'Avoid using array index as React key. Use a stable unique identifier.',
    filter: (code) => /key=\{(idx|index|i)\}/.test(code),
  },
  {
    id: 'no-parse-float-without-validation',
    pattern: 'parseFloat($X).toFixed($Y)',
    message: 'parseFloat can return NaN. Validate input or use toNumber() helper from shared/types.ts.',
  },
];

function isTestFile(filePath: string): boolean {
  return /\.(test|spec)\.(ts|tsx)$/.test(filePath) || filePath.includes('/tests/');
}

function findTsFiles(dir: string, files: string[] = []): string[] {
  const entries = fs.readdirSync(dir, { withFileTypes: true });

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);

    if (entry.isDirectory()) {
      if (['node_modules', 'dist', 'build', '.git'].includes(entry.name)) continue;
      findTsFiles(fullPath, files);
    } else if (entry.isFile() && /\.(ts|tsx)$/.test(entry.name)) {
      files.push(fullPath);
    }
  }

  return files;
}

function lintFile(filePath: string, rules: LintRule[]): LintViolation[] {
  const violations: LintViolation[] = [];
  const content = fs.readFileSync(filePath, 'utf-8');
  const lang = filePath.endsWith('.tsx') ? Lang.Tsx : Lang.TypeScript;
  const testFile = isTestFile(filePath);

  const ast = parse(lang, content);
  const root = ast.root();

  for (const rule of rules) {
    // skip rules that don't apply to test files
    if (testFile && rule.includeTests === false) continue;

    const matches = root.findAll(rule.pattern);

    for (const match of matches) {
      const code = match.text();

      if (rule.filter && !rule.filter(code)) continue;

      const range = match.range();
      violations.push({
        file: filePath,
        line: range.start.line + 1,
        column: range.start.column + 1,
        rule: rule.id,
        message: rule.message,
        code: code.length > 80 ? code.slice(0, 77) + '...' : code,
      });
    }
  }

  return violations;
}

function main(): void {
  const rootDir = process.cwd();
  const files = findTsFiles(rootDir);

  console.log(`Scanning ${files.length} TypeScript files...\n`);

  const allViolations: LintViolation[] = [];

  for (const file of files) {
    const violations = lintFile(file, rules);
    allViolations.push(...violations);
  }

  if (allViolations.length === 0) {
    console.log('No ast-grep lint violations found.');
    process.exit(0);
  }

  console.log(`Found ${allViolations.length} violation(s):\n`);

  for (const v of allViolations) {
    const relPath = path.relative(rootDir, v.file);
    console.log(`${relPath}:${v.line}:${v.column}`);
    console.log(`  ${v.rule}: ${v.message}`);
    console.log(`  > ${v.code}\n`);
  }

  process.exit(1);
}

main();
