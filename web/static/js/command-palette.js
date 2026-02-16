// ClawIDE Command Palette — Alpine.js-powered searchable command interface
// Provides fuzzy search, keyboard navigation, recent commands, and mobile FAB
(function() {
    'use strict';

    var RECENT_KEY = 'editor.preferences.recentCommands';
    var MAX_RECENT = 5;

    // --- Heroicon SVG paths (outline, 24x24 viewBox) ---
    var ICONS = {
        sort: '<path stroke-linecap="round" stroke-linejoin="round" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12"/>',
        text: '<path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h7"/>',
        line: '<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14"/>',
        copy: '<path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184"/>',
        navigate: '<path stroke-linecap="round" stroke-linejoin="round" d="M3 4.5h14.25M3 9h9.75M3 13.5h5.25m5.25-.75L17.25 9m0 0L21 12.75M17.25 9v12"/>',
        select: '<path stroke-linecap="round" stroke-linejoin="round" d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"/>',
        indent: '<path stroke-linecap="round" stroke-linejoin="round" d="M17.25 8.25L21 12m0 0l-3.75 3.75M21 12H3"/>',
        comment: '<path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5h10.5m-10.5 3h7.5m-7.5 3h4.5M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"/>',
        file: '<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9z"/>',
        transform: '<path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"/>',
        delete: '<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"/>',
        sidebar: '<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v12a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18V6zM10.5 3.75h7.5A2.25 2.25 0 0 1 20.25 6v12a2.25 2.25 0 0 1-2.25 2.25h-7.5"/>',
    };

    function makeIcon(pathKey) {
        return '<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">' + (ICONS[pathKey] || ICONS.text) + '</svg>';
    }

    // --- Command Registry ---
    // Fields match what workspace.html template expects: name, icon (HTML), category, shortcut, handler
    var commands = [
        // Text Transformations
        { id: 'sortLinesAscending', name: 'Sort Lines Ascending', category: 'Text', icon: makeIcon('sort'), shortcut: '', handler: 'sortLinesAscending' },
        { id: 'sortLinesDescending', name: 'Sort Lines Descending', category: 'Text', icon: makeIcon('sort'), shortcut: '', handler: 'sortLinesDescending' },
        { id: 'transformToUppercase', name: 'Transform to UPPERCASE', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToUppercase' },
        { id: 'transformToLowercase', name: 'Transform to lowercase', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToLowercase' },
        { id: 'transformToTitleCase', name: 'Transform to Title Case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToTitleCase' },
        { id: 'transformToSnakeCase', name: 'Transform to snake_case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToSnakeCase' },
        { id: 'transformToCamelCase', name: 'Transform to camelCase', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToCamelCase' },
        { id: 'transformToKebabCase', name: 'Transform to kebab-case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToKebabCase' },
        { id: 'trimTrailingWhitespace', name: 'Trim Trailing Whitespace', category: 'Text', icon: makeIcon('text'), shortcut: '', handler: 'trimTrailingWhitespace' },
        { id: 'deleteEmptyLines', name: 'Delete Empty Lines', category: 'Text', icon: makeIcon('delete'), shortcut: '', handler: 'deleteEmptyLines' },

        // Line Operations
        { id: 'duplicateLine', name: 'Duplicate Line', category: 'Line', icon: makeIcon('copy'), shortcut: '', handler: 'duplicateLine' },
        { id: 'deleteLine', name: 'Delete Line', category: 'Line', icon: makeIcon('delete'), shortcut: '', handler: 'deleteLine' },
        { id: 'joinLines', name: 'Join Lines', category: 'Line', icon: makeIcon('line'), shortcut: '', handler: 'joinLines' },
        { id: 'reverseLines', name: 'Reverse Lines', category: 'Line', icon: makeIcon('sort'), shortcut: '', handler: 'reverseLines' },
        { id: 'removeDuplicateLines', name: 'Remove Duplicate Lines', category: 'Line', icon: makeIcon('delete'), shortcut: '', handler: 'removeDuplicateLines' },
        { id: 'indentSelection', name: 'Indent Selection', category: 'Line', icon: makeIcon('indent'), shortcut: 'Tab', handler: 'indentSelection' },
        { id: 'outdentSelection', name: 'Outdent Selection', category: 'Line', icon: makeIcon('indent'), shortcut: 'Shift+Tab', handler: 'outdentSelection' },
        { id: 'toggleComment', name: 'Toggle Comment', category: 'Line', icon: makeIcon('comment'), shortcut: 'Cmd+/', handler: 'toggleComment' },

        // Navigation
        { id: 'goToLine', name: 'Go to Line...', category: 'Navigation', icon: makeIcon('navigate'), shortcut: 'Ctrl+G', handler: 'goToLine' },

        // File Info
        { id: 'copyFilePath', name: 'Copy File Path', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyFilePath' },
        { id: 'copyRelativePath', name: 'Copy Relative Path', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyRelativePath' },
        { id: 'copyFileName', name: 'Copy File Name', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyFileName' },

        // Selection
        { id: 'selectAll', name: 'Select All', category: 'Selection', icon: makeIcon('select'), shortcut: 'Cmd+A', handler: 'selectAll' },
        { id: 'selectLine', name: 'Select Line', category: 'Selection', icon: makeIcon('select'), shortcut: 'Cmd+L', handler: 'selectLine' },
        { id: 'selectWord', name: 'Select Word', category: 'Selection', icon: makeIcon('select'), shortcut: '', handler: 'selectWord' },

        // View
        { id: 'toggleSidebar', name: 'Toggle Sidebar', category: 'View', icon: makeIcon('sidebar'), shortcut: 'Cmd+B', handler: 'toggleSidebar' },
    ];

    // --- Fuzzy Search ---
    function fuzzyMatch(query, text) {
        query = query.toLowerCase();
        text = text.toLowerCase();

        if (text === query) return { match: true, score: 100 };
        if (text.indexOf(query) === 0) return { match: true, score: 80 };
        if (text.indexOf(query) !== -1) return { match: true, score: 60 };

        var qi = 0;
        var score = 0;
        var consecutive = 0;
        for (var ti = 0; ti < text.length && qi < query.length; ti++) {
            if (text.charAt(ti) === query.charAt(qi)) {
                qi++;
                consecutive++;
                score += consecutive * 2;
            } else {
                consecutive = 0;
            }
        }

        if (qi === query.length) {
            return { match: true, score: score };
        }

        return { match: false, score: 0 };
    }

    function searchCommands(query, cmds) {
        if (!query || query.trim() === '') return cmds;

        var results = [];
        for (var i = 0; i < cmds.length; i++) {
            var nameMatch = fuzzyMatch(query, cmds[i].name);
            var catMatch = fuzzyMatch(query, cmds[i].category);
            var bestScore = Math.max(nameMatch.score, catMatch.score);
            if (nameMatch.match || catMatch.match) {
                results.push({ command: cmds[i], score: bestScore });
            }
        }

        results.sort(function(a, b) { return b.score - a.score; });
        return results.map(function(r) { return r.command; });
    }

    // --- Recent Commands ---
    function loadRecent() {
        try {
            var stored = localStorage.getItem(RECENT_KEY);
            return stored ? JSON.parse(stored) : [];
        } catch (e) {
            return [];
        }
    }

    function saveRecent(recentIds) {
        try {
            localStorage.setItem(RECENT_KEY, JSON.stringify(recentIds.slice(0, MAX_RECENT)));
        } catch (e) {
            // ignore
        }
    }

    function addToRecent(commandId) {
        var recent = loadRecent();
        recent = recent.filter(function(id) { return id !== commandId; });
        recent.unshift(commandId);
        saveRecent(recent);
    }

    function getRecentCommands() {
        var recentIds = loadRecent();
        var result = [];
        for (var i = 0; i < recentIds.length; i++) {
            for (var j = 0; j < commands.length; j++) {
                if (commands[j].id === recentIds[i]) {
                    result.push(commands[j]);
                    break;
                }
            }
        }
        return result;
    }

    // --- Execute Command by ID ---
    function runCommandById(commandId) {
        var cmd = null;
        for (var i = 0; i < commands.length; i++) {
            if (commands[i].id === commandId) {
                cmd = commands[i];
                break;
            }
        }
        if (!cmd) return false;

        addToRecent(commandId);

        if (typeof window.ClawIDECommands !== 'undefined' && typeof window.ClawIDECommands[cmd.handler] === 'function') {
            return window.ClawIDECommands[cmd.handler]();
        }
        console.warn('Command handler not found:', cmd.handler);
        return false;
    }

    // --- Alpine.js Component Data ---
    // Template uses: open, query, selectedIndex, filteredCommands, recentCommands
    // Methods: close(), openPalette(), executeCommand(cmd), onSearchInput(), onKeydown(e), isRecent(cmd)
    window._clawIDECommandPaletteData = function() {
        return {
            open: false,
            query: '',
            selectedIndex: 0,
            filteredCommands: [],
            recentCommands: [],

            init: function() {
                this.recentCommands = getRecentCommands();
                this.updateFiltered();
            },

            openPalette: function() {
                this.open = true;
                this.query = '';
                this.selectedIndex = 0;
                this.recentCommands = getRecentCommands();
                this.updateFiltered();
                var self = this;
                this.$nextTick(function() {
                    var input = document.getElementById('command-palette-search');
                    if (input) input.focus();
                });
            },

            close: function() {
                this.open = false;
                this.query = '';
                this.selectedIndex = 0;
            },

            updateFiltered: function() {
                if (!this.query || !this.query.trim()) {
                    var recentIds = this.recentCommands.map(function(c) { return c.id; });
                    var rest = commands.filter(function(c) { return recentIds.indexOf(c.id) === -1; });
                    this.filteredCommands = this.recentCommands.concat(rest);
                } else {
                    this.filteredCommands = searchCommands(this.query, commands);
                }
                this.selectedIndex = 0;
            },

            onSearchInput: function() {
                this.updateFiltered();
            },

            onKeydown: function(e) {
                if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    if (this.selectedIndex < this.filteredCommands.length - 1) {
                        this.selectedIndex++;
                    }
                    this.scrollSelectedIntoView();
                } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    if (this.selectedIndex > 0) {
                        this.selectedIndex--;
                    }
                    this.scrollSelectedIntoView();
                } else if (e.key === 'Enter') {
                    e.preventDefault();
                    if (this.filteredCommands.length > 0) {
                        this.executeCommand(this.filteredCommands[this.selectedIndex]);
                    }
                } else if (e.key === 'Escape') {
                    e.preventDefault();
                    this.close();
                }
            },

            scrollSelectedIntoView: function() {
                var idx = this.selectedIndex;
                this.$nextTick(function() {
                    var item = document.querySelector('[data-palette-index="' + idx + '"]');
                    if (item) {
                        item.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
                    }
                });
            },

            executeCommand: function(cmd) {
                this.close();
                setTimeout(function() {
                    runCommandById(cmd.id);
                }, 50);
            },

            isRecent: function(cmd) {
                return this.recentCommands.some(function(c) { return c.id === cmd.id; });
            },
        };
    };

    // --- Keyboard Shortcuts (global) ---
    document.addEventListener('keydown', function(e) {
        var isCmdK = (e.metaKey || e.ctrlKey) && e.key === 'k';
        var isCmdShiftP = (e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'P';

        if (isCmdK || isCmdShiftP) {
            e.preventDefault();
            if (typeof window.ClawIDEPalette !== 'undefined') {
                window.ClawIDEPalette.toggle();
            }
        }
    });

    // --- Public API ---
    window.ClawIDEPalette = {
        toggle: function() {
            // Find the Alpine component by walking DOM for x-data with our function
            var els = document.querySelectorAll('[x-data]');
            for (var i = 0; i < els.length; i++) {
                var el = els[i];
                if (el._x_dataStack && el._x_dataStack[0] && typeof el._x_dataStack[0].openPalette === 'function') {
                    var data = el._x_dataStack[0];
                    if (data.open) {
                        data.close();
                    } else {
                        data.openPalette();
                    }
                    return;
                }
            }
        },
        open: function() {
            var els = document.querySelectorAll('[x-data]');
            for (var i = 0; i < els.length; i++) {
                var el = els[i];
                if (el._x_dataStack && el._x_dataStack[0] && typeof el._x_dataStack[0].openPalette === 'function') {
                    el._x_dataStack[0].openPalette();
                    return;
                }
            }
        },
        close: function() {
            var els = document.querySelectorAll('[x-data]');
            for (var i = 0; i < els.length; i++) {
                var el = els[i];
                if (el._x_dataStack && el._x_dataStack[0] && typeof el._x_dataStack[0].close === 'function' && 'openPalette' in el._x_dataStack[0]) {
                    el._x_dataStack[0].close();
                    return;
                }
            }
        },
        executeCommand: runCommandById,
        getCommands: function() { return commands.slice(); },
    };
})();
