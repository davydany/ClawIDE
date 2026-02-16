# Mobile-Optimized Editor Enhancement - Implementation Specification

## Overview
Implementing three mobile-friendly features for the ClawIDE editor:
1. **Word Wrap Toggle** (Alt+Z) - Dynamic text wrapping in CodeMirror editor
2. **Sidebar Collapse** (Cmd+B) - Collapsible file tree with mobile-first design
3. **Command Palette** (Cmd+K / Cmd+Shift+P) - Searchable command interface with keyboard + touch support

## Architecture & Patterns

### State Management
- **localStorage** (immediate): `editor.preferences.*` (word wrap, sidebar states)
- **API persistence** (debounced 500ms): POST to `/projects/{id}/api/preferences`
- **Hybrid approach**: Local changes happen immediately, API sync in background

### CodeMirror 6 Integration
- Use **Compartment** pattern for dynamic extensions (see `langCompartment` in codemirror-entry.js:110)
- Store `view._clawIDEWrapCompartment` on editor view for runtime reconfiguration
- Dispatch changes with `view.dispatch({ effects: compartment.reconfigure(...) })`

### Alpine.js Components
- Use existing patterns: `x-data`, `x-show`, `x-model`, `@click`, `x-effect`
- Recent commands tracked in localStorage (max 5)
- Fuzzy search with real-time filtering

### Keyboard Shortcuts
- Existing pattern at editor.js:827-851 (Cmd+S, Cmd+W)
- Add new shortcuts: Cmd+B (sidebar), Alt+Z (word wrap), Cmd+K (palette)
- Detect platform: `(e.metaKey || e.ctrlKey)` for Cmd/Ctrl

## Technology Stack
- CodeMirror 6: EditorView, Compartment, state dispatch
- Alpine.js: Already loaded, used throughout workspace.html
- Tailwind CSS: Existing mobile-first patterns
- Vanilla JavaScript: IIFE modules for sidebar.js, command-palette.js

## Critical Files
- `/web/src/codemirror-entry.js` - Editor creation with compartments
- `/web/static/js/editor.js` - Keyboard shortcuts and pane management
- `/web/static/js/sidebar.js` - Resize logic with debounced persistence
- `/web/templates/pages/workspace.html` - Alpine components and UI
- **NEW** `/web/static/js/editor-commands.js` - Command handlers (200 lines)
- **NEW** `/web/static/js/command-palette.js` - Alpine component (350 lines)

## Implementation Order
1. Word Wrap (Compartment setup + toggle)
2. Sidebar Collapse (CSS + state management)
3. Command Palette UI (Alpine component + search)
4. Text Commands (25+ manipulation handlers)
5. Polish & Testing (accessibility, mobile, cross-browser)

## Success Criteria
✓ All keyboard shortcuts functional (tested in DevTools console)
✓ localStorage preferences persist across page reloads
✓ Mobile FAB button present and functional
✓ Command palette closes on Esc or after command execution
✓ All 25 text commands execute correctly
✓ Touch targets ≥ 48px on mobile
✓ No console errors or warnings
✓ API preferences persist (check network tab)

## Notes
- Debounce API saves to prevent excessive requests (500ms)
- Mobile state separate from desktop state (use matchMedia for breakpoint)
- All commands exposed via `window.ClawIDECommands` for testing
- All UI exposed via `window.ClawIDEPalette` for debugging
- FAB button hidden on desktop (`md:hidden` Tailwind class)
