# Mobile-Optimized Editor - Implementation Tasks

## Phase 1: Word Wrap Foundation (2-3 hours)
- [X] **1.1** Modify `codemirror-entry.js`: Import Compartment and add wrapCompartment
- [X] **1.2** Load wordWrap preference from localStorage (default: true)
- [X] **1.3** Add EditorView.lineWrapping extension via compartment
- [X] **1.4** Store compartment reference on view (`view._clawIDEWrapCompartment`)
- [X] **1.5** Create `toggleWordWrap(view)` function with compartment reconfiguration
- [X] **1.6** Create `getWordWrapState(view)` function for status querying
- [X] **1.7** Expose functions via `window.ClawIDECodeMirror`
- [X] **1.8** Create `editor-commands.js` with debounced API save pattern
- [X] **1.9** Add `toggleWordWrap` command handler in editor-commands.js
- [X] **1.10** Modify `editor.js`: Add Alt+Z keyboard shortcut
- [X] **1.11** Expose `getFocusedPaneId()` and `getActiveTab()` via window.ClawIDEEditor
- [X] **1.12** Test: Alt+Z toggles word wrap, persists on reload, API call is debounced

## Phase 2: Sidebar Collapse (1-2 hours)
- [X] **2.1** Modify `sidebar.js`: Add sidebarCollapsed state variable
- [X] **2.2** Detect mobile breakpoint: `window.matchMedia('(max-width: 768px)').matches`
- [X] **2.3** Load separate states: `sidebarCollapsed` (desktop) and `sidebarCollapsedMobile`
- [X] **2.4** Implement `toggleSidebarCollapse()` function
- [X] **2.5** Apply CSS classes for collapsed state (40px width)
- [X] **2.6** Add smooth CSS transition (300ms ease-in-out)
- [X] **2.7** Save state to localStorage with debounced API call
- [X] **2.8** Expose `window.ClawIDESidebar.toggleCollapse()`
- [X] **2.9** Modify `workspace.html`: Add toggle button in sidebar header with icon
- [X] **2.10** Add CSS: `.sidebar.collapsed { width: 40px; overflow: hidden; transition: width 300ms }`
- [X] **2.11** Modify `editor.js`: Add Cmd+B keyboard shortcut
- [X] **2.12** Test: Button click toggles sidebar, Cmd+B works, state persists, mobile/desktop separate

## Phase 3: Command Palette UI (3-4 hours)
- [X] **3.1** Create `command-palette.js`: Alpine.js component structure
- [X] **3.2** Add fuzzy search implementation (exact → starts with → contains → fuzzy)
- [X] **3.3** Implement keyboard event listeners (Cmd+K, Cmd+Shift+P, Esc)
- [X] **3.4** Add keyboard navigation (↑↓ keys for selection, Enter to execute)
- [X] **3.5** Track recent commands in localStorage (max 5)
- [X] **3.6** Load recent commands on palette open (show at top if no search)
- [X] **3.7** Expose command execution via `window.ClawIDEPalette`
- [X] **3.8** Modify `workspace.html`: Add modal structure with Alpine directives
- [X] **3.9** Add command list rendering with keyboard navigation highlight
- [X] **3.10** Add mobile FAB button (hidden on desktop with md:hidden)
- [X] **3.11** Add search input with auto-focus on modal open
- [X] **3.12** Implement palette close on Esc or after command execution
- [X] **3.13** Test: Cmd+K opens, search filters, navigation works, recent commands show, FAB visible on mobile

## Phase 4: Text Commands (2-3 hours)
- [X] **4.1** Create command registry in command-palette.js (25+ commands)
- [X] **4.2** Organize commands by category (Editor, Text Transformation, Navigation, Selection, Utility)
- [X] **4.3** Implement `sortLinesAscending()` - Split, sort, rejoin
- [X] **4.4** Implement `sortLinesDescending()` - Sort reversed
- [X] **4.5** Implement `transformToUppercase()` - .toUpperCase()
- [X] **4.6** Implement `transformToLowercase()` - .toLowerCase()
- [X] **4.7** Implement `transformToTitleCase()` - Regex pattern
- [X] **4.8** Implement `trimTrailingWhitespace()` - .replace(/[ \t]+$/gm, '')
- [X] **4.9** Implement `deleteEmptyLines()` - Filter empty lines
- [X] **4.10** Implement `duplicateLine()` - Insert '\n' + lineText
- [X] **4.11** Implement `deleteLine()` - Remove line text
- [X] **4.12** Implement `goToLine()` - Prompt + state.doc.line()
- [X] **4.13** Implement `copyFilePath()` - navigator.clipboard.writeText()
- [X] **4.14** Implement `copyRelativePath()` - Relative to project root
- [X] **4.15** Implement comment toggle, indent, outdent commands
- [X] **4.16** Add Heroicons to all commands (visual indicators)
- [X] **4.17** Link commands in palette to handlers in window.ClawIDECommands
- [X] **4.18** Test each command: Select text → Execute → Verify result

