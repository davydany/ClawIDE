// ClawIDE Modifier Keys Toolbar
// Virtual modifier & special keys for touch devices
(function() {
    'use strict';

    // Modifier state: 'inactive' | 'active-once' | 'locked'
    var modifiers = {
        ctrl: 'inactive',
        alt: 'inactive',
        meta: 'inactive'
    };

    // Double-tap detection timing (ms)
    var DOUBLE_TAP_MS = 300;
    var lastTapTime = {};

    // CSS class helpers
    function updateButtonState(btn, modName) {
        var state = modifiers[modName];
        btn.classList.remove('active-once', 'locked');
        btn.removeAttribute('aria-pressed');
        if (state === 'active-once') {
            btn.classList.add('active-once');
            btn.setAttribute('aria-pressed', 'true');
        } else if (state === 'locked') {
            btn.classList.add('locked');
            btn.setAttribute('aria-pressed', 'true');
        }
    }

    function updateAllButtons() {
        var toolbar = document.getElementById('modifier-toolbar');
        if (!toolbar) return;
        var btns = toolbar.querySelectorAll('[data-modifier]');
        for (var i = 0; i < btns.length; i++) {
            var mod = btns[i].getAttribute('data-modifier');
            updateButtonState(btns[i], mod);
        }
    }

    // State machine transition on tap
    function handleModifierTap(modName) {
        var now = Date.now();
        var prev = lastTapTime[modName] || 0;
        var state = modifiers[modName];

        if (state === 'inactive') {
            if (now - prev < DOUBLE_TAP_MS) {
                // Double tap -> locked
                modifiers[modName] = 'locked';
            } else {
                // Single tap -> active-once
                modifiers[modName] = 'active-once';
            }
        } else {
            // Any tap while active-once or locked -> inactive
            modifiers[modName] = 'inactive';
        }

        lastTapTime[modName] = now;
        updateAllButtons();
    }

    // Deactivate non-locked modifiers (after a keypress)
    function deactivateOnce() {
        var changed = false;
        for (var key in modifiers) {
            if (modifiers[key] === 'active-once') {
                modifiers[key] = 'inactive';
                changed = true;
            }
        }
        if (changed) updateAllButtons();
    }

    // Check if any modifier is active
    function anyModifierActive() {
        for (var key in modifiers) {
            if (modifiers[key] !== 'inactive') return true;
        }
        return false;
    }

    // Apply Ctrl modifier to a single character
    function ctrlChar(ch) {
        var code = ch.charCodeAt(0);
        // a-z -> 0x01-0x1a
        if (code >= 97 && code <= 122) {
            return String.fromCharCode(code - 96);
        }
        // A-Z -> 0x01-0x1a
        if (code >= 65 && code <= 90) {
            return String.fromCharCode(code - 64);
        }
        // Special Ctrl combos
        if (ch === '[') return '\x1b';
        if (ch === '\\') return '\x1c';
        if (ch === ']') return '\x1d';
        if (ch === '^') return '\x1e';
        if (ch === '_') return '\x1f';
        if (ch === '?') return '\x7f';
        if (ch === '@') return '\x00';
        // For other chars, return as-is
        return ch;
    }

    // Build modifier parameter for CSI sequences (arrows, etc.)
    // 1=none, 2=Shift, 3=Alt, 4=Shift+Alt, 5=Ctrl, 6=Ctrl+Shift, 7=Ctrl+Alt, 8=Ctrl+Shift+Alt
    function modifierParam() {
        var val = 1;
        if (modifiers.ctrl !== 'inactive') val += 4;
        if (modifiers.alt !== 'inactive') val += 2;
        if (modifiers.meta !== 'inactive') val += 8; // treat Cmd similar to meta
        return val;
    }

    // Data interceptor - transforms outgoing keystrokes based on modifier state
    function interceptor(data) {
        if (!anyModifierActive()) return data;

        // Multi-char input (paste or escape sequences) -> skip transformation
        if (data.length > 1) {
            deactivateOnce();
            return data;
        }

        // Single character transformation
        var result = data;

        if (modifiers.ctrl !== 'inactive') {
            result = ctrlChar(result);
        }

        if (modifiers.alt !== 'inactive' || modifiers.meta !== 'inactive') {
            // Prepend ESC for Alt/Meta
            result = '\x1b' + result;
        }

        deactivateOnce();
        return result;
    }

    // Send a key sequence to the focused terminal pane
    function sendKey(sequence) {
        var paneID = window.ClawIDETerminal.getFocusedPaneID();
        if (!paneID) return;
        window.ClawIDETerminal.sendInput(paneID, sequence);
    }

    // Arrow key sequences with modifier support
    function arrowSequence(direction) {
        // direction: A=Up, B=Down, C=Right, D=Left
        var mod = modifierParam();
        var seq;
        if (mod > 1) {
            seq = '\x1b[1;' + mod + direction;
        } else {
            seq = '\x1b[' + direction;
        }
        deactivateOnce();
        return seq;
    }

    // Direct key handlers
    var directKeys = {
        tab: function() { sendKey('\x09'); },
        esc: function() { sendKey('\x1b'); },
        up: function() { sendKey(arrowSequence('A')); },
        down: function() { sendKey(arrowSequence('B')); },
        right: function() { sendKey(arrowSequence('C')); },
        left: function() { sendKey(arrowSequence('D')); }
    };

    // Initialize toolbar interactions
    function init() {
        var toolbar = document.getElementById('modifier-toolbar');
        if (!toolbar) return;

        // Register data interceptor
        if (window.ClawIDETerminal && window.ClawIDETerminal.addDataInterceptor) {
            window.ClawIDETerminal.addDataInterceptor(interceptor);
        }

        // Prevent focus loss on all toolbar buttons
        toolbar.addEventListener('pointerdown', function(e) {
            e.preventDefault();
        });
        toolbar.addEventListener('touchstart', function(e) {
            // Only prevent default on the toolbar buttons, not passthrough
            if (e.target.closest('button')) {
                e.preventDefault();
            }
        }, { passive: false });

        // Modifier buttons
        var modBtns = toolbar.querySelectorAll('[data-modifier]');
        for (var i = 0; i < modBtns.length; i++) {
            (function(btn) {
                var modName = btn.getAttribute('data-modifier');
                btn.addEventListener('click', function() {
                    handleModifierTap(modName);
                });
            })(modBtns[i]);
        }

        // Direct key buttons
        var keyBtns = toolbar.querySelectorAll('[data-key]');
        for (var j = 0; j < keyBtns.length; j++) {
            (function(btn) {
                var keyName = btn.getAttribute('data-key');
                btn.addEventListener('click', function() {
                    if (directKeys[keyName]) {
                        directKeys[keyName]();
                    }
                });
            })(keyBtns[j]);
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose for testing
    window.ClawIDEModifierKeys = {
        getState: function() { return Object.assign({}, modifiers); },
        reset: function() {
            modifiers.ctrl = 'inactive';
            modifiers.alt = 'inactive';
            modifiers.meta = 'inactive';
            updateAllButtons();
        }
    };
})();
