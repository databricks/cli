import { test, expect } from '@playwright/test';
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';

test('smoke test - app loads and displays data', async ({ page }) => {
  // Create temp directory for test artifacts
  const tempDir = join(process.cwd(), '.smoke-test');
  mkdirSync(tempDir, { recursive: true });

  // Capture console logs and errors (including React errors)
  const consoleLogs: string[] = [];
  const consoleErrors: string[] = [];
  page.on('console', (msg) => {
    const type = msg.type();
    const text = msg.text();
    consoleLogs.push(`[${type}] ${text}`);

    // Separately track error messages (React errors appear here)
    if (type === 'error') {
      consoleErrors.push(text);
    }
  });

  // Capture page errors
  const pageErrors: string[] = [];
  page.on('pageerror', (error) => {
    pageErrors.push(`Page error: ${error.message}`);
  });

  // Capture failed requests
  const failedRequests: string[] = [];
  page.on('requestfailed', (request) => {
    failedRequests.push(`Failed request: ${request.url()} - ${request.failure()?.errorText}`);
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

    // Verify console logs were captured
    expect(consoleLogs.length).toBeGreaterThan(0);
    expect(consoleErrors.length).toBe(0);
    expect(pageErrors.length).toBe(0);
  } finally {
    // Always capture artifacts, even if test fails
    const screenshotPath = join(tempDir, 'app-loaded.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });

    const logsPath = join(tempDir, 'console-logs.txt');
    const allLogs = [
      '=== Console Logs ===',
      ...consoleLogs,
      '\n=== Console Errors (React errors) ===',
      ...consoleErrors,
      '\n=== Page Errors ===',
      ...pageErrors,
      '\n=== Failed Requests ===',
      ...failedRequests,
    ];
    writeFileSync(logsPath, allLogs.join('\n'), 'utf-8');

    console.log(`Screenshot saved to: ${screenshotPath}`);
    console.log(`Console logs saved to: ${logsPath}`);
    if (consoleErrors.length > 0) {
      console.log('Console errors detected:', consoleErrors);
    }
    if (pageErrors.length > 0) {
      console.log('Page errors detected:', pageErrors);
    }
    if (failedRequests.length > 0) {
      console.log('Failed requests detected:', failedRequests);
    }
  }
});
