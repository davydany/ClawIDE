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

function createEditor(container, content, filename, onDocChange, onSave) {
    var langCompartment = new Compartment();

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

    // Store compartment ref for language reconfiguration
    view._ccmuxLangCompartment = langCompartment;

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
    if (filename && view._ccmuxLangCompartment) {
        getLanguageExtension(filename).then(function(langExt) {
            if (view.dom.parentNode) {
                view.dispatch({
                    effects: view._ccmuxLangCompartment.reconfigure(langExt),
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

// Expose to global scope
window.CCMuxCodeMirror = {
    createEditor: createEditor,
    getContent: getContent,
    setContent: setContent,
    destroyEditor: destroyEditor,
};
