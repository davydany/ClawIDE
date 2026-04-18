// ClawIDE PromptForge
// Global prompt library with folder tree, markdown editor, Jinja (Nunjucks)
// compilation, and versioned compiled-output history.
(function() {
    'use strict';

    var API = '/api/promptforge';

    // ── State ────────────────────────────────────────────────────────────

    var modalEl = null;
    var state = {
        folders: [],              // [{id, name, parent_id, order, ...}]
        prompts: [],              // summaries (no body)
        expandedFolders: {},      // folder_id -> true
        selected: null,           // full prompt with body
        editorView: null,         // CodeMirror view
        previewMode: 'off',       // 'off' | 'side' | 'preview'
        previewHeadings: [],
        tocPanel: null,
        versions: [],             // [{id, title, compiled_at}]
        versionPreviewEl: null,   // for the read-only compiled preview overlay
        dirty: false,
        saveTimer: null
    };

    var VAR_TYPES = [
        { value: 'string',  label: 'String' },
        { value: 'text',    label: 'Text (multi-line)' },
        { value: 'number',  label: 'Number' },
        { value: 'boolean', label: 'Boolean' },
        { value: 'select',  label: 'Select (enum)' },
        { value: 'date',    label: 'Date' }
    ];

    // ── Public API ───────────────────────────────────────────────────────

    function open() {
        if (modalEl) return;
        createModal();
        loadTree();
    }

    function close() {
        if (!modalEl) return;
        if (state.dirty && !window.confirm('Discard unsaved changes to this prompt?')) {
            return;
        }
        teardownEditor();
        modalEl.remove();
        modalEl = null;
        state.selected = null;
        state.versions = [];
        state.dirty = false;
    }

    // ── Modal shell ──────────────────────────────────────────────────────

    function createModal() {
        modalEl = document.createElement('div');
        modalEl.id = 'promptforge-modal';
        modalEl.className = 'fixed inset-0 z-[200] flex items-center justify-center';

        modalEl.innerHTML = ''
            + '<div class="absolute inset-0 bg-black/70 backdrop-blur-sm" id="pf-backdrop"></div>'
            + '<div class="relative w-[92vw] max-w-6xl h-[86vh] bg-surface-base border border-th-border-strong rounded-xl shadow-2xl flex flex-col overflow-hidden">'
            + '  <div class="flex items-center justify-between px-5 py-3 border-b border-th-border">'
            + '    <h2 class="text-base font-semibold text-th-text-primary flex items-center gap-2">'
            + '      <svg class="w-5 h-5 text-accent-text" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>'
            + '      Prompt Forge'
            + '    </h2>'
            + '    <button id="pf-close-btn" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors" title="Close (Esc)">'
            + '      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '    </button>'
            + '  </div>'
            + '  <div class="flex flex-1 min-h-0">'
            // Tree pane
            + '    <div class="w-1/5 min-w-[200px] max-w-[320px] border-r border-th-border flex flex-col bg-surface-base/60">'
            + '      <div class="flex items-center gap-1 px-2 py-2 border-b border-th-border">'
            + '        <button id="pf-new-folder-btn" class="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 text-[11px] text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors" title="New folder">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>'
            + '          Folder'
            + '        </button>'
            + '        <button id="pf-new-prompt-btn" class="flex-1 flex items-center justify-center gap-1 px-2 py-1.5 text-[11px] text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors" title="New prompt">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '          Prompt'
            + '        </button>'
            + '      </div>'
            + '      <div id="pf-tree" class="flex-1 overflow-y-auto px-1 py-2 text-xs"></div>'
            + '    </div>'
            // Editor pane
            + '    <div class="flex-1 flex flex-col min-w-0">'
            + '      <div id="pf-editor-pane" class="flex flex-1 min-h-0 items-center justify-center text-th-text-faint text-sm">'
            + '        Select a prompt or create a new one.'
            + '      </div>'
            + '    </div>'
            + '  </div>'
            + '</div>';

        document.body.appendChild(modalEl);

        document.getElementById('pf-backdrop').addEventListener('click', close);
        document.getElementById('pf-close-btn').addEventListener('click', close);
        document.getElementById('pf-new-folder-btn').addEventListener('click', function() { createFolderPrompt(''); });
        document.getElementById('pf-new-prompt-btn').addEventListener('click', function() { createPromptPrompt(''); });

        modalEl.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                close();
                e.stopPropagation();
            } else if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                e.preventDefault();
                savePrompt();
            }
        });
    }

    // ── Tree loading + rendering ─────────────────────────────────────────

    function loadTree() {
        Promise.all([
            fetch(API + '/folders').then(function(r) { return r.json(); }),
            fetch(API + '/prompts').then(function(r) { return r.json(); })
        ]).then(function(results) {
            state.folders = results[0] || [];
            state.prompts = results[1] || [];
            renderTree();
        }).catch(function(err) {
            console.error('PromptForge: failed to load tree', err);
            toast('Failed to load prompt library');
        });
    }

    function renderTree() {
        var container = document.getElementById('pf-tree');
        if (!container) return;

        if (state.folders.length === 0 && state.prompts.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-center py-6 px-2">'
                + '<div class="mb-2">No prompts yet</div>'
                + '<div class="text-[10px]">Create a folder or prompt to get started.</div>'
                + '</div>';
            return;
        }

        container.innerHTML = '';
        renderLevel(container, '', 0);
    }

    function renderLevel(container, parentID, depth) {
        var children = state.folders.filter(function(f) { return f.parent_id === parentID; });
        children.sort(function(a, b) {
            return (a.order - b.order) || a.name.localeCompare(b.name);
        });
        for (var i = 0; i < children.length; i++) {
            renderFolderNode(container, children[i], depth);
        }

        var rootPrompts = state.prompts.filter(function(p) { return (p.folder_id || '') === parentID; });
        rootPrompts.sort(function(a, b) { return a.title.localeCompare(b.title); });
        for (var j = 0; j < rootPrompts.length; j++) {
            renderPromptNode(container, rootPrompts[j], depth);
        }
    }

    function renderFolderNode(container, folder, depth) {
        var expanded = !!state.expandedFolders[folder.id];
        var row = document.createElement('div');
        row.className = 'pf-tree-item group flex items-center gap-1 py-1 pr-1 rounded cursor-pointer text-th-text-tertiary hover:bg-surface-raised';
        row.style.paddingLeft = (4 + depth * 10) + 'px';
        row.innerHTML = ''
            + '<svg class="w-3 h-3 text-th-text-muted flex-shrink-0 transition-transform ' + (expanded ? 'rotate-90' : '') + '" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>'
            + '<svg class="w-3.5 h-3.5 text-accent-text flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>'
            + '<span class="truncate flex-1">' + escapeHTML(folder.name) + '</span>'
            + '<span class="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">'
            + '  <button data-action="new-prompt" title="New prompt in this folder" class="p-0.5 hover:text-th-text-primary"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg></button>'
            + '  <button data-action="new-folder" title="New subfolder" class="p-0.5 hover:text-th-text-primary"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg></button>'
            + '  <button data-action="rename-folder" title="Rename" class="p-0.5 hover:text-th-text-primary"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg></button>'
            + '  <button data-action="delete-folder" title="Delete" class="p-0.5 hover:text-red-400"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M1 7h22M9 7V4a2 2 0 012-2h2a2 2 0 012 2v3"/></svg></button>'
            + '</span>';

        row.addEventListener('click', function(e) {
            var btn = e.target.closest('button[data-action]');
            if (btn) {
                e.stopPropagation();
                var act = btn.getAttribute('data-action');
                if (act === 'new-prompt') createPromptPrompt(folder.id);
                else if (act === 'new-folder') createFolderPrompt(folder.id);
                else if (act === 'rename-folder') renameFolder(folder);
                else if (act === 'delete-folder') deleteFolder(folder);
                return;
            }
            state.expandedFolders[folder.id] = !state.expandedFolders[folder.id];
            renderTree();
        });
        container.appendChild(row);

        if (expanded) {
            renderLevel(container, folder.id, depth + 1);
        }
    }

    function renderPromptNode(container, p, depth) {
        var row = document.createElement('div');
        var isActive = state.selected && state.selected.id === p.id;
        row.className = 'pf-tree-item group flex items-center gap-1 py-1 pr-1 rounded cursor-pointer ' +
            (isActive ? 'bg-accent/25 text-th-text-primary' : 'text-th-text-tertiary hover:bg-surface-raised');
        row.style.paddingLeft = (4 + depth * 10 + 14) + 'px';
        var badge = (p.type === 'jinja')
            ? '<span class="text-[9px] px-1 py-0 rounded bg-purple-900/60 text-purple-300 flex-shrink-0">J</span>'
            : '<span class="text-[9px] px-1 py-0 rounded bg-slate-800/60 text-slate-400 flex-shrink-0">P</span>';
        row.innerHTML = ''
            + '<svg class="w-3.5 h-3.5 text-th-text-muted flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>'
            + badge
            + '<span class="truncate flex-1">' + escapeHTML(p.title) + '</span>'
            + '<span class="flex items-center gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">'
            + '  <button data-action="rename-prompt" title="Rename" class="p-0.5 hover:text-th-text-primary"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg></button>'
            + '  <button data-action="delete-prompt" title="Delete" class="p-0.5 hover:text-red-400"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6M1 7h22M9 7V4a2 2 0 012-2h2a2 2 0 012 2v3"/></svg></button>'
            + '</span>';

        row.addEventListener('click', function(e) {
            var btn = e.target.closest('button[data-action]');
            if (btn) {
                e.stopPropagation();
                var act = btn.getAttribute('data-action');
                if (act === 'rename-prompt') renamePrompt(p);
                else if (act === 'delete-prompt') deletePrompt(p);
                return;
            }
            selectPrompt(p.id);
        });
        container.appendChild(row);
    }

    // ── Folder actions ───────────────────────────────────────────────────

    function createFolderPrompt(parentID) {
        var name = window.prompt('Folder name (letters, numbers, dots, hyphens, underscores):');
        if (!name) return;
        fetch(API + '/folders', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, parent_id: parentID || '' })
        }).then(handleJSON).then(function(folder) {
            if (parentID) state.expandedFolders[parentID] = true;
            state.expandedFolders[folder.id] = true;
            return loadTree();
        }).catch(showErr);
    }

    function renameFolder(folder) {
        var name = window.prompt('Rename folder to:', folder.name);
        if (!name || name === folder.name) return;
        fetch(API + '/folders/' + encodeURIComponent(folder.id), {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name })
        }).then(handleJSON).then(function() { return loadTree(); }).catch(showErr);
    }

    function deleteFolder(folder) {
        if (!window.confirm('Delete folder "' + folder.name + '"? This only works if the folder is empty.')) return;
        fetch(API + '/folders/' + encodeURIComponent(folder.id), { method: 'DELETE' })
            .then(function(r) {
                if (r.status === 409) {
                    if (!window.confirm('Folder is not empty. Delete it AND all prompts + subfolders inside? This cannot be undone.')) {
                        throw new Error('cancelled');
                    }
                    return fetch(API + '/folders/' + encodeURIComponent(folder.id) + '?cascade=true', { method: 'DELETE' });
                }
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r;
            })
            .then(function(r) {
                if (r && !r.ok) return r.text().then(function(t) { throw new Error(t); });
                if (state.selected && state.selected.folder_id === folder.id) {
                    state.selected = null;
                    renderEditor();
                }
                return loadTree();
            }).catch(function(err) {
                if (err && err.message !== 'cancelled') showErr(err);
            });
    }

    // ── Prompt actions ───────────────────────────────────────────────────

    function createPromptPrompt(folderID) {
        var title = window.prompt('Prompt title (letters, numbers, dots, hyphens, underscores):');
        if (!title) return;
        fetch(API + '/prompts', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                folder_id: folderID || '',
                title: title,
                type: 'plain',
                variables: [],
                content: '# ' + title + '\n\nWrite your prompt here.\n'
            })
        }).then(handleJSON).then(function(p) {
            if (folderID) state.expandedFolders[folderID] = true;
            return loadTree().then(function() { return selectPrompt(p.id); });
        }).catch(showErr);
    }

    function renamePrompt(p) {
        var title = window.prompt('Rename prompt to:', p.title);
        if (!title || title === p.title) return;
        fetch(API + '/prompts/' + encodeURIComponent(p.id), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title: title })
        }).then(handleJSON).then(function() {
            return loadTree().then(function() {
                if (state.selected && state.selected.id === p.id) {
                    return selectPrompt(p.id);
                }
            });
        }).catch(showErr);
    }

    function deletePrompt(p) {
        if (!window.confirm('Delete prompt "' + p.title + '" and all its compiled versions? This cannot be undone.')) return;
        fetch(API + '/prompts/' + encodeURIComponent(p.id), { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                if (state.selected && state.selected.id === p.id) {
                    state.selected = null;
                    renderEditor();
                }
                return loadTree();
            }).catch(showErr);
    }

    function selectPrompt(id) {
        if (state.dirty && !window.confirm('Discard unsaved changes to this prompt?')) {
            return Promise.resolve();
        }
        teardownEditor();
        return fetch(API + '/prompts/' + encodeURIComponent(id))
            .then(handleJSON)
            .then(function(p) {
                state.selected = p;
                state.dirty = false;
                state.previewMode = 'off';
                renderEditor();
                renderTree();
                return loadVersions();
            }).catch(showErr);
    }

    // ── Editor pane ──────────────────────────────────────────────────────

    function renderEditor() {
        var pane = document.getElementById('pf-editor-pane');
        if (!pane) return;

        if (!state.selected) {
            pane.className = 'flex flex-1 min-h-0 items-center justify-center text-th-text-faint text-sm';
            pane.innerHTML = 'Select a prompt or create a new one.';
            return;
        }

        var p = state.selected;
        pane.className = 'flex-1 flex flex-col min-h-0';
        pane.innerHTML = ''
            // Header: title + type + save + actions
            + '<div class="flex items-center justify-between gap-2 px-4 py-2.5 border-b border-th-border flex-shrink-0">'
            + '  <div class="flex items-center gap-2 flex-1 min-w-0">'
            + '    <input id="pf-title" type="text" value="' + escapeAttr(p.title) + '"'
            + '           class="flex-1 min-w-0 bg-surface-raised border border-th-border-strong rounded px-2.5 py-1.5 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '    <select id="pf-type" class="bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 text-xs text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '      <option value="plain"' + (p.type === 'plain' ? ' selected' : '') + '>Plain</option>'
            + '      <option value="jinja"' + (p.type === 'jinja' ? ' selected' : '') + '>Jinja template</option>'
            + '    </select>'
            + '  </div>'
            + '  <div class="flex items-center gap-1">'
            + '    <div class="editor-preview-group flex items-center gap-0.5 mr-1">'
            + '      <button id="pf-view-edit" class="editor-control-btn px-2 py-1 text-[11px] rounded hover:bg-surface-raised" title="Edit only">Edit</button>'
            + '      <button id="pf-view-side" class="editor-control-btn px-2 py-1 text-[11px] rounded hover:bg-surface-raised" title="Side-by-side">Side</button>'
            + '      <button id="pf-view-full" class="editor-control-btn px-2 py-1 text-[11px] rounded hover:bg-surface-raised" title="Full preview">Full</button>'
            + '    </div>'
            + '    <button id="pf-compile-btn" class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded transition-colors font-medium flex items-center gap-1" title="Compile with variables">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>'
            + '      Compile'
            + '    </button>'
            + '    <button id="pf-insert-btn" class="px-2.5 py-1.5 text-xs text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors" title="Insert into focused terminal">Insert</button>'
            + '    <button id="pf-copy-btn" class="px-2.5 py-1.5 text-xs text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors" title="Copy to clipboard">Copy</button>'
            + '    <button id="pf-save-btn" class="px-3 py-1.5 text-xs bg-accent/70 hover:bg-accent text-th-text-primary rounded transition-colors font-medium" title="Save (Cmd+S)">Save</button>'
            + '  </div>'
            + '</div>'
            // Variables drawer
            + '<div id="pf-vars-drawer" class="border-b border-th-border bg-surface-base/60 hidden"></div>'
            + '<div class="px-4 py-1.5 text-[10px] text-th-text-faint flex items-center justify-between border-b border-th-border">'
            + '  <button id="pf-vars-toggle" class="hover:text-th-text-tertiary uppercase tracking-wider">Variables (' + (p.variables || []).length + ')</button>'
            + '  <span id="pf-dirty-indicator" class="text-accent-text opacity-0">● unsaved</span>'
            + '</div>'
            // Editor + preview area
            + '<div id="pf-editor-wrapper" class="editor-tab-wrapper flex-1 flex min-h-0 relative"></div>'
            // Versions panel
            + '<div id="pf-versions" class="border-t border-th-border max-h-[35%] overflow-y-auto flex-shrink-0"></div>';

        document.getElementById('pf-title').addEventListener('change', function() { markDirty(); });
        document.getElementById('pf-type').addEventListener('change', function() { markDirty(); renderVariablesDrawer(); });
        document.getElementById('pf-vars-toggle').addEventListener('click', function() {
            var drawer = document.getElementById('pf-vars-drawer');
            if (!drawer) return;
            if (drawer.classList.contains('hidden')) {
                drawer.classList.remove('hidden');
                renderVariablesDrawer();
            } else {
                drawer.classList.add('hidden');
            }
        });
        document.getElementById('pf-view-edit').addEventListener('click', function() { setPreviewMode('off'); });
        document.getElementById('pf-view-side').addEventListener('click', function() { setPreviewMode('side'); });
        document.getElementById('pf-view-full').addEventListener('click', function() { setPreviewMode('preview'); });
        document.getElementById('pf-compile-btn').addEventListener('click', openCompileDialog);
        document.getElementById('pf-insert-btn').addEventListener('click', function() { insertIntoTerminal(readEditorContent()); });
        document.getElementById('pf-copy-btn').addEventListener('click', function() { copyToClipboard(readEditorContent()); });
        document.getElementById('pf-save-btn').addEventListener('click', savePrompt);

        mountEditor();
        setPreviewMode(state.previewMode);
        renderVersions();
    }

    function mountEditor() {
        var wrapper = document.getElementById('pf-editor-wrapper');
        if (!wrapper) return;
        wrapper.innerHTML = '';

        if (!window.ClawIDECodeMirror) {
            // Fallback to a textarea if the CodeMirror bundle failed to load.
            var ta = document.createElement('textarea');
            ta.id = 'pf-fallback-editor';
            ta.className = 'cm-editor-wrap w-full h-full p-3 bg-surface-base text-th-text-primary font-mono text-sm resize-none focus:outline-none';
            ta.value = state.selected ? (state.selected.content || '') : '';
            ta.addEventListener('input', onEditorChange);
            wrapper.appendChild(ta);
            return;
        }

        state.editorView = window.ClawIDECodeMirror.createEditor(
            wrapper,
            (state.selected && state.selected.title ? state.selected.title : 'prompt') + '.md',
            state.selected ? (state.selected.content || '') : '',
            { onChange: onEditorChange }
        );
    }

    function teardownEditor() {
        state.previewHeadings = [];
        state.tocPanel = null;
        if (state.editorView && window.ClawIDECodeMirror) {
            try { window.ClawIDECodeMirror.destroyEditor(state.editorView); } catch (e) {}
        }
        state.editorView = null;
    }

    function onEditorChange() {
        markDirty();
        // Debounced preview refresh
        if (state.previewMode === 'off') return;
        if (state.previewRefresh) clearTimeout(state.previewRefresh);
        state.previewRefresh = setTimeout(renderPreview, 150);
    }

    function readEditorContent() {
        if (!state.editorView) {
            var ta = document.getElementById('pf-fallback-editor');
            return ta ? ta.value : (state.selected ? state.selected.content : '');
        }
        if (window.ClawIDECodeMirror && window.ClawIDECodeMirror.getContent) {
            return window.ClawIDECodeMirror.getContent(state.editorView);
        }
        return state.selected ? state.selected.content : '';
    }

    function markDirty() {
        state.dirty = true;
        var ind = document.getElementById('pf-dirty-indicator');
        if (ind) ind.style.opacity = '1';
    }

    function clearDirty() {
        state.dirty = false;
        var ind = document.getElementById('pf-dirty-indicator');
        if (ind) ind.style.opacity = '0';
    }

    // ── Variables drawer ─────────────────────────────────────────────────

    function renderVariablesDrawer() {
        var drawer = document.getElementById('pf-vars-drawer');
        if (!drawer) return;
        var vars = (state.selected && state.selected.variables) || [];
        var type = document.getElementById('pf-type') ? document.getElementById('pf-type').value : 'plain';

        var html = '<div class="px-4 py-3 space-y-2">';
        if (type !== 'jinja') {
            html += '<div class="text-[11px] text-th-text-faint italic">Variables only apply when prompt type is <code class="text-th-text-tertiary">jinja</code>.</div>';
        }
        for (var i = 0; i < vars.length; i++) {
            html += variableRowHTML(vars[i], i);
        }
        html += '<button id="pf-add-var" class="flex items-center gap-1 px-2 py-1 text-[11px] text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors">'
            + '<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + 'Add variable</button>';
        html += '</div>';
        drawer.innerHTML = html;

        drawer.querySelectorAll('[data-var-field]').forEach(function(el) {
            el.addEventListener('change', onVariableFieldChange);
            el.addEventListener('input', onVariableFieldChange);
        });
        drawer.querySelectorAll('[data-remove-var]').forEach(function(btn) {
            btn.addEventListener('click', function() {
                var idx = parseInt(btn.getAttribute('data-remove-var'), 10);
                state.selected.variables.splice(idx, 1);
                markDirty();
                renderVariablesDrawer();
                updateVarCount();
            });
        });
        var addBtn = document.getElementById('pf-add-var');
        if (addBtn) addBtn.addEventListener('click', function() {
            if (!state.selected.variables) state.selected.variables = [];
            state.selected.variables.push({ name: 'var' + (state.selected.variables.length + 1), type: 'string', label: '', default: '', options: [], required: false });
            markDirty();
            renderVariablesDrawer();
            updateVarCount();
        });
    }

    function variableRowHTML(v, idx) {
        var typeOpts = VAR_TYPES.map(function(t) {
            return '<option value="' + t.value + '"' + (v.type === t.value ? ' selected' : '') + '>' + escapeHTML(t.label) + '</option>';
        }).join('');
        var optionsInput = v.type === 'select'
            ? '<input data-var-field data-idx="' + idx + '" data-key="options" type="text" value="' + escapeAttr((v.options || []).join(',')) + '" placeholder="option1, option2" class="pf-variable-row-input flex-1 bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-[11px] text-th-text-primary focus:outline-none focus:border-accent-border">'
            : '<span class="flex-1"></span>';
        return ''
            + '<div class="pf-variable-row flex items-center gap-1.5">'
            + '  <input data-var-field data-idx="' + idx + '" data-key="name" type="text" value="' + escapeAttr(v.name || '') + '" placeholder="name" class="w-28 bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-[11px] font-mono text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '  <select data-var-field data-idx="' + idx + '" data-key="type" class="w-32 bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-[11px] text-th-text-primary focus:outline-none focus:border-accent-border">' + typeOpts + '</select>'
            + '  <input data-var-field data-idx="' + idx + '" data-key="label" type="text" value="' + escapeAttr(v.label || '') + '" placeholder="Label" class="w-32 bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-[11px] text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '  <input data-var-field data-idx="' + idx + '" data-key="default" type="text" value="' + escapeAttr(v.default || '') + '" placeholder="default" class="w-32 bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-[11px] text-th-text-primary focus:outline-none focus:border-accent-border">'
            + optionsInput
            + '  <label class="flex items-center gap-1 text-[10px] text-th-text-faint"><input data-var-field data-idx="' + idx + '" data-key="required" type="checkbox"' + (v.required ? ' checked' : '') + '> required</label>'
            + '  <button data-remove-var="' + idx + '" title="Remove" class="p-1 text-th-text-muted hover:text-red-400"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg></button>'
            + '</div>';
    }

    function onVariableFieldChange(e) {
        var el = e.currentTarget;
        var idx = parseInt(el.getAttribute('data-idx'), 10);
        var key = el.getAttribute('data-key');
        var v = state.selected.variables[idx];
        if (!v) return;
        if (key === 'required') {
            v.required = el.checked;
        } else if (key === 'options') {
            v.options = el.value.split(',').map(function(s) { return s.trim(); }).filter(Boolean);
        } else if (key === 'type') {
            v.type = el.value;
            markDirty();
            renderVariablesDrawer(); // options input appears/disappears
            return;
        } else {
            v[key] = el.value;
        }
        markDirty();
    }

    function updateVarCount() {
        var toggle = document.getElementById('pf-vars-toggle');
        if (!toggle) return;
        var n = (state.selected.variables || []).length;
        toggle.textContent = 'Variables (' + n + ')';
    }

    // ── Save prompt ──────────────────────────────────────────────────────

    function savePrompt() {
        if (!state.selected) return;
        var titleInput = document.getElementById('pf-title');
        var typeInput = document.getElementById('pf-type');
        if (!titleInput || !typeInput) return;

        var payload = {
            title: titleInput.value.trim(),
            type: typeInput.value,
            variables: state.selected.variables || [],
            content: readEditorContent()
        };
        fetch(API + '/prompts/' + encodeURIComponent(state.selected.id), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        }).then(handleJSON).then(function(p) {
            state.selected = p;
            clearDirty();
            toast('Saved');
            return loadTree();
        }).catch(showErr);
    }

    // ── Preview modes ────────────────────────────────────────────────────

    function setPreviewMode(mode) {
        state.previewMode = mode;
        var wrapper = document.getElementById('pf-editor-wrapper');
        if (!wrapper) return;

        // Reset classes + TOC
        wrapper.classList.remove('preview-side', 'preview-only');
        if (state.tocPanel && state.tocPanel.parentNode) {
            state.tocPanel.parentNode.removeChild(state.tocPanel);
            state.tocPanel = null;
        }
        var existingHandle = wrapper.querySelector('.editor-preview-resize-handle');
        if (existingHandle) existingHandle.remove();
        var existingPreview = wrapper.querySelector('.md-preview-container');
        if (existingPreview) existingPreview.remove();

        // Ensure .cm-editor-wrap wraps the editor (for fallback, the textarea already has the class)
        var cmWrap = wrapper.querySelector('.cm-editor-wrap');
        if (!cmWrap && state.editorView && state.editorView.dom) {
            // CodeMirror appended the DOM directly; wrap it.
            var existing = state.editorView.dom;
            cmWrap = document.createElement('div');
            cmWrap.className = 'cm-editor-wrap';
            wrapper.insertBefore(cmWrap, existing);
            cmWrap.appendChild(existing);
        }

        highlightPreviewButton(mode);

        if (mode === 'off') {
            if (cmWrap) cmWrap.style.display = '';
            return;
        }

        if (mode === 'side') {
            wrapper.classList.add('preview-side');
            if (cmWrap) cmWrap.style.display = '';
            var handle = document.createElement('div');
            handle.className = 'editor-preview-resize-handle';
            if (cmWrap) {
                wrapper.insertBefore(handle, cmWrap.nextSibling);
            }
            attachResize(handle, wrapper);
        } else if (mode === 'preview') {
            wrapper.classList.add('preview-only');
            if (cmWrap) cmWrap.style.display = 'none';
        }

        var preview = document.createElement('div');
        preview.className = 'md-preview-container note-markdown-preview text-sm text-th-text-tertiary';
        wrapper.appendChild(preview);

        renderPreview();
    }

    function highlightPreviewButton(mode) {
        ['edit', 'side', 'full'].forEach(function(key) {
            var btn = document.getElementById('pf-view-' + key);
            if (!btn) return;
            var active = (mode === 'off' && key === 'edit') || (mode === 'side' && key === 'side') || (mode === 'preview' && key === 'full');
            btn.classList.toggle('active', active);
            btn.classList.toggle('bg-accent/30', active);
            btn.classList.toggle('text-th-text-primary', active);
        });
    }

    function renderPreview() {
        var wrapper = document.getElementById('pf-editor-wrapper');
        if (!wrapper) return;
        var container = wrapper.querySelector('.md-preview-container');
        if (!container) return;
        var text = readEditorContent();
        var result = window.ClawIDEMarkdown
            ? window.ClawIDEMarkdown.renderInto(container, text)
            : (container.textContent = text || '', { headings: [] });
        state.previewHeadings = (result && result.headings) || [];

        if (state.previewMode === 'preview') {
            ensureTocPanel(wrapper);
        }
    }

    function ensureTocPanel(wrapper) {
        if (state.tocPanel && state.tocPanel.parentNode === wrapper) {
            refreshTocContents();
            return;
        }
        var panel = document.createElement('aside');
        panel.className = 'md-preview-toc-panel';
        panel.innerHTML = '<div class="md-toc-title">On this page</div><nav><ul></ul></nav>';
        wrapper.insertBefore(panel, wrapper.firstChild);
        state.tocPanel = panel;
        refreshTocContents();
    }

    function refreshTocContents() {
        if (!state.tocPanel) return;
        var ul = state.tocPanel.querySelector('ul');
        if (!ul) return;
        ul.innerHTML = '';
        if (state.previewHeadings.length < 2) {
            state.tocPanel.style.display = 'none';
            return;
        }
        state.tocPanel.style.display = '';
        state.previewHeadings.forEach(function(h) {
            var li = document.createElement('li');
            var a = document.createElement('a');
            a.href = '#' + h.id;
            a.dataset.headingId = h.id;
            a.style.setProperty('--toc-indent', ((h.level - 1) * 12) + 'px');
            a.textContent = h.text;
            a.addEventListener('click', function(e) {
                e.preventDefault();
                var el = document.getElementById(h.id);
                if (el) el.scrollIntoView({ behavior: 'smooth', block: 'start' });
            });
            li.appendChild(a);
            ul.appendChild(li);
        });
    }

    function attachResize(handle, wrapper) {
        handle.addEventListener('mousedown', function(e) {
            e.preventDefault();
            var cmWrap = wrapper.querySelector('.cm-editor-wrap');
            if (!cmWrap) return;
            var startX = e.clientX;
            var startWidth = cmWrap.offsetWidth;
            var totalWidth = wrapper.offsetWidth;
            document.body.style.cursor = 'col-resize';
            document.body.style.userSelect = 'none';
            function move(ev) {
                var ratio = (startWidth + ev.clientX - startX) / totalWidth;
                ratio = Math.max(0.15, Math.min(0.85, ratio));
                cmWrap.style.flex = '0 0 ' + (ratio * 100) + '%';
            }
            function up() {
                document.body.style.cursor = '';
                document.body.style.userSelect = '';
                document.removeEventListener('mousemove', move);
                document.removeEventListener('mouseup', up);
            }
            document.addEventListener('mousemove', move);
            document.addEventListener('mouseup', up);
        });
    }

    // ── Compile flow ─────────────────────────────────────────────────────

    function openCompileDialog() {
        if (!state.selected) return;
        var type = document.getElementById('pf-type') ? document.getElementById('pf-type').value : 'plain';
        if (type !== 'jinja') {
            // For plain prompts there is nothing to compile — just surface actions.
            showCompiledResult({
                title: state.selected.title + ' (plain)',
                content: readEditorContent(),
                savable: false
            });
            return;
        }
        if (typeof nunjucks === 'undefined') {
            alert('Nunjucks template library failed to load.');
            return;
        }

        var body = readEditorContent();
        var declared = (state.selected.variables || []).slice();
        var detectedNames = detectTemplateVariables(body);

        // Merge: keep declared definitions (with type/label/etc.) and append any
        // names found in the template that weren't declared, as simple strings.
        var declaredByName = {};
        declared.forEach(function(v) { declaredByName[v.name] = true; });
        var merged = declared.slice();
        detectedNames.forEach(function(name) {
            if (!declaredByName[name]) {
                merged.push({ name: name, type: 'string', label: '', default: '', options: [], required: false, _autoDetected: true });
            }
        });

        if (merged.length === 0) {
            // Truly no variables — compile the body as-is and offer to save.
            showCompiledResult({
                title: state.selected.title,
                content: body,
                savable: true,
                values: {}
            });
            return;
        }

        var vars = merged;
        var dialog = document.createElement('dialog');
        dialog.className = 'bg-surface-base text-th-text-secondary rounded-xl shadow-2xl border border-th-border-strong p-0 backdrop:bg-black/60';
        dialog.style.width = '560px';
        dialog.style.maxWidth = '90vw';

        var fields = '';
        var autoDetectedCount = 0;
        vars.forEach(function(v, i) {
            var label = v.label || v.name;
            var req = v.required ? '<span class="text-red-400">*</span>' : '';
            var autoBadge = v._autoDetected
                ? ' <span class="text-[9px] px-1 py-0 rounded bg-amber-900/50 text-amber-300 normal-case lowercase">auto</span>'
                : '';
            if (v._autoDetected) autoDetectedCount++;
            fields += '<div class="flex flex-col gap-1 mb-3">'
                + '<label class="text-[11px] font-medium text-th-text-muted uppercase tracking-wider">' + escapeHTML(label) + ' ' + req + autoBadge + ' <span class="text-th-text-faint lowercase normal-case ml-1 font-mono">{' + escapeHTML(v.name) + '}</span></label>'
                + variableInputHTML(v, i)
                + '</div>';
        });

        var hint = 'Fill in values for each variable. The template compiles in-browser.';
        if (autoDetectedCount > 0) {
            hint += ' ' + autoDetectedCount + ' variable' + (autoDetectedCount === 1 ? ' was' : 's were') +
                ' auto-detected from the template — declare them in the Variables drawer to customize type or default.';
        }

        dialog.innerHTML = ''
            + '<div class="px-5 pt-4 pb-3 border-b border-th-border-strong">'
            + '  <h3 class="text-base font-semibold text-th-text-primary">Compile prompt</h3>'
            + '  <p class="text-xs text-th-text-muted mt-1">' + escapeHTML(hint) + '</p>'
            + '</div>'
            + '<div class="px-5 py-4 max-h-[60vh] overflow-y-auto" id="pf-compile-form">' + fields + '</div>'
            + '<div class="px-5 py-3 border-t border-th-border-strong flex items-center gap-2">'
            + '  <input id="pf-version-title" type="text" placeholder="Version title (defaults to timestamp)" class="flex-1 bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 text-xs text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '  <button id="pf-compile-cancel" class="px-3 py-1.5 text-xs text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors">Cancel</button>'
            + '  <button id="pf-compile-submit" class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded transition-colors font-medium">Compile</button>'
            + '</div>';

        document.body.appendChild(dialog);
        dialog.showModal();

        dialog.querySelector('#pf-compile-cancel').addEventListener('click', function() {
            dialog.close();
            dialog.remove();
        });
        dialog.querySelector('#pf-compile-submit').addEventListener('click', function() {
            var values = {};
            for (var i = 0; i < vars.length; i++) {
                var el = dialog.querySelector('[data-compile-var="' + i + '"]');
                if (!el) continue;
                var v = vars[i];
                var raw;
                if (v.type === 'boolean') {
                    raw = el.checked;
                } else if (v.type === 'number') {
                    if (el.value === '') {
                        raw = null;
                    } else {
                        raw = Number(el.value);
                        if (isNaN(raw)) {
                            alert('Variable "' + v.name + '" must be a number.');
                            return;
                        }
                    }
                } else {
                    raw = el.value;
                }
                if (v.required && (raw === '' || raw === null || raw === undefined)) {
                    alert('Variable "' + v.name + '" is required.');
                    return;
                }
                values[v.name] = raw;
            }
            var titleOverride = dialog.querySelector('#pf-version-title').value.trim();
            var rendered;
            try {
                rendered = nunjucks.renderString(readEditorContent(), values);
            } catch (err) {
                alert('Template error: ' + (err && err.message ? err.message : err));
                return;
            }
            dialog.close();
            dialog.remove();
            showCompiledResult({
                title: titleOverride || ('Compiled ' + formatNow()),
                content: rendered,
                savable: true,
                values: values
            });
        });
    }

    // detectTemplateVariables scans the template body for Jinja/Nunjucks
    // variable references like {{ name }}, {{ name | filter }}, or
    // {{ name.attr }} and returns a de-duplicated list of the top-level
    // identifiers. Reserved names like `loop`, `true`, `false`, `null`, and
    // control-flow identifiers are skipped.
    function detectTemplateVariables(body) {
        if (!body) return [];
        var reserved = { 'true': 1, 'false': 1, 'null': 1, 'none': 1, 'loop': 1, 'self': 1,
            'if': 1, 'else': 1, 'elif': 1, 'endif': 1, 'for': 1, 'endfor': 1, 'in': 1,
            'and': 1, 'or': 1, 'not': 1, 'block': 1, 'endblock': 1, 'extends': 1,
            'include': 1, 'set': 1, 'macro': 1, 'endmacro': 1, 'raw': 1, 'endraw': 1 };
        var seen = {};
        var found = [];
        // Matches {{ IDENT ... }} capturing IDENT as the first identifier after `{{`.
        // Also picks up the primary identifier inside {% if IDENT %} / {% for x in IDENT %}.
        var exprRegex = /\{\{\s*([a-zA-Z_][a-zA-Z0-9_]*)/g;
        var m;
        while ((m = exprRegex.exec(body)) !== null) {
            var name = m[1];
            if (reserved[name.toLowerCase()]) continue;
            if (!seen[name]) { seen[name] = 1; found.push(name); }
        }
        // `{% for x in items %}` → capture `items`
        var forRegex = /\{%\s*for\s+[a-zA-Z_][a-zA-Z0-9_]*\s+in\s+([a-zA-Z_][a-zA-Z0-9_]*)/g;
        while ((m = forRegex.exec(body)) !== null) {
            var n = m[1];
            if (reserved[n.toLowerCase()]) continue;
            if (!seen[n]) { seen[n] = 1; found.push(n); }
        }
        // `{% if IDENT %}` → capture IDENT
        var ifRegex = /\{%\s*(?:if|elif)\s+([a-zA-Z_][a-zA-Z0-9_]*)/g;
        while ((m = ifRegex.exec(body)) !== null) {
            var k = m[1];
            if (reserved[k.toLowerCase()]) continue;
            if (!seen[k]) { seen[k] = 1; found.push(k); }
        }
        return found;
    }

    function variableInputHTML(v, i) {
        var cls = 'w-full bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 text-xs text-th-text-primary focus:outline-none focus:border-accent-border';
        var def = v.default || '';
        switch (v.type) {
            case 'text':
                return '<textarea data-compile-var="' + i + '" rows="3" class="' + cls + ' font-mono resize-y">' + escapeHTML(def) + '</textarea>';
            case 'number':
                return '<input data-compile-var="' + i + '" type="number" value="' + escapeAttr(def) + '" class="' + cls + '">';
            case 'boolean':
                return '<label class="inline-flex items-center gap-2 text-xs text-th-text-tertiary"><input data-compile-var="' + i + '" type="checkbox"' + (def === 'true' ? ' checked' : '') + '> true</label>';
            case 'select':
                var opts = (v.options || []).map(function(o) { return '<option value="' + escapeAttr(o) + '"' + (o === def ? ' selected' : '') + '>' + escapeHTML(o) + '</option>'; }).join('');
                return '<select data-compile-var="' + i + '" class="' + cls + '">' + opts + '</select>';
            case 'date':
                return '<input data-compile-var="' + i + '" type="date" value="' + escapeAttr(def) + '" class="' + cls + '">';
            default:
                return '<input data-compile-var="' + i + '" type="text" value="' + escapeAttr(def) + '" class="' + cls + '">';
        }
    }

    function showCompiledResult(opts) {
        var dialog = document.createElement('dialog');
        dialog.className = 'bg-surface-base text-th-text-secondary rounded-xl shadow-2xl border border-th-border-strong p-0 backdrop:bg-black/60';
        dialog.style.width = '760px';
        dialog.style.maxWidth = '92vw';

        dialog.innerHTML = ''
            + '<div class="px-5 pt-4 pb-3 border-b border-th-border-strong flex items-center justify-between gap-2">'
            + '  <input id="pf-result-title" type="text" value="' + escapeAttr(opts.title) + '" class="flex-1 bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '  <button id="pf-result-close" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded"><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg></button>'
            + '</div>'
            + '<div id="pf-result-preview" class="px-5 py-4 overflow-y-auto md-preview-container note-markdown-preview text-sm text-th-text-tertiary" style="max-height: 52vh;"></div>'
            + '<div class="px-5 py-3 border-t border-th-border-strong flex items-center gap-2 justify-end">'
            + '  <button id="pf-result-insert" class="px-3 py-1.5 text-xs text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors">Insert into terminal</button>'
            + '  <button id="pf-result-copy" class="px-3 py-1.5 text-xs text-th-text-tertiary hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors">Copy</button>'
            + (opts.savable ? '<button id="pf-result-save" class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded font-medium">Save version</button>' : '')
            + '</div>';

        document.body.appendChild(dialog);
        dialog.showModal();

        var previewEl = dialog.querySelector('#pf-result-preview');
        if (window.ClawIDEMarkdown) window.ClawIDEMarkdown.renderInto(previewEl, opts.content);
        else previewEl.textContent = opts.content;

        function teardown() { dialog.close(); dialog.remove(); }
        dialog.querySelector('#pf-result-close').addEventListener('click', teardown);
        dialog.querySelector('#pf-result-insert').addEventListener('click', function() { insertIntoTerminal(opts.content); });
        dialog.querySelector('#pf-result-copy').addEventListener('click', function() { copyToClipboard(opts.content); });
        if (opts.savable) {
            dialog.querySelector('#pf-result-save').addEventListener('click', function() {
                var title = dialog.querySelector('#pf-result-title').value.trim();
                fetch(API + '/prompts/' + encodeURIComponent(state.selected.id) + '/versions', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ title: title, variable_values: opts.values || {}, content: opts.content })
                }).then(handleJSON).then(function() {
                    toast('Version saved');
                    teardown();
                    return loadVersions();
                }).catch(showErr);
            });
        }
    }

    // ── Versions list ────────────────────────────────────────────────────

    function loadVersions() {
        if (!state.selected) {
            state.versions = [];
            renderVersions();
            return Promise.resolve();
        }
        return fetch(API + '/prompts/' + encodeURIComponent(state.selected.id) + '/versions')
            .then(handleJSON)
            .then(function(list) {
                state.versions = list || [];
                renderVersions();
            })
            .catch(function(err) {
                console.error('PromptForge: versions load failed', err);
                state.versions = [];
                renderVersions();
            });
    }

    function renderVersions() {
        var container = document.getElementById('pf-versions');
        if (!container) return;
        if (!state.selected || state.versions.length === 0) {
            container.innerHTML = '<div class="px-4 py-2 text-[11px] text-th-text-faint">No compiled versions yet.</div>';
            return;
        }
        var rows = state.versions.map(function(v) {
            return ''
                + '<div class="pf-version-row flex items-center gap-2 px-4 py-1.5 text-xs border-b border-th-border/60 hover:bg-surface-raised">'
                + '  <button data-version-view="' + v.id + '" class="flex-1 text-left text-th-text-tertiary hover:text-th-text-primary truncate">' + escapeHTML(v.title) + '</button>'
                + '  <span class="text-[10px] text-th-text-faint">' + escapeHTML(formatStamp(v.compiled_at)) + '</span>'
                + '  <button data-version-rename="' + v.id + '" title="Rename" class="p-1 text-th-text-muted hover:text-th-text-primary"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/></svg></button>'
                + '  <button data-version-delete="' + v.id + '" title="Delete" class="p-1 text-th-text-muted hover:text-red-400"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7M1 7h22"/></svg></button>'
                + '</div>';
        }).join('');
        container.innerHTML = '<div class="px-4 py-1.5 text-[10px] text-th-text-faint uppercase tracking-wider border-b border-th-border sticky top-0 bg-surface-base">Compiled versions (' + state.versions.length + ')</div>' + rows;

        container.querySelectorAll('[data-version-view]').forEach(function(btn) {
            btn.addEventListener('click', function() { viewVersion(btn.getAttribute('data-version-view')); });
        });
        container.querySelectorAll('[data-version-rename]').forEach(function(btn) {
            btn.addEventListener('click', function() { renameVersion(btn.getAttribute('data-version-rename')); });
        });
        container.querySelectorAll('[data-version-delete]').forEach(function(btn) {
            btn.addEventListener('click', function() { deleteVersion(btn.getAttribute('data-version-delete')); });
        });
    }

    function viewVersion(id) {
        fetch(API + '/prompts/' + encodeURIComponent(state.selected.id) + '/versions/' + encodeURIComponent(id))
            .then(handleJSON)
            .then(function(v) {
                showCompiledResult({ title: v.title, content: v.content || '', savable: false });
            }).catch(showErr);
    }

    function renameVersion(id) {
        var current = (state.versions.filter(function(v) { return v.id === id; })[0] || {}).title || '';
        var title = window.prompt('Rename compiled version to:', current);
        if (!title || title === current) return;
        fetch(API + '/prompts/' + encodeURIComponent(state.selected.id) + '/versions/' + encodeURIComponent(id), {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title: title })
        }).then(handleJSON).then(function() { return loadVersions(); }).catch(showErr);
    }

    function deleteVersion(id) {
        if (!window.confirm('Delete this compiled version?')) return;
        fetch(API + '/prompts/' + encodeURIComponent(state.selected.id) + '/versions/' + encodeURIComponent(id), { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return loadVersions();
            }).catch(showErr);
    }

    // ── Clipboard + terminal insertion ───────────────────────────────────

    function insertIntoTerminal(text) {
        if (!text) return;
        if (!window.ClawIDETerminal) {
            toast('Terminal not available on this page');
            return;
        }
        var paneID = window.ClawIDETerminal.getFocusedPaneID && window.ClawIDETerminal.getFocusedPaneID();
        if (!paneID) {
            var all = window.ClawIDETerminal.getAllPaneIDs && window.ClawIDETerminal.getAllPaneIDs();
            if (!all || !all.length) {
                toast('No terminal pane to insert into');
                return;
            }
            paneID = all[0];
        }
        window.ClawIDETerminal.sendInput(paneID, text);
        toast('Inserted into terminal');
    }

    function copyToClipboard(text) {
        if (!text) return;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(text).then(function() { toast('Copied'); }).catch(fallback);
        } else {
            fallback();
        }
        function fallback() {
            var ta = document.createElement('textarea');
            ta.value = text;
            ta.style.cssText = 'position:fixed;left:-9999px;top:0;';
            document.body.appendChild(ta);
            ta.select();
            try { document.execCommand('copy'); toast('Copied'); } catch (e) { toast('Copy failed'); }
            document.body.removeChild(ta);
        }
    }

    // ── Utilities ────────────────────────────────────────────────────────

    function handleJSON(r) {
        if (!r.ok) return r.text().then(function(t) { throw new Error(t || ('HTTP ' + r.status)); });
        if (r.status === 204) return null;
        return r.json();
    }

    function showErr(err) {
        if (!err || err.message === 'cancelled') return;
        console.error('PromptForge:', err);
        toast(String(err.message || err));
    }

    function toast(msg) {
        if (window.ClawIDEToast && window.ClawIDEToast.show) {
            window.ClawIDEToast.show(msg);
        } else {
            console.log('[PromptForge] ' + msg);
        }
    }

    function escapeHTML(s) {
        if (s === undefined || s === null) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(String(s)));
        return div.innerHTML;
    }

    function escapeAttr(s) {
        if (s === undefined || s === null) return '';
        return String(s).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    function formatNow() {
        var d = new Date();
        return d.getFullYear() + '-' + pad(d.getMonth() + 1) + '-' + pad(d.getDate()) + ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes()) + ':' + pad(d.getSeconds());
    }

    function formatStamp(iso) {
        try {
            var d = new Date(iso);
            return d.toLocaleString();
        } catch (e) { return String(iso || ''); }
    }

    function pad(n) { return n < 10 ? '0' + n : String(n); }

    // ── Init ─────────────────────────────────────────────────────────────

    document.addEventListener('DOMContentLoaded', function() {
        var btn = document.getElementById('promptforge-open-btn');
        if (btn) btn.addEventListener('click', open);
    });

    window.ClawIDEPromptForge = {
        open: open,
        close: close
    };
})();
