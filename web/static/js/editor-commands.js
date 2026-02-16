// ClawIDE Editor Commands — Text manipulation handlers for command palette
// Each command operates on the active CodeMirror 6 EditorView via state/dispatch
(function() {
    'use strict';

    // --- Helpers ---

    function getView() {
        if (typeof window.ClawIDEEditor === 'undefined') return null;
        return window.ClawIDEEditor.getActiveEditorView();
    }

    function getSelectionRange(view) {
        var sel = view.state.selection.main;
        return { from: sel.from, to: sel.to, empty: sel.empty };
    }

    function getSelectedText(view) {
        var sel = view.state.selection.main;
        return view.state.sliceDoc(sel.from, sel.to);
    }

    function replaceSelection(view, text) {
        view.dispatch({
            changes: { from: view.state.selection.main.from, to: view.state.selection.main.to, insert: text },
        });
    }

    // Get the full line range for the cursor or selection
    function getLineRange(view) {
        var sel = view.state.selection.main;
        var startLine = view.state.doc.lineAt(sel.from);
        var endLine = view.state.doc.lineAt(sel.to);
        return {
            from: startLine.from,
            to: endLine.to,
            startLine: startLine,
            endLine: endLine,
        };
    }

    // Expand selection to full lines and return the text
    function getSelectedLines(view) {
        var range = getLineRange(view);
        return view.state.sliceDoc(range.from, range.to);
    }

    function replaceLines(view, text) {
        var range = getLineRange(view);
        view.dispatch({
            changes: { from: range.from, to: range.to, insert: text },
        });
    }

    // Get the current cursor line
    function getCursorLine(view) {
        return view.state.doc.lineAt(view.state.selection.main.head);
    }

    // Comment style map based on file extension
    function getCommentStyle(filePath) {
        if (!filePath) return { line: '//' };
        var ext = filePath.split('.').pop().toLowerCase();
        var hashComments = ['py', 'rb', 'sh', 'bash', 'zsh', 'yaml', 'yml', 'toml', 'pl', 'r', 'coffee', 'makefile'];
        var slashComments = ['js', 'mjs', 'cjs', 'jsx', 'ts', 'tsx', 'java', 'c', 'cpp', 'h', 'cs', 'go', 'rs', 'swift', 'kt', 'scala', 'json', 'jsonc', 'scss', 'less'];
        var htmlComments = ['html', 'htm', 'xml', 'svg', 'vue'];
        var cssComments = ['css'];
        var luaComments = ['lua'];
        var sqlComments = ['sql'];

        if (hashComments.indexOf(ext) !== -1) return { line: '#' };
        if (slashComments.indexOf(ext) !== -1) return { line: '//' };
        if (htmlComments.indexOf(ext) !== -1) return { block: ['<!--', '-->'] };
        if (cssComments.indexOf(ext) !== -1) return { block: ['/*', '*/'] };
        if (luaComments.indexOf(ext) !== -1) return { line: '--' };
        if (sqlComments.indexOf(ext) !== -1) return { line: '--' };
        return { line: '//' };
    }

    // --- Text Transformation Commands ---

    function sortLinesAscending() {
        var view = getView();
        if (!view) return false;
        var text = getSelectedLines(view);
        var lines = text.split('\n');
        lines.sort(function(a, b) { return a.localeCompare(b); });
        replaceLines(view, lines.join('\n'));
        return true;
    }

    function sortLinesDescending() {
        var view = getView();
        if (!view) return false;
        var text = getSelectedLines(view);
        var lines = text.split('\n');
        lines.sort(function(a, b) { return b.localeCompare(a); });
        replaceLines(view, lines.join('\n'));
        return true;
    }

    function transformToUppercase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        replaceSelection(view, text.toUpperCase());
        return true;
    }

    function transformToLowercase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        replaceSelection(view, text.toLowerCase());
        return true;
    }

    function transformToTitleCase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        var titled = text.replace(/\b\w/g, function(c) { return c.toUpperCase(); });
        replaceSelection(view, titled);
        return true;
    }

    function trimTrailingWhitespace() {
        var view = getView();
        if (!view) return false;
        var doc = view.state.doc.toString();
        var trimmed = doc.replace(/[ \t]+$/gm, '');
        if (trimmed === doc) return true;
        view.dispatch({
            changes: { from: 0, to: view.state.doc.length, insert: trimmed },
        });
        return true;
    }

    function deleteEmptyLines() {
        var view = getView();
        if (!view) return false;
        var doc = view.state.doc.toString();
        var lines = doc.split('\n');
        var filtered = lines.filter(function(line) { return line.trim() !== ''; });
        var result = filtered.join('\n');
        if (result === doc) return true;
        view.dispatch({
            changes: { from: 0, to: view.state.doc.length, insert: result },
        });
        return true;
    }

    // --- Line Operations ---

    function duplicateLine() {
        var view = getView();
        if (!view) return false;
        var line = getCursorLine(view);
        var lineText = view.state.sliceDoc(line.from, line.to);
        view.dispatch({
            changes: { from: line.to, insert: '\n' + lineText },
        });
        return true;
    }

    function deleteLine() {
        var view = getView();
        if (!view) return false;
        var line = getCursorLine(view);
        var from = line.from;
        var to = line.to;
        if (to < view.state.doc.length) {
            to += 1;
        } else if (from > 0) {
            from -= 1;
        }
        view.dispatch({
            changes: { from: from, to: to },
        });
        return true;
    }

    function indentSelection() {
        var view = getView();
        if (!view) return false;
        var range = getLineRange(view);
        var text = view.state.sliceDoc(range.from, range.to);
        var indented = text.split('\n').map(function(line) { return '    ' + line; }).join('\n');
        view.dispatch({
            changes: { from: range.from, to: range.to, insert: indented },
        });
        return true;
    }

    function outdentSelection() {
        var view = getView();
        if (!view) return false;
        var range = getLineRange(view);
        var text = view.state.sliceDoc(range.from, range.to);
        var outdented = text.split('\n').map(function(line) {
            if (line.substring(0, 4) === '    ') return line.substring(4);
            if (line.charAt(0) === '\t') return line.substring(1);
            var match = line.match(/^( {1,3})/);
            if (match) return line.substring(match[1].length);
            return line;
        }).join('\n');
        view.dispatch({
            changes: { from: range.from, to: range.to, insert: outdented },
        });
        return true;
    }

    function toggleComment() {
        var view = getView();
        if (!view) return false;
        var filePath = null;
        if (typeof window.ClawIDEEditor !== 'undefined') {
            filePath = window.ClawIDEEditor.getCurrentFile();
        }
        var commentStyle = getCommentStyle(filePath);

        var range = getLineRange(view);
        var text = view.state.sliceDoc(range.from, range.to);
        var lines = text.split('\n');

        if (commentStyle.line) {
            var prefix = commentStyle.line + ' ';
            var allCommented = lines.every(function(line) {
                return line.trim() === '' || line.trimStart().indexOf(commentStyle.line) === 0;
            });

            var result;
            if (allCommented) {
                var commentRe = new RegExp('^(\\s*)' + commentStyle.line.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ' ?');
                result = lines.map(function(line) { return line.replace(commentRe, '$1'); }).join('\n');
            } else {
                result = lines.map(function(line) {
                    if (line.trim() === '') return line;
                    return prefix + line;
                }).join('\n');
            }
            view.dispatch({
                changes: { from: range.from, to: range.to, insert: result },
            });
        } else if (commentStyle.block) {
            var open = commentStyle.block[0] + ' ';
            var close = ' ' + commentStyle.block[1];
            var trimmedText = text.trim();
            if (trimmedText.indexOf(commentStyle.block[0]) === 0 && trimmedText.lastIndexOf(commentStyle.block[1]) === trimmedText.length - commentStyle.block[1].length) {
                var uncommented = trimmedText.substring(commentStyle.block[0].length).slice(0, -commentStyle.block[1].length).trim();
                view.dispatch({
                    changes: { from: range.from, to: range.to, insert: uncommented },
                });
            } else {
                view.dispatch({
                    changes: { from: range.from, to: range.to, insert: open + text + close },
                });
            }
        }
        return true;
    }

    function joinLines() {
        var view = getView();
        if (!view) return false;
        var range = getLineRange(view);
        var text = view.state.sliceDoc(range.from, range.to);
        var joined = text.split('\n').map(function(l) { return l.trim(); }).join(' ');
        view.dispatch({
            changes: { from: range.from, to: range.to, insert: joined },
        });
        return true;
    }

    function reverseLines() {
        var view = getView();
        if (!view) return false;
        var text = getSelectedLines(view);
        var lines = text.split('\n');
        lines.reverse();
        replaceLines(view, lines.join('\n'));
        return true;
    }

    function removeDuplicateLines() {
        var view = getView();
        if (!view) return false;
        var text = getSelectedLines(view);
        var lines = text.split('\n');
        var seen = {};
        var unique = lines.filter(function(line) {
            if (seen[line]) return false;
            seen[line] = true;
            return true;
        });
        replaceLines(view, unique.join('\n'));
        return true;
    }

    // --- Navigation ---

    function goToLine() {
        var view = getView();
        if (!view) return false;
        var totalLines = view.state.doc.lines;
        var input = prompt('Go to line (1-' + totalLines + '):');
        if (!input) return false;
        var lineNum = parseInt(input, 10);
        if (isNaN(lineNum) || lineNum < 1 || lineNum > totalLines) return false;
        var line = view.state.doc.line(lineNum);
        view.dispatch({
            selection: { anchor: line.from },
            scrollIntoView: true,
        });
        view.focus();
        return true;
    }

    // --- File Info Commands ---

    function copyFilePath() {
        var filePath = null;
        if (typeof window.ClawIDEEditor !== 'undefined') {
            filePath = window.ClawIDEEditor.getCurrentFile();
        }
        if (!filePath) return false;
        navigator.clipboard.writeText(filePath).catch(function(err) {
            console.error('Failed to copy file path:', err);
        });
        return true;
    }

    function copyRelativePath() {
        var filePath = null;
        if (typeof window.ClawIDEEditor !== 'undefined') {
            filePath = window.ClawIDEEditor.getCurrentFile();
        }
        if (!filePath) return false;
        navigator.clipboard.writeText(filePath).catch(function(err) {
            console.error('Failed to copy relative path:', err);
        });
        return true;
    }

    function copyFileName() {
        var filePath = null;
        if (typeof window.ClawIDEEditor !== 'undefined') {
            filePath = window.ClawIDEEditor.getCurrentFile();
        }
        if (!filePath) return false;
        var fileName = filePath.split('/').pop();
        navigator.clipboard.writeText(fileName).catch(function(err) {
            console.error('Failed to copy file name:', err);
        });
        return true;
    }

    // --- Selection Commands ---

    function selectAll() {
        var view = getView();
        if (!view) return false;
        view.dispatch({
            selection: { anchor: 0, head: view.state.doc.length },
        });
        return true;
    }

    function selectLine() {
        var view = getView();
        if (!view) return false;
        var line = getCursorLine(view);
        view.dispatch({
            selection: { anchor: line.from, head: line.to },
        });
        return true;
    }

    function selectWord() {
        var view = getView();
        if (!view) return false;
        var pos = view.state.selection.main.head;
        var line = view.state.doc.lineAt(pos);
        var lineText = line.text;
        var offsetInLine = pos - line.from;

        var wordStart = offsetInLine;
        var wordEnd = offsetInLine;
        while (wordStart > 0 && /\w/.test(lineText.charAt(wordStart - 1))) wordStart--;
        while (wordEnd < lineText.length && /\w/.test(lineText.charAt(wordEnd))) wordEnd++;

        if (wordStart === wordEnd) return false;
        view.dispatch({
            selection: { anchor: line.from + wordStart, head: line.from + wordEnd },
        });
        return true;
    }

    // --- Case Transformation Commands ---

    function transformToSnakeCase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        var snake = text
            .replace(/([a-z])([A-Z])/g, '$1_$2')
            .replace(/[\s\-]+/g, '_')
            .toLowerCase();
        replaceSelection(view, snake);
        return true;
    }

    function transformToCamelCase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        var camel = text
            .replace(/[-_\s]+(.)?/g, function(_, c) { return c ? c.toUpperCase() : ''; })
            .replace(/^./, function(c) { return c.toLowerCase(); });
        replaceSelection(view, camel);
        return true;
    }

    function transformToKebabCase() {
        var view = getView();
        if (!view) return false;
        var sel = getSelectionRange(view);
        if (sel.empty) return false;
        var text = getSelectedText(view);
        var kebab = text
            .replace(/([a-z])([A-Z])/g, '$1-$2')
            .replace(/[\s_]+/g, '-')
            .toLowerCase();
        replaceSelection(view, kebab);
        return true;
    }

    // --- Expose to global scope ---
    window.ClawIDECommands = {
        // Text Transformations
        sortLinesAscending: sortLinesAscending,
        sortLinesDescending: sortLinesDescending,
        transformToUppercase: transformToUppercase,
        transformToLowercase: transformToLowercase,
        transformToTitleCase: transformToTitleCase,
        transformToSnakeCase: transformToSnakeCase,
        transformToCamelCase: transformToCamelCase,
        transformToKebabCase: transformToKebabCase,
        trimTrailingWhitespace: trimTrailingWhitespace,
        deleteEmptyLines: deleteEmptyLines,

        // Line Operations
        duplicateLine: duplicateLine,
        deleteLine: deleteLine,
        indentSelection: indentSelection,
        outdentSelection: outdentSelection,
        toggleComment: toggleComment,
        joinLines: joinLines,
        reverseLines: reverseLines,
        removeDuplicateLines: removeDuplicateLines,

        // Navigation
        goToLine: goToLine,

        // File Info
        copyFilePath: copyFilePath,
        copyRelativePath: copyRelativePath,
        copyFileName: copyFileName,

        // Selection
        selectAll: selectAll,
        selectLine: selectLine,
        selectWord: selectWord,

        // File Operations
        newFile: function() {
            if (window.ClawIDENewFile) window.ClawIDENewFile.openModal('');
        },
        newFolder: function() {
            if (window.ClawIDENewFile) window.ClawIDENewFile.openFolderModal('');
        },
    };
})();