## Phase 5: Polish & Testing (2 hours) ✓ COMPLETE
- [X] **5.1** Mobile optimizations: iOS safe area handling for FAB
- [X] **5.2** Verify touch targets are ≥ 48px
- [X] **5.3** Test virtual keyboard doesn't overlap modal
- [X] **5.4** Add haptic feedback (navigator.vibrate if supported)
- [X] **5.5** Accessibility: Add ARIA labels to buttons
- [X] **5.6** Accessibility: Test keyboard-only navigation
- [X] **5.7** Accessibility: Add focus trap in command palette
- [X] **5.8** Accessibility: Screen reader announcements
- [X] **5.9** Visual polish: Smooth transitions (300ms ease-in-out)
- [X] **5.10** Visual polish: Loading states for async operations
- [X] **5.11** Error handling: Console messages for failures
- [X] **5.12** Performance: Debounce search input (150ms)
- [X] **5.13** Performance: Optimize fuzzy search for large lists
- [X] **5.14** Cross-device: Test iPhone Safari (FAB, full-screen modal)
- [X] **5.15** Cross-device: Test Android Chrome (touch targets, back button)
- [X] **5.16** Cross-device: Test iPad Safari (hybrid UI, keyboard + touch)
- [X] **5.17** Cross-device: Test Desktop (all keyboard shortcuts)
- [X] **5.18** Run tests: `make test` passes all cases
- [X] **5.19** Verify: No console errors or warnings
- [X] **5.20** Final checklist: All features work end-to-end

## Testing Checklist

### Word Wrap
- [X] Alt+Z toggles wrap in active editor
- [X] State persists on page reload
- [X] Works across multiple tabs
- [X] localStorage contains `wordWrap` preference

### Sidebar Collapse
- [X] Toggle button works
- [X] Cmd+B keyboard shortcut works
- [X] Smooth 300ms transition
- [X] Mobile vs desktop states separate
- [X] State persists on reload
- [X] File tree hidden when collapsed

### Command Palette
- [X] Opens with Cmd+K
- [X] Opens with Cmd+Shift+P
- [X] FAB button works on mobile
- [X] Search filters instantly
- [X] Keyboard navigation (↑↓ Enter Esc)
- [X] Recent commands prioritized
- [X] All 26 commands present (25 editor + 1 view)
- [X] Commands execute correctly
- [X] Closes after execution or on Esc

### Text Commands
- [X] Sort ascending/descending
- [X] Case transformations (upper, lower, title)
- [X] Trim whitespace removes trailing spaces
- [X] Delete empty lines filters correctly
- [X] Go to line prompt works
- [X] Copy path to clipboard
- [X] Duplicate/delete line functions
- [X] Comment toggle works
- [X] Indent/outdent work

### Cross-Device
- [X] iPhone Safari: FAB positioning, full-screen modal
- [X] Android Chrome: Touch targets, gestures
- [X] iPad Safari: Keyboard shortcuts + touch
- [X] Desktop: All keyboard shortcuts functional
- [X] No layout shifts on mobile
- [X] Virtual keyboard doesn't break UI

## Implementation Notes
- Follow existing code patterns (IIFE modules, debounced persistence)
- Use CodeMirror Compartment API for dynamic reconfiguration
- Alpine.js already loaded - leverage existing patterns
- Mobile-first CSS with Tailwind breakpoints
- Debounce API calls to 500ms to prevent excessive requests
- localStorage keys: `editor.preferences.*` (wordWrap, sidebarCollapsed, sidebarCollapsedMobile, recentCommands)
