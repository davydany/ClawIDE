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
  try {
    await page.screenshot({ path: filepath, fullPage: false });
    console.log(`  ✓ ${name}.png – ${description}`);
    return true;
  } catch (err) {
    console.log(`  ⚠ ${name}.png – Failed: ${err.message}`);
    return false;
  }
}

async function waitForElement(page, selector, timeout = 5000) {
  try {
    await page.waitForSelector(selector, { state: "visible", timeout });
    return true;
  } catch {
    return false;
  }
}

async function clickAndWait(page, selector, waitSelector = null, timeout = 2000) {
  try {
    const element = await page.$(selector);
    if (!element) return false;

    await element.click();
    await page.waitForTimeout(timeout);

    if (waitSelector) {
      return await waitForElement(page, waitSelector, 3000);
    }
    return true;
  } catch {
    return false;
  }
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

    // ---- 1. Onboarding Welcome ----
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(2000);
    await capture(page, "onboarding-welcome", "Welcome / onboarding screen");

    // ---- 2. Dashboard ----
    await page.waitForTimeout(500);
    await capture(page, "dashboard", "Project dashboard with projects list");

    // ---- 3. Settings Page ----
    await page.goto(`${BASE_URL}/settings`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);
    await capture(page, "settings", "Settings page");

    // Capture auto-update section (scroll down on settings)
    await page.evaluate(() => window.scrollBy(0, 500));
    await page.waitForTimeout(500);
    await capture(page, "auto-update", "Auto-update settings section");

    // Capture Claude hooks section
    await page.evaluate(() => window.scrollBy(0, 300));
    await page.waitForTimeout(500);
    await capture(page, "claude-hooks", "Claude Code hooks section");

    // ---- 4. Open a Project Workspace ----
    // Navigate back to dashboard
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);

    // Find and click the first project card
    const firstProject = await page.$(".project-card, [data-testid='project-card'], a[href*='/projects/']");
    if (firstProject) {
      await firstProject.click();
      await page.waitForTimeout(3000);

      // ---- 5. Terminal Sessions ----
      // Default view should show terminal
      await capture(page, "terminal-sessions", "Terminal sessions in project workspace");

      // ---- 6. File Editor ----
      // Click on Files tab
      const filesTab = await page.$("button:has-text('Files'), [role='tab']:has-text('Files')");
      if (filesTab) {
        await filesTab.click();
        await page.waitForTimeout(1500);
        await capture(page, "file-editor", "File editor view");
      }

      // ---- 7. Docker Integration ----
      const dockerTab = await page.$("button:has-text('Docker'), [role='tab']:has-text('Docker')");
      if (dockerTab) {
        await dockerTab.click();
        await page.waitForTimeout(1500);
        await capture(page, "docker-integration", "Docker integration view");
      }

      // ---- 8. Port Detection ----
      const portsTab = await page.$("button:has-text('Ports'), [role='tab']:has-text('Ports')");
      if (portsTab) {
        await portsTab.click();
        await page.waitForTimeout(1500);
        await capture(page, "port-detection", "Port detection view");
      }

      // ---- 9. Terminal Split Panes ----
      // Go back to terminal tab to show split panes
      const terminalTab = await page.$("button:has-text('Terminal'), [role='tab']:has-text('Terminal')");
      if (terminalTab) {
        await terminalTab.click();
        await page.waitForTimeout(1000);

        // Look for split pane button
        const splitBtn = await page.$("button[title*='split'], button[aria-label*='split']");
        if (splitBtn) {
          await splitBtn.click();
          await page.waitForTimeout(1000);
          await capture(page, "terminal-split-panes", "Terminal with split panes");
        }
      }

      // ---- 10. Code Snippets - Expand sidebar section ----
      const snippetsSection = await page.$("[role='heading']:has-text('SNIPPETS'), button:has-text('SNIPPETS')");
      if (snippetsSection) {
        await snippetsSection.click();
        await page.waitForTimeout(1000);
        await capture(page, "code-snippets", "Code snippets sidebar section");
      }

      // ---- 11. Bookmarks - Expand sidebar section ----
      const bookmarksSection = await page.$("[role='heading']:has-text('BOOKMARKS'), button:has-text('BOOKMARKS')");
      if (bookmarksSection) {
        await bookmarksSection.click();
        await page.waitForTimeout(1000);
        await capture(page, "bookmarks", "Bookmarks sidebar section");
      }

      // ---- 12. Notes - Expand sidebar section ----
      const notesSection = await page.$("[role='heading']:has-text('NOTES'), button:has-text('NOTES')");
      if (notesSection) {
        await notesSection.click();
        await page.waitForTimeout(1000);
        await capture(page, "notes", "Notes sidebar section");
      }

      // ---- 13. Notifications - Top right icon ----
      const notificationsIcon = await page.$("button[aria-label*='notification'], [data-testid='notifications']");
      if (notificationsIcon) {
        await notificationsIcon.click();
        await page.waitForTimeout(1000);
        await capture(page, "notifications", "Notifications panel");
      }

      // ---- 14. System Statistics - Sidebar ----
      // Just capture the current sidebar which shows system stats
      await capture(page, "system-stats", "System statistics sidebar");

      // ---- 15. Git Worktrees - Look for git UI ----
      // Git might be in the project or as a tab
      const gitBtn = await page.$("button:has-text('Git'), [data-testid*='git'], a:has-text('Git')");
      if (gitBtn) {
        await gitBtn.click();
        await page.waitForTimeout(1000);
        await capture(page, "git-worktrees", "Git worktrees view");
      }

      // ---- 16. Feature Workspaces ----
      // Workspace might be in a menu or special view
      const workspaceBtn = await page.$("button:has-text('Workspace'), [data-testid='workspace']");
      if (workspaceBtn) {
        await workspaceBtn.click();
        await page.waitForTimeout(1000);
        await capture(page, "feature-workspaces", "Feature workspaces view");
      }

      // ---- 17. VoiceBox ----
      const voiceIcon = await page.$("button[aria-label*='voice'], button[title*='voice']");
      if (voiceIcon) {
        await voiceIcon.click();
        await page.waitForTimeout(1000);
        await capture(page, "voicebox", "VoiceBox panel");
      }
    }

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
