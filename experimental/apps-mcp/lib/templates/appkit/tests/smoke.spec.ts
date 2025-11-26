import { test, expect } from '@playwright/test';
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';

test('smoke test - app loads and displays data', async ({ page }) => {
  // Create temp directory for test artifacts
  const tempDir = join(process.cwd(), '.smoke-test');
  mkdirSync(tempDir, { recursive: true });

  // Capture console logs
  const consoleLogs: string[] = [];
  page.on('console', (msg) => {
    const type = msg.type();
    const text = msg.text();
    consoleLogs.push(`[${type}] ${text}`);
  });

  try {
    // Navigate to the app
    await page.goto('/');

    // Wait for the page title to be visible
    await expect(page.getByRole('heading', { name: 'Minimal Databricks App' })).toBeVisible();

    // Wait for SQL query result to load (wait for "hello world" to appear)
    await expect(page.getByText('hello world', { exact: true })).toBeVisible({ timeout: 30000 });

    // Wait for health check to complete (wait for "OK" status)
    await expect(page.getByText('OK')).toBeVisible({ timeout: 30000 });

    // Verify both data sections are present
    expect(consoleLogs.length).toBeGreaterThan(0);
  } finally {
    // Always capture artifacts, even if test fails
    const screenshotPath = join(tempDir, 'app-loaded.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });

    const logsPath = join(tempDir, 'console-logs.txt');
    writeFileSync(logsPath, consoleLogs.join('\n'), 'utf-8');

    console.log(`Screenshot saved to: ${screenshotPath}`);
    console.log(`Console logs saved to: ${logsPath}`);
  }
});
