# Phase 5: Polish & Testing - Comprehensive Verification

## Testing Date: 2026-02-16
## Status: IN PROGRESS

---

## Section 1: Feature Integration Testing

### 1.1 Word Wrap Toggle (Alt+Z)
- [ ] Open any file in editor
- [ ] Press Alt+Z → lines should wrap at editor width
- [ ] Press Alt+Z again → lines should unwrap
- [ ] Open DevTools Console → check `localStorage.getItem('editor.preferences.wordWrap')`
- [ ] Reload page → preference persists
- [ ] Check Network tab → debounced API call to `/api/settings` appears
- [ ] Test in multiple editor panes → state applies to focused pane only

### 1.2 Sidebar Collapse (Cmd+B)
- [ ] Click hamburger toggle button in sidebar header → sidebar collapses to 40px
- [ ] Click again → sidebar expands
- [ ] Press Cmd+B (or Ctrl+B) → toggles sidebar
- [ ] Check transition is smooth (300ms ease-in-out)
- [ ] Resize window to mobile (< 768px) → sidebar state independent from desktop
- [ ] Check localStorage: `editor.preferences.sidebarCollapsed` and `editor.preferences.sidebarCollapsedMobile`
- [ ] Reload page → state persists for current breakpoint
- [ ] Verify resize handle is hidden when collapsed
- [ ] Test on tablet (iPad) → responds to breakpoint

### 1.3 Command Palette (Cmd+K / Cmd+Shift+P)
- [ ] Press Cmd+K → command palette modal opens
- [ ] Type "wrap" → filters to word wrap command
- [ ] Press ↑↓ → navigates command list with highlight
- [ ] Press Enter → executes command and closes palette
- [ ] Press Esc → closes palette without executing
- [ ] Click search input X button → clears search
- [ ] Click command item → executes and closes
- [ ] Recent commands appear at top on fresh open
- [ ] Check localStorage: `editor.preferences.recentCommands` (max 5)
- [ ] Test on mobile: FAB button appears bottom-right
- [ ] FAB button opens palette

### 1.4 Text Commands - Full Registry
- [ ] Open command palette (Cmd+K)
- [ ] Verify 25+ commands visible in list
- [ ] Commands organized by category: Text, Line, Navigation, File, Selection
- [ ] Each command has icon (Heroicons SVG)
- [ ] Each command has category badge
- [ ] Recent commands have indicator dot

---

## Section 2: Text Command Execution Testing

### 2.1 Text Transformations
- [ ] Select text lines → Cmd+K → "Sort Ascending" → lines sorted A-Z
- [ ] Select text → "Sort Descending" → lines sorted Z-A
- [ ] Select text → "Uppercase" → ALL CAPS
- [ ] Select text → "Lowercase" → all lowercase
- [ ] Select text → "Title Case" → First Letter Capitalized Each Word
- [ ] Select text → "Snake Case" → snake_case format
- [ ] Select text → "Camel Case" → camelCaseFormat
- [ ] Select text → "Kebab Case" → kebab-case-format
- [ ] Text with trailing spaces → "Trim Whitespace" → trailing spaces removed
- [ ] Multiple empty lines → "Delete Empty Lines" → empty lines removed

