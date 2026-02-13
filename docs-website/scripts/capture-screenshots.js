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
const WAIT_OPTIONS = { waitUntil: "networkidle", timeout: 30000 };

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

async function waitForStable(page, selector, timeout = 10000) {
  try {
    await page.waitForSelector(selector, { state: "visible", timeout });
  } catch {
    // If the selector doesn't appear, continue anyway — the page may have loaded differently
    console.log(`  ⚠ Selector "${selector}" not found, capturing current state`);
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
    // ---- 1. Onboarding Welcome ----
    console.log("Capturing screenshots...\n");

    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);
    await capture(page, "onboarding-welcome", "Welcome / onboarding screen");

    // ---- 2. Dashboard ----
    // After onboarding, navigate to the main dashboard
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await waitForStable(page, "[data-testid='dashboard'], .dashboard, main");
    await page.waitForTimeout(500);
    await capture(page, "dashboard", "Project dashboard");

    // ---- 3. Terminal Sessions ----
    // Navigate to a project workspace and capture the terminal tab
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await waitForStable(page, "[data-testid='terminal'], .terminal-container, .xterm");
    await page.waitForTimeout(500);
    await capture(page, "terminal-sessions", "Terminal sessions tab");

    // ---- 4. Terminal Split Panes ----
    // Look for a split pane button and click it
    const splitBtn = await page.$("[data-testid='split-pane'], [title*='split' i], button:has-text('Split')");
    if (splitBtn) {
      await splitBtn.click();
      await page.waitForTimeout(1000);
    }
    await capture(page, "terminal-split-panes", "Terminal with split panes");

    // ---- 5. File Editor ----
    // Navigate to the file editor tab
    const fileTab = await page.$("[data-testid='file-editor-tab'], [href*='editor'], a:has-text('Files'), a:has-text('Editor')");
    if (fileTab) {
      await fileTab.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='file-editor'], .editor-container, .CodeMirror, .monaco-editor");
    await capture(page, "file-editor", "File editor tab");

    // ---- 6. Docker Integration ----
    const dockerTab = await page.$("[data-testid='docker-tab'], [href*='docker'], a:has-text('Docker')");
    if (dockerTab) {
      await dockerTab.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='docker'], .docker-container");
    await capture(page, "docker-integration", "Docker integration tab");

    // ---- 7. Git Worktrees ----
    const gitTab = await page.$("[data-testid='git-tab'], [href*='git'], a:has-text('Git')");
    if (gitTab) {
      await gitTab.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='git'], .git-container");
    await capture(page, "git-worktrees", "Git worktrees tab");

    // ---- 8. Feature Workspaces ----
    const workspaceLink = await page.$("[data-testid='workspace'], [href*='workspace'], a:has-text('Workspace')");
    if (workspaceLink) {
      await workspaceLink.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='workspace'], .workspace-container");
    await capture(page, "feature-workspaces", "Feature workspace view");

    // ---- 9. Port Detection ----
    const portsTab = await page.$("[data-testid='ports-tab'], [href*='port'], a:has-text('Ports')");
    if (portsTab) {
      await portsTab.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='ports'], .ports-container");
    await capture(page, "port-detection", "Port detection tab");

    // ---- 10. Settings ----
    await page.goto(`${BASE_URL}/settings`, WAIT_OPTIONS);
    await page.waitForTimeout(500);
    await capture(page, "settings", "Settings page");

    // ---- 11. Code Snippets ----
    // Navigate back and open the snippet drawer
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    const snippetBtn = await page.$("[data-testid='snippet-drawer'], [title*='snippet' i], button:has-text('Snippet')");
    if (snippetBtn) {
      await snippetBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='snippets'], .snippet-drawer, .snippets-panel");
    await capture(page, "code-snippets", "Code snippets drawer");

    // ---- 12. Bookmarks ----
    // Navigate to a project and open the bookmarks panel
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    const bookmarksBtn = await page.$("[data-testid='bookmarks'], [title*='bookmark' i], button:has-text('Bookmark'), a:has-text('Bookmark')");
    if (bookmarksBtn) {
      await bookmarksBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='bookmarks-panel'], .bookmarks-panel, .bookmarks-container");
    await capture(page, "bookmarks", "Bookmarks panel");

    // ---- 13. Notes ----
    const notesBtn = await page.$("[data-testid='notes'], [title*='note' i], button:has-text('Note'), a:has-text('Note')");
    if (notesBtn) {
      await notesBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='notes-panel'], .notes-panel, .notes-container");
    await capture(page, "notes", "Notes panel");

    // ---- 14. Notifications ----
    const notifBtn = await page.$("[data-testid='notifications'], [title*='notification' i], button:has-text('Notification'), .notification-bell");
    if (notifBtn) {
      await notifBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='notifications-panel'], .notifications-panel, .notifications-container");
    await capture(page, "notifications", "Notification center");

    // ---- 15. System Statistics ----
    const statsBtn = await page.$("[data-testid='system-stats'], [title*='stats' i], button:has-text('Stats'), a:has-text('Stats'), a:has-text('System')");
    if (statsBtn) {
      await statsBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='system-stats-panel'], .system-stats, .stats-container");
    await capture(page, "system-stats", "System statistics view");

    // ---- 16. Auto-Update (Settings page) ----
    await page.goto(`${BASE_URL}/settings`, WAIT_OPTIONS);
    await page.waitForTimeout(500);
    // Scroll to the update section if it exists
    const updateSection = await page.$("[data-testid='update-section'], .update-section, #update-settings");
    if (updateSection) {
      await updateSection.scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
    }
    await capture(page, "auto-update", "Auto-update section in settings");

    // ---- 17. Claude Code Hooks (Settings page) ----
    const claudeSection = await page.$("[data-testid='claude-section'], .claude-section, #claude-settings");
    if (claudeSection) {
      await claudeSection.scrollIntoViewIfNeeded();
      await page.waitForTimeout(500);
    }
    await capture(page, "claude-hooks", "Claude Code hooks section in settings");

    // ---- 18. VoiceBox ----
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    const voiceBtn = await page.$("[data-testid='voicebox'], [title*='voice' i], button:has-text('Voice'), a:has-text('Voice')");
    if (voiceBtn) {
      await voiceBtn.click();
      await page.waitForTimeout(1000);
    }
    await waitForStable(page, "[data-testid='voicebox-panel'], .voicebox-panel, .voicebox-container");
    await capture(page, "voicebox", "VoiceBox panel");

    console.log(`\nDone! ${fs.readdirSync(SCREENSHOT_DIR).filter(f => f.endsWith(".png")).length} screenshots saved to:`);
    console.log(`  ${SCREENSHOT_DIR}\n`);

  } catch (err) {
    console.error(`\nError during screenshot capture: ${err.message}`);
    console.error("Make sure ClawIDE is running at", BASE_URL);
    process.exit(1);
  } finally {
    await browser.close();
  }
}

main();
