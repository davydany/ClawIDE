---
title: "Themes"
description: "Switch between light and dark mode with system preference detection."
weight: 55
---

ClawIDE supports light and dark color themes, allowing you to work comfortably in any lighting environment.

## Switching Themes

Use the theme toggle in the top bar to switch between light and dark mode. Your preference is saved and persists across browser sessions.

## System Preference Detection

On first launch, ClawIDE detects your operating system's color scheme preference via `prefers-color-scheme` and applies the matching theme automatically. You can override this at any time using the manual toggle.

## How It Works

The theme is applied by setting a `data-theme` attribute on the root HTML element. All UI surfaces — terminals, the file editor, sidebar, cards, and toolbars — respond to the active theme. The preference is stored in `localStorage` so it survives page reloads and restarts.