### 2.2 Line Operations
- [ ] Cursor on line → "Duplicate Line" → line duplicated below
- [ ] Cursor on line → "Delete Line" → line deleted
- [ ] Selection → "Indent" → indented 4 spaces (or language indent)
- [ ] Selection → "Outdent" → de-indented
- [ ] Line with code → "Toggle Comment" → line commented (// or #)
- [ ] Commented line → "Toggle Comment" → comment removed
- [ ] Multiple lines → "Join Lines" → joined with space
- [ ] Selection → "Reverse Lines" → lines reversed
- [ ] Duplicated text → "Remove Duplicates" → duplicates removed

### 2.3 Navigation
- [ ] Cmd+K → "Go to Line" → prompt appears
- [ ] Enter line number (e.g., 50) → cursor jumps to line 50
- [ ] Cancel prompt → nothing happens

### 2.4 File Operations
- [ ] "Copy File Path" → clipboard contains absolute path
- [ ] "Copy Relative Path" → clipboard contains relative path
- [ ] "Copy File Name" → clipboard contains just filename

### 2.5 Selection Commands
- [ ] "Select All" → all content selected
- [ ] "Select Line" → entire line selected
- [ ] "Select Word" → current word selected

---

## Section 3: Keyboard Navigation & Accessibility

### 3.1 Keyboard-Only Navigation
- [ ] Tab through UI elements → all interactive elements reachable
- [ ] Focus on search input → search input focused
- [ ] Shift+Tab → reverse navigation works
- [ ] Space/Enter on buttons → activates command
- [ ] Esc from modal → closes and focus returns

### 3.2 ARIA Attributes
- [ ] Search input has `aria-label="Search commands"`
- [ ] Modal has `aria-label="Command Palette"`
- [ ] Command list has `role="listbox"`
- [ ] Selected command has `aria-selected="true"`
- [ ] Screen reader announces command count

### 3.3 Focus Management
- [ ] Modal open → focus trapped in modal
- [ ] Cmd+K → focus moves to search input
- [ ] Command executed → focus returns to editor
- [ ] Modal closed → focus returns to previously focused element

---

## Section 4: Mobile Optimization

### 4.1 Touch Targets
- [ ] All buttons ≥ 48px × 48px
- [ ] FAB button: 56px (w-14 h-14) ✓
- [ ] Toggle button: 40px × 40px ✓
- [ ] Search input: 44px+ height ✓

### 4.2 Mobile Breakpoints
- [ ] iPhone SE (375px): FAB visible, modal full-screen
- [ ] iPhone 14 (390px): All UI properly sized
- [ ] iPhone 14 Pro Max (430px): No overflow
- [ ] iPad (768px): Sidebar visible (desktop mode)
- [ ] iPad Pro (1024px): Desktop layout

### 4.3 Safe Area Handling
- [ ] FAB button on iPhone X+ with notch → respects safe-area-inset-bottom
- [ ] FAB position: `bottom-4 sm:bottom-20` (above other controls)
- [ ] Modal dialog: respects viewport height

### 4.4 Virtual Keyboard
- [ ] iOS: Virtual keyboard doesn't overlap modal
- [ ] Android: Virtual keyboard doesn't overlap search input
- [ ] Search input scrolls into view when focused
- [ ] Modal height adjusts when keyboard shown

### 4.5 Haptic Feedback
- [ ] Command execution provides light haptic (navigator.vibrate)
- [ ] Mobile button taps provide feedback

---

## Section 5: Performance & Optimization

### 5.1 Search Performance
- [ ] Type "sort" → filters instantly (< 50ms)
- [ ] Search with 25+ commands → no lag
- [ ] Debounce interval: 150ms ✓
- [ ] No console errors during search

### 5.2 Debounced Persistence
- [ ] Toggle word wrap → localStorage updates immediately
- [ ] API call debounced at 500ms ✓
- [ ] Rapid toggles → only one API call after 500ms pause
- [ ] Network tab shows single `/api/settings` request

### 5.3 Memory & Rendering
- [ ] Modal open/close cycles → no memory leaks
- [ ] 100+ rapid command searches → smooth performance
- [ ] Editor remains responsive during command execution

---

## Section 6: Cross-Browser Testing

### 6.1 Desktop Browsers
- [ ] Chrome (latest): All features work
- [ ] Firefox (latest): All features work
- [ ] Safari (latest): All features work
- [ ] Edge (latest): All features work

### 6.2 Mobile Browsers
- [ ] iOS Safari: All features work, FAB visible
- [ ] Android Chrome: All features work, FAB visible
- [ ] Android Firefox: All features work
- [ ] Samsung Internet: All features work

### 6.3 Tablet Browsers
- [ ] iPad Safari: Desktop layout, no FAB
- [ ] iPad Chrome: Desktop layout, no FAB
- [ ] Android Tablet: Responsive layout

---

## Section 7: Error Handling & Edge Cases

### 7.1 Error Scenarios
- [ ] API call fails (network error) → localStorage preference still works locally
- [ ] Invalid line number in "Go to Line" → graceful fallback
- [ ] Empty selection on text command → command returns false, no-op
- [ ] Non-existent file path → copy path still works

### 7.2 Edge Cases
- [ ] Very long lines (1000+ chars) → wrap works, no layout break
- [ ] Very large files (10MB+) → toggles don't freeze editor
- [ ] Rapid Cmd+K/Cmd+B/Alt+Z → all shortcuts debounced properly
- [ ] Palette open + click outside → closes with ESC behavior

### 7.3 State Consistency
- [ ] Multiple editor panes → each has own preferences
- [ ] Switch between panes → correct state for each pane
- [ ] Close pane → state still persists for that pane name
- [ ] Reload → all previous states restored

---

## Section 8: Visual Polish

### 8.1 Transitions & Animations
- [ ] Sidebar collapse: 300ms smooth transition ✓
- [ ] Modal open: fade-in transition smooth
- [ ] Command highlight: smooth color transition
- [ ] Recent commands: list appears without jank

### 8.2 Visual Hierarchy
- [ ] Recent commands visually distinct (dot indicator)
- [ ] Search results properly highlighted
- [ ] Selected command has clear highlight (indigo-600/20)
- [ ] Category badges clearly readable

### 8.3 Color & Contrast
- [ ] All text meets WCAG AA contrast ratio (4.5:1)
- [ ] Dark theme colors consistent (gray-900, gray-800, indigo-600)
- [ ] Icons visible in both light and dark backgrounds
- [ ] Focus indicators clearly visible

---

## Section 9: Code Quality Checks

### 9.1 Console Errors
- [ ] Open DevTools Console
- [ ] No JavaScript errors after page load
- [ ] No warnings on interaction
- [ ] localStorage access successful

### 9.2 Network Requests
- [ ] `/api/settings` called for word wrap changes
- [ ] `/api/settings` called for sidebar changes
- [ ] Debounced: only one call per 500ms pause
- [ ] All requests return 200 OK

### 9.3 Browser APIs
- [ ] localStorage available and working
- [ ] navigator.clipboard available for copy commands
- [ ] matchMedia for mobile detection working
- [ ] SVG icons rendering correctly

---

## Section 10: Final Integration Checklist

### 10.1 All Features Working Together
- [ ] Open editor with file
- [ ] Toggle word wrap (Alt+Z)
- [ ] Collapse sidebar (Cmd+B)
- [ ] Open command palette (Cmd+K)
- [ ] Execute text command (e.g., Sort)
- [ ] Close palette (Esc)
- [ ] All changes reflected in editor
- [ ] Reload page → all state persists

### 10.2 Keyboard Shortcuts Summary
- [ ] **Alt+Z** → Toggle word wrap ✓
- [ ] **Cmd+B** → Toggle sidebar ✓
- [ ] **Cmd+K** → Open command palette ✓
- [ ] **Cmd+Shift+P** → Open command palette (alternative) ✓
- [ ] **↑↓** → Navigate commands in palette ✓
- [ ] **Enter** → Execute command ✓
- [ ] **Esc** → Close palette ✓

### 10.3 localStorage Keys
- [ ] `editor.preferences.wordWrap` exists
- [ ] `editor.preferences.sidebarCollapsed` exists
- [ ] `editor.preferences.sidebarCollapsedMobile` exists
- [ ] `editor.preferences.recentCommands` exists (array, max 5)

### 10.4 API Endpoints
- [ ] `POST /projects/{id}/api/preferences` receives wordWrap updates
- [ ] `POST /projects/{id}/api/preferences` receives sidebar updates
- [ ] Debounced: 500ms delay between calls
- [ ] Error handling: logs to console on failure

---

## Sign-Off Checklist

When all sections verified:
- [ ] All 10 sections tested
- [ ] No critical bugs found
- [ ] No console errors
- [ ] All keyboard shortcuts working
- [ ] Mobile experience verified on real device
- [ ] Cross-browser testing passed
- [ ] Ready for commit and merge

---

## Notes
- Test on real mobile devices when possible (not just browser DevTools)
- Check iOS safe areas on iPhone X and later
- Verify on both portrait and landscape modes
- Test with different editor content types (JavaScript, Python, HTML, CSS)
- Test with rapid user interactions (spam clicking, rapid keyboard events)
