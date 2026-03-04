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

async function main() {
  console.log(`\nClawIDE Screenshot Capture`);
  console.log(`==========================`);
  console.log(`Target: ${BASE_URL}`);
  console.log(`Output: ${SCREENSHOT_DIR}\n`);

  await ensureScreenshotDir();

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: VIEWPORT });
  const page = await context.newPage();

  let projectID = null;
  let featureID = null;

  try {
    console.log("Capturing screenshots...\n");

    // ---- 1. Dashboard ----
    await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
    await page.waitForTimeout(2000);
    await capture(page, "onboarding-welcome", "Welcome / onboarding screen");
    await capture(page, "dashboard", "Project dashboard with projects list");

    // Find the first project link to get its ID
    const projectLink = await page.$("a[href*='/projects/']");
    if (projectLink) {
      const href = await projectLink.getAttribute("href");
      const match = href.match(/\/projects\/([^/]+)/);
      if (match) projectID = match[1];
    }

    // ---- 2. Settings Page ----
    await page.goto(`${BASE_URL}/settings`, WAIT_OPTIONS);
    await page.waitForTimeout(1000);
    await capture(page, "settings", "Settings page – General and AI Agent config");

    // ---- 3. Auto-Update Section (scroll to bottom of settings) ----
    // Find auto-update section by scrolling to the bottom
    const autoUpdateVisible = await page.evaluate(() => {
      const el = document.querySelector('[x-data*="autoUpdate"]') ||
                 Array.from(document.querySelectorAll('h2, h3, .font-semibold')).find(e => e.textContent.includes('Auto'));
      if (el) { el.scrollIntoView({ block: 'center' }); return true; }
      window.scrollTo(0, document.body.scrollHeight);
      return false;
    });
    await page.waitForTimeout(500);
    await capture(page, "auto-update", "Auto-update settings section");

    // ---- 4. Claude Hooks Section ----
    const claudeHooksVisible = await page.evaluate(() => {
      const el = Array.from(document.querySelectorAll('h2, h3, .font-semibold')).find(e =>
        e.textContent.includes('Claude') || e.textContent.includes('Hook')
      );
      if (el) { el.scrollIntoView({ block: 'center' }); return true; }
      return false;
    });
    await page.waitForTimeout(500);
    await capture(page, "claude-hooks", "Claude Code hooks settings section");

    // ---- 5. Project Wizard ----
    // Try direct route first, fall back to clicking New Project button on dashboard
    const wizardResponse = await page.goto(`${BASE_URL}/projects/wizard`, WAIT_OPTIONS);
    if (wizardResponse && wizardResponse.status() === 200) {
      await page.waitForTimeout(1500);
      await capture(page, "project-wizard", "Project wizard with templates and LLM generation");
    } else {
      // Fall back: go to dashboard and click New Project
      await page.goto(`${BASE_URL}/`, WAIT_OPTIONS);
      await page.waitForTimeout(1000);
      try {
        const newProjectBtn = await page.$("button:has-text('New Project'), a:has-text('New Project'), [data-tour='new-project']");
        if (newProjectBtn && await newProjectBtn.isVisible()) {
          await newProjectBtn.click();
          await page.waitForTimeout(2000);
          await capture(page, "project-wizard", "Project wizard modal");
        }
      } catch { console.log("  ℹ Project wizard not accessible"); }
    }

    // ---- 6. Open a Project Workspace ----
    if (projectID) {
      await page.goto(`${BASE_URL}/projects/${projectID}/`, WAIT_OPTIONS);
      await page.waitForTimeout(3000);

      // ---- 7. Terminal Sessions ----
      await capture(page, "terminal-sessions", "Terminal sessions in project workspace");

      // ---- 8. File Editor ----
      const filesTab = await page.$("button:has-text('Files'), [role='tab']:has-text('Files')");
      if (filesTab) {
        await filesTab.click();
        await page.waitForTimeout(1500);
        await capture(page, "file-editor", "File editor with directory tree and code editor");
      }

      // ---- 9. Docker Integration ----
      const dockerTab = await page.$("button:has-text('Docker'), [role='tab']:has-text('Docker')");
      if (dockerTab) {
        await dockerTab.click();
        await page.waitForTimeout(1500);
        await capture(page, "docker-integration", "Docker integration with healthchecks and inline logs");
      }

      // ---- 10. Port Detection ----
      try {
        const portsTab = await page.$("button:has-text('Ports'), [role='tab']:has-text('Ports')");
        if (portsTab && await portsTab.isVisible()) {
          await portsTab.click();
          await page.waitForTimeout(1500);
          await capture(page, "port-detection", "Port detection view");
        }
      } catch { console.log("  ℹ Ports tab not accessible — skipping port-detection"); }

      // ---- 11. Git Worktrees ----
      try {
        const gitBtn = await page.$("button:has-text('Git'), [data-testid*='git'], a:has-text('Git')");
        if (gitBtn && await gitBtn.isVisible()) {
          await gitBtn.click();
          await page.waitForTimeout(1500);
          await capture(page, "git-worktrees", "Git worktrees and branch management");
        }
      } catch { console.log("  ℹ Git tab not accessible — skipping git-worktrees"); }

      // ---- 12. Code Snippets sidebar section ----
      try {
        const snippetsHeader = await page.$("text=SNIPPETS");
        if (snippetsHeader && await snippetsHeader.isVisible()) {
          await snippetsHeader.click();
          await page.waitForTimeout(1000);
          await capture(page, "code-snippets", "Code snippets sidebar section");
        }
      } catch { console.log("  ℹ Snippets section not accessible"); }

      // ---- 13. Bookmarks sidebar section ----
      try {
        const bookmarksHeader = await page.$("text=BOOKMARKS");
        if (bookmarksHeader && await bookmarksHeader.isVisible()) {
          await bookmarksHeader.click();
          await page.waitForTimeout(1000);
          await capture(page, "bookmarks", "Bookmarks sidebar section with bookmarks bar");
        }
      } catch { console.log("  ℹ Bookmarks section not accessible"); }

      // ---- 14. Notes sidebar section ----
      try {
        const notesHeader = await page.$("text=NOTES");
        if (notesHeader && await notesHeader.isVisible()) {
          await notesHeader.click();
          await page.waitForTimeout(1000);
          await capture(page, "notes", "Notes sidebar section with folders");
        }
      } catch { console.log("  ℹ Notes section not accessible"); }

      // ---- 15. Scratchpad sidebar section ----
      try {
        const scratchpadHeader = await page.$("text=SCRATCHPAD");
        if (scratchpadHeader && await scratchpadHeader.isVisible()) {
          await scratchpadHeader.click();
          await page.waitForTimeout(500);
        } else {
          const scratchpad = await page.$("#scratchpad-content");
          if (scratchpad) {
            await scratchpad.scrollIntoViewIfNeeded();
            await page.waitForTimeout(500);
          }
        }
        await capture(page, "scratchpad", "Scratchpad with auto-save");
      } catch { console.log("  ℹ Scratchpad not accessible"); }

      // ---- 16. Notifications panel ----
      try {
        const bellBtn = await page.$("[x-data*='notificationBell'] button, button[aria-label*='otification']");
        if (bellBtn && await bellBtn.isVisible()) {
          await bellBtn.click();
          await page.waitForTimeout(1000);
          await capture(page, "notifications", "Notifications panel");
          await bellBtn.click();
          await page.waitForTimeout(300);
        }
      } catch { console.log("  ℹ Notifications bell not accessible"); }

      // ---- 17. VoiceBox ----
      try {
        const voiceBtn = await page.$("[data-action='voicebox'], button[aria-label*='Voice']");
        if (voiceBtn && await voiceBtn.isVisible()) {
          await voiceBtn.click();
          await page.waitForTimeout(1000);
          await capture(page, "voicebox", "VoiceBox modal for voice memos");
          const closeBtn = await page.$("#voicebox-modal button:has-text('Close'), #voicebox-modal [aria-label='Close']");
          if (closeBtn) await closeBtn.click();
          await page.waitForTimeout(300);
        }
      } catch { console.log("  ℹ VoiceBox not accessible"); }

      // ---- 18. System Stats sidebar ----
      await capture(page, "system-stats", "System statistics in sidebar");

      // ---- 19. Feature Workspaces ----
      // First check if a feature exists, or try to find one
      const featureLink = await page.$("a[href*='/features/']");
      if (featureLink) {
        const fhref = await featureLink.getAttribute("href");
        await page.goto(`${BASE_URL}${fhref}`, WAIT_OPTIONS);
        await page.waitForTimeout(2000);
        await capture(page, "feature-workspaces", "Feature workspace with isolated sessions and files");

        // ---- 20. Merge Review tab ----
        try {
          // Use page.evaluate to click the Review tab via Alpine.js or direct DOM
          await page.evaluate(() => {
            // Find the Review tab button by text content
            const buttons = Array.from(document.querySelectorAll('button'));
            const reviewBtn = buttons.find(b => b.textContent.trim() === 'Review');
            if (reviewBtn) {
              reviewBtn.click();
              return;
            }
            // Fallback: set Alpine.js state directly
            const xDataEl = document.querySelector('[x-data]');
            if (xDataEl && xDataEl._x_dataStack) {
              xDataEl._x_dataStack[0].activeTab = 'review';
            }
          });
          await page.waitForTimeout(2000);
          await capture(page, "merge-review", "Merge review with side-by-side diff viewer");
        } catch { console.log("  ℹ Review tab not accessible — skipping merge-review"); }
      } else {
        console.log("  ℹ No feature workspace found — skipping feature-workspaces and merge-review screenshots");
        console.log("    Create a feature in a project first, then re-run this script");
      }
    } else {
      console.log("  ℹ No projects found — skipping project workspace screenshots");
      console.log("    Run scripts/setup-demo-project.sh first, then re-run this script");
    }

    const count = fs.readdirSync(SCREENSHOT_DIR).filter(f => f.endsWith(".png")).length;
    console.log(`\nDone! ${count} screenshots saved to:`);
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
