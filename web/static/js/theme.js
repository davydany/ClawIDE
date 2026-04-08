// ClawIDE Theme Manager
(function() {
    'use strict';

    var THEME_STORAGE_KEY = 'clawide-theme';
    var MODE_STORAGE_KEY = 'clawide-mode';
    var VALID_THEMES = ['default', 'claude', 'mono'];
    var VALID_MODES = ['dark', 'light'];

    function get() {
        return document.documentElement.dataset.theme || 'default';
    }

    function getMode() {
        return document.documentElement.dataset.mode || 'dark';
    }

    function set(name) {
        if (VALID_THEMES.indexOf(name) === -1) name = 'default';

        // Apply to DOM immediately
        if (name === 'default') {
            delete document.documentElement.dataset.theme;
        } else {
            document.documentElement.dataset.theme = name;
        }

        // Persist to localStorage for FOUC prevention on next load
        localStorage.setItem(THEME_STORAGE_KEY, name);

        // Persist to backend
        fetch('/api/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ theme: name })
        }).catch(function(err) {
            console.warn('Failed to save theme preference:', err);
        });

        // Dispatch event for terminal, mermaid, highlight.js listeners
        window.dispatchEvent(new CustomEvent('clawide:theme-changed', {
            detail: { theme: name, mode: getMode() }
        }));
    }

    function setMode(mode) {
        if (VALID_MODES.indexOf(mode) === -1) mode = 'dark';

        // Apply to DOM immediately
        if (mode === 'dark') {
            delete document.documentElement.dataset.mode;
        } else {
            document.documentElement.dataset.mode = mode;
        }

        // Persist to localStorage for FOUC prevention
        localStorage.setItem(MODE_STORAGE_KEY, mode);

        // Persist to backend
        fetch('/api/settings', {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ mode: mode })
        }).catch(function(err) {
            console.warn('Failed to save mode preference:', err);
        });

        // Dispatch event for dependent systems
        window.dispatchEvent(new CustomEvent('clawide:theme-changed', {
            detail: { theme: get(), mode: mode }
        }));
    }

    function toggleMode() {
        setMode(getMode() === 'dark' ? 'light' : 'dark');
    }

    window.ClawIDETheme = {
        get: get,
        set: set,
        getMode: getMode,
        setMode: setMode,
        toggleMode: toggleMode,
        VALID_THEMES: VALID_THEMES,
        VALID_MODES: VALID_MODES
    };
})();
