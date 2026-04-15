// CodeMirror 6 entry point — bundled via esbuild
import { EditorView, basicSetup } from 'codemirror';
import { EditorState, Compartment } from '@codemirror/state';
import { keymap } from '@codemirror/view';
import { oneDark } from '@codemirror/theme-one-dark';
import { languages } from '@codemirror/language-data';
import { javascript } from '@codemirror/lang-javascript';
import { python } from '@codemirror/lang-python';
import { html } from '@codemirror/lang-html';
import { css } from '@codemirror/lang-css';
import { json } from '@codemirror/lang-json';
import { markdown } from '@codemirror/lang-markdown';
import { yaml } from '@codemirror/lang-yaml';
import { sql } from '@codemirror/lang-sql';
import { search, openSearchPanel } from '@codemirror/search';
import { MergeView } from '@codemirror/merge';

// Map file extensions to language support
var extMap = {
    'js': javascript,
    'mjs': javascript,
    'cjs': javascript,
    'jsx': function() { return javascript({ jsx: true }); },
    'ts': function() { return javascript({ typescript: true }); },
    'tsx': function() { return javascript({ typescript: true, jsx: true }); },
    'py': python,
    'pyw': python,
    'html': html,
    'htm': html,
    'css': css,
    'scss': css,
    'less': css,
    'json': json,
    'jsonc': json,
    'md': markdown,
    'markdown': markdown,
    'yaml': yaml,
    'yml': yaml,
    'sql': sql,
    'go': null,       // fallback to language-data auto-detect
    'rs': null,
    'rb': null,
    'java': null,
    'c': null,
    'cpp': null,
    'h': null,
    'sh': null,
    'bash': null,
    'toml': null,
    'xml': null,
    'dockerfile': null,
};

function getLanguageForFilename(filename) {
    if (!filename) return [];

    var ext = filename.split('.').pop().toLowerCase();

    // Check explicit map first
    if (ext in extMap && extMap[ext] !== null) {
        var langFn = extMap[ext];
        if (typeof langFn === 'function') {
            var result = langFn();
            // If it returns a LanguageSupport directly (like python()), use it
            return [result];
        }
    }

    // Use @codemirror/language-data auto-detection as fallback
    // This covers Go, Rust, Java, C/C++, Shell, XML, TOML, Dockerfile, etc.
    var langDesc = languages.find(function(lang) {
        return lang.extensions.some(function(e) { return e === ext; }) ||
               lang.filename && lang.filename.test(filename);
    });
    if (langDesc) {
        return [langDesc.support || langDesc.load().then(function() { return []; })];
    }

    return [];
}

// Async version that loads language support (some langs are lazy-loaded)
function getLanguageExtension(filename) {
    if (!filename) return Promise.resolve([]);

    var ext = filename.split('.').pop().toLowerCase();

    // Check explicit map first
    if (ext in extMap && extMap[ext] !== null) {
        var langFn = extMap[ext];
        if (typeof langFn === 'function') {
            return Promise.resolve([langFn()]);
        }
    }

    // Use language-data for auto-detection (may load asynchronously)
    var langDesc = languages.find(function(lang) {
        return lang.extensions.some(function(e) { return e === ext; }) ||
               (lang.filename && lang.filename.test(filename));
    });
    if (langDesc) {
        return langDesc.load().then(function(support) {
            return [support];
        });
    }

    return Promise.resolve([]);
}

// --- Word Wrap State ---
var WRAP_STORAGE_KEY = 'editor.preferences.wordWrap';

function loadWordWrapPreference() {
    try {
        var stored = localStorage.getItem(WRAP_STORAGE_KEY);
        if (stored === null) return true; // default: enabled
        return stored === 'true';
    } catch (e) {
        return true;
    }
}

function saveWordWrapPreference(enabled) {
    try {
        localStorage.setItem(WRAP_STORAGE_KEY, String(enabled));
    } catch (e) {
        // localStorage unavailable
    }
}

function toggleWordWrap(view) {
    if (!view || !view._clawIDEWrapCompartment) return false;
    var newState = !getWordWrapState(view);
    view.dispatch({
        effects: view._clawIDEWrapCompartment.reconfigure(
            newState ? EditorView.lineWrapping : []
        ),
    });
    view._clawIDEWordWrap = newState;
    saveWordWrapPreference(newState);
    return newState;
}

function getWordWrapState(view) {
    if (!view || !view._clawIDEWrapCompartment) return false;
    return !!view._clawIDEWordWrap;
}

