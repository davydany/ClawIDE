/**
 * capture-screenshots.js
 *
 * Playwright script to capture ClawIDE screenshots for the documentation site.
 * Requires ClawIDE running at http://localhost:9800.
 *
 * Usage:
 *   node scripts/capture-screenshots.js
 *
 * Prerequisites:
 *   npm install (installs playwright from package.json)
 *   npx playwright install chromium
 */

const { chromium } = require("playwright");
const path = require("path");
const fs = require("fs");

const BASE_URL = process.env.CLAWIDE_URL || "http://localhost:9800";
const SCREENSHOT_DIR = path.join(__dirname, "..", "static", "images", "screenshots");
const VIEWPORT = { width: 1440, height: 900 };

// How long to wait for network to settle
const WAIT_OPTIONS = { waitUntil: "domcontentloaded", timeout: 30000 };

async function ensureScreenshotDir() {
  if (!fs.existsSync(SCREENSHOT_DIR)) {
    fs.mkdirSync(SCREENSHOT_DIR, { recursive: true });
    console.log(`Created screenshot directory: ${SCREENSHOT_DIR}`);
  }
}

async function capture(page, name, description) {
  const filepath = path.join(SCREENSHOT_DIR, `${name}.png`);
  await page.screenshot({ path: filepath, fullPage: false });
  console.log(`  ✓ ${name}.png – ${description}`);
}

async function main() {
  console.log(`\nClawIDE Screenshot Capture`);
  console.log(`==========================`);
  console.log(`Target: ${BASE_URL}`);
  console.log(`Output: ${SCREENSHOT_DIR}\n`);

  await ensureScreenshotDir();

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: VIEWPORT });
  const page = await context.newPage();

  try {
    console.log("Capturing screenshots...\n");

    // ---- 1. Welcome / Onboarding ----
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(2000);
    await capture(page, "onboarding-welcome", "Welcome / onboarding screen");

    // ---- 2. Dashboard ----
    // Main dashboard view with projects list
    await page.waitForTimeout(500);
    await capture(page, "dashboard", "Project dashboard with projects list");

    // ---- 3. Settings Page ----
    await page.goto(`${BASE_URL}/settings`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);
    await capture(page, "settings", "Settings page");

    // ---- 4. Starred Projects ----
    // Back to dashboard and focus on starred projects
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);

    // Find and click on a starred project
    const starredProject = await page.$(".project-card:has(svg[class*='star'])");
    if (starredProject) {
      await starredProject.click();
      await page.waitForTimeout(2000);
      await capture(page, "project-workspace", "Project workspace view");

      // Go back to dashboard
      await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
      await page.waitForTimeout(1000);
    }

    // ---- 5. All Projects View ----
    // Scroll down to see the full projects list
    await page.evaluate(() => window.scrollBy(0, 300));
    await page.waitForTimeout(500);
    await capture(page, "all-projects", "All projects view");

    console.log(`\nDone! ${fs.readdirSync(SCREENSHOT_DIR).filter(f => f.endsWith(".png")).length} screenshots saved to:`);
    console.log(`  ${SCREENSHOT_DIR}\n`);

  } catch (err) {
    console.error(`\nError during screenshot capture: ${err.message}`);
    console.error(err.stack);
    console.error("Make sure ClawIDE is running at", BASE_URL);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

main();