function createEditor(container, content, filename, onDocChange, onSave) {
    var langCompartment = new Compartment();
    var wrapCompartment = new Compartment();
    var wordWrapEnabled = loadWordWrapPreference();

    var saveKeymap = onSave ? keymap.of([{
        key: 'Mod-s',
        run: function() {
            onSave();
            return true;
        },
    }]) : [];

    var updateListener = onDocChange ? EditorView.updateListener.of(function(update) {
        if (update.docChanged) {
            onDocChange(update);
        }
    }) : [];

    var extensions = [
        basicSetup,
        oneDark,
        search(),
        langCompartment.of([]),
        wrapCompartment.of(wordWrapEnabled ? EditorView.lineWrapping : []),
        saveKeymap,
        updateListener,
        EditorView.theme({
            '&': { height: '100%' },
            '.cm-scroller': { overflow: 'auto' },
            '.cm-content': { fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace' },
        }),
    ];

    var state = EditorState.create({
        doc: content || '',
        extensions: extensions,
    });

    var view = new EditorView({
        state: state,
        parent: container,
    });

    // Store compartment refs for runtime reconfiguration
    view._clawIDELangCompartment = langCompartment;
    view._clawIDEWrapCompartment = wrapCompartment;
    view._clawIDEWordWrap = wordWrapEnabled;

    // Load language asynchronously
    getLanguageExtension(filename).then(function(langExt) {
        if (langExt.length > 0 && view.dom.parentNode) {
            view.dispatch({
                effects: langCompartment.reconfigure(langExt),
            });
        }
    });

    return view;
}

function getContent(view) {
    return view.state.doc.toString();
}

function setContent(view, text, filename) {
    view.dispatch({
        changes: {
            from: 0,
            to: view.state.doc.length,
            insert: text,
        },
    });

    // Reconfigure language if filename changed
    if (filename && view._clawIDELangCompartment) {
        getLanguageExtension(filename).then(function(langExt) {
            if (view.dom.parentNode) {
                view.dispatch({
                    effects: view._clawIDELangCompartment.reconfigure(langExt),
                });
            }
        });
    }
}

function destroyEditor(view) {
    if (view) {
        view.destroy();
    }
}

// --- Scroll helpers (for markdown preview sync) ---

function getTopVisibleLine(view) {
    try {
        var offset = view.scrollDOM.scrollTop - view.documentTop;
        var block = view.lineBlockAtHeight(offset);
        return view.state.doc.lineAt(block.from).number;
    } catch (e) {
        return 1;
    }
}

function scrollToLine(view, line) {
    if (!view) return;
    var doc = view.state.doc;
    if (line < 1) line = 1;
    if (line > doc.lines) line = doc.lines;
    var pos = doc.line(line).from;
    view.dispatch({ effects: EditorView.scrollIntoView(pos, { y: 'start' }) });
}

function getScrollDOM(view) {
    return view && view.scrollDOM;
}

// --- MergeView support ---

function createMergeView(container, docA, docB, filename, options) {
    return getLanguageExtension(filename || '').then(function(langExt) {
        var readOnlyBase = [
            basicSetup,
            oneDark,
            search(),
            EditorState.readOnly.of(true),
            EditorView.editable.of(false),
            EditorView.theme({
                '&': { height: '100%' },
                '.cm-scroller': { overflow: 'auto' },
                '.cm-content': { fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace' },
            }),
        ];
        var mv = new MergeView({
            a: { doc: docA || '', extensions: readOnlyBase.concat(langExt) },
            b: { doc: docB || '', extensions: readOnlyBase.concat(langExt) },
            parent: container,
            orientation: (options && options.orientation) || 'a-b',
            highlightChanges: true,
            gutter: true,
            collapseUnchanged: { margin: 3, minSize: 4 },
        });
        return mv;
    });
}

function destroyMergeView(mv) {
    if (mv) {
        mv.destroy();
    }
}

// Expose to global scope
window.ClawIDECodeMirror = {
    createEditor: createEditor,
    getContent: getContent,
    setContent: setContent,
    destroyEditor: destroyEditor,
    toggleWordWrap: toggleWordWrap,
    getWordWrapState: getWordWrapState,
    createMergeView: createMergeView,
    destroyMergeView: destroyMergeView,
    getTopVisibleLine: getTopVisibleLine,
    scrollToLine: scrollToLine,
    getScrollDOM: getScrollDOM,
};
