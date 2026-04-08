// ClawIDE Notes Manager
// Sidebar-based notes with project/global toggle, search, CRUD, and markdown preview.
(function() {
    'use strict';

    var API_BASE = '/api/notes';
    var debounceTimer = null;
    var notes = [];
    var editingID = null;
    var scope = 'project'; // 'project' or 'global'
    var projectID = '';
    var gitStatusLoaded = false;

    // DOM references
    var container, list, searchInput, form, titleInput, contentInput;
    var formTitle, cancelBtn, previewEl, previewToggle;
    var tabProject, tabGlobal;

    function init() {
        container = document.getElementById('notes-container');
        list = document.getElementById('notes-list');
        searchInput = document.getElementById('notes-search');
        form = document.getElementById('notes-form');
        titleInput = document.getElementById('notes-title');
        contentInput = document.getElementById('notes-content');
        formTitle = document.getElementById('notes-form-title');
        cancelBtn = document.getElementById('notes-cancel');
        previewEl = document.getElementById('notes-preview');
        previewToggle = document.getElementById('notes-preview-toggle');
        tabProject = document.getElementById('notes-tab-project');
        tabGlobal = document.getElementById('notes-tab-global');

        if (!container) return;

        projectID = container.getAttribute('data-project-id') || '';

        // Search with debounce
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(function() {
                    loadNotes();
                }, 250);
            });
        }

        // Form submit
        if (form) {
            form.addEventListener('submit', function(e) {
                e.preventDefault();
                saveNote();
            });
        }

        // Cancel edit
        if (cancelBtn) {
            cancelBtn.addEventListener('click', resetForm);
        }

        // Preview toggle
        if (previewToggle) {
            previewToggle.addEventListener('click', togglePreview);
        }

        // Listen for git status updates
        document.addEventListener('clawide-git-status-update', function(e) {
            if (e.detail && e.detail.type === 'notes') {
                renderGitUI();
                renderList(); // re-render with badges
            }
        });

        // Initial load
        loadNotes();
        loadGitStatus();
    }

    function loadGitStatus() {
        if (!projectID || !window.ClawIDEGit) return;
        window.ClawIDEGit.fetchStatus('notes', projectID).then(function() {
            gitStatusLoaded = true;
            renderGitUI();
            renderList(); // re-render with badges
        });
    }

    function renderGitUI() {
        if (!window.ClawIDEGit) return;

        // Render warning banner
        var bannerEl = document.getElementById('notes-git-banner');
        if (bannerEl) {
            bannerEl.innerHTML = window.ClawIDEGit.renderWarningBanner('notes');
        }

        // Render git toolbar (refresh + commit button)
        var toolbarEl = document.getElementById('notes-git-toolbar');
        if (toolbarEl) {
            var status = window.ClawIDEGit.getCachedStatus('notes');
            if (status && status.is_git_repo && !status.is_ignored) {
                toolbarEl.innerHTML = window.ClawIDEGit.renderRefreshButton('notes') +
                    window.ClawIDEGit.renderCommitButton('notes');
                toolbarEl.style.display = '';
            } else {
                toolbarEl.innerHTML = '';
                toolbarEl.style.display = 'none';
            }
        }
    }

    function setScope(newScope) {
        scope = newScope;

        // Update tab styles
        if (tabProject && tabGlobal) {
            if (scope === 'project') {
                tabProject.className = 'flex-1 px-2 py-1 text-xs font-medium rounded bg-accent/30 text-accent-text';
                tabGlobal.className = 'flex-1 px-2 py-1 text-xs font-medium rounded text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised';
            } else {
                tabProject.className = 'flex-1 px-2 py-1 text-xs font-medium rounded text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised';
                tabGlobal.className = 'flex-1 px-2 py-1 text-xs font-medium rounded bg-accent/30 text-accent-text';
            }
        }

        resetForm();
        loadNotes();
    }

    function loadNotes() {
        var params = [];
        var pid = scope === 'project' ? projectID : '';
        if (pid) {
            params.push('project_id=' + encodeURIComponent(pid));
        }
        var query = searchInput ? searchInput.value.trim() : '';
        if (query) {
            params.push('q=' + encodeURIComponent(query));
        }

        var url = API_BASE;
        if (params.length > 0) {
            url += '?' + params.join('&');
        }

        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                notes = data || [];
                renderList();
            })
            .catch(function(err) {
                console.error('Failed to load notes:', err);
            });
    }

    function renderList() {
        if (!list) return;

        if (notes.length === 0) {
            list.innerHTML = '<div class="text-th-text-faint text-xs p-3 text-center">No notes yet</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < notes.length; i++) {
            var note = notes[i];
            var preview = note.content.length > 80 ? note.content.substring(0, 80) + '...' : note.content;
            html += '<div class="note-item" data-id="' + note.id + '">';
            html += '  <div class="flex items-center justify-between gap-2">';
            html += '    <span class="text-xs text-th-text-primary font-medium truncate">' + escapeHTML(note.title) + '</span>';
            html += '    <div class="flex items-center gap-1 flex-shrink-0">';
            // Git status badge
            if (gitStatusLoaded && window.ClawIDEGit) {
                var notePath = '.clawide/notes/' + note.id + '.md';
                var gitSt = window.ClawIDEGit.getFileStatus('notes', notePath);
                if (gitSt) {
                    html += '      ' + window.ClawIDEGit.renderBadge(gitSt);
                }
            }
            html += '      <button class="p-0.5 rounded text-th-text-faint hover:text-th-text-primary transition-colors" title="Edit" data-note-edit="' + note.id + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>';
            html += '      </button>';
            html += '      <button class="p-0.5 rounded text-th-text-faint hover:text-red-400 transition-colors" title="Delete" data-note-delete="' + note.id + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
            html += '      </button>';
            html += '    </div>';
            html += '  </div>';
            html += '  <div class="text-[10px] text-th-text-faint mt-0.5 truncate">' + escapeHTML(preview) + '</div>';
            html += '</div>';
        }
        list.innerHTML = html;

        // Bind action buttons
        var editBtns = list.querySelectorAll('[data-note-edit]');
        for (var j = 0; j < editBtns.length; j++) {
            editBtns[j].addEventListener('click', function() {
                startEdit(this.getAttribute('data-note-edit'));
            });
        }

        var deleteBtns = list.querySelectorAll('[data-note-delete]');
        for (var k = 0; k < deleteBtns.length; k++) {
            deleteBtns[k].addEventListener('click', function() {
                deleteNote(this.getAttribute('data-note-delete'));
            });
        }
    }

    function saveNote() {
        var title = titleInput.value.trim();
        var content = contentInput.value;
        if (!title) return;

        var method, url;
        if (editingID) {
            method = 'PUT';
            url = API_BASE + '/' + editingID;
        } else {
            method = 'POST';
            url = API_BASE;
        }

        var body = { title: title, content: content };
        if (!editingID) {
            body.project_id = scope === 'project' ? projectID : '';
        }

        fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Save failed');
            return r.json();
        })
        .then(function() {
            resetForm();
            loadNotes();
        })
        .catch(function(err) {
            console.error('Failed to save note:', err);
        });
    }

    function startEdit(id) {
        var note = findNote(id);
        if (!note) return;

        editingID = id;
        titleInput.value = note.title;
        contentInput.value = note.content;
        if (formTitle) formTitle.textContent = 'Edit Note';
        if (cancelBtn) cancelBtn.style.display = '';
        // Hide preview when entering edit mode
        if (previewEl) previewEl.classList.add('hidden');
    }

    function deleteNote(id) {
        if (!confirm('Delete this note?')) return;

        fetch(API_BASE + '/' + id, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) throw new Error('Delete failed');
                loadNotes();
            })
            .catch(function(err) {
                console.error('Failed to delete note:', err);
            });
    }

    function togglePreview() {
        if (!previewEl || !contentInput) return;

        if (previewEl.classList.contains('hidden')) {
            previewEl.innerHTML = renderMarkdown(contentInput.value);
            previewEl.classList.remove('hidden');
            if (previewToggle) previewToggle.textContent = 'Edit';
        } else {
            previewEl.classList.add('hidden');
            if (previewToggle) previewToggle.textContent = 'Preview';
        }
    }

    function renderMarkdown(text) {
        if (!text) return '<span class="text-th-text-faint">Nothing to preview</span>';

        // Minimal markdown renderer: headings, bold, italic, code, links, line breaks
        var html = escapeHTML(text);

        // Headings (### > ## > #)
        html = html.replace(/^### (.+)$/gm, '<strong class="text-sm text-th-text-primary">$1</strong>');
        html = html.replace(/^## (.+)$/gm, '<strong class="text-sm text-th-text-primary">$1</strong>');
        html = html.replace(/^# (.+)$/gm, '<strong class="text-base text-th-text-primary">$1</strong>');

        // Bold
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');

        // Italic
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');

        // Inline code
        html = html.replace(/`([^`]+)`/g, '<code class="bg-surface-overlay px-1 rounded text-[10px]">$1</code>');

        // Links
        html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" class="text-accent-text hover:underline">$1</a>');

        // Line breaks
        html = html.replace(/\n/g, '<br>');

        return html;
    }

    function resetForm() {
        editingID = null;
        if (titleInput) titleInput.value = '';
        if (contentInput) contentInput.value = '';
        if (formTitle) formTitle.textContent = 'New Note';
        if (cancelBtn) cancelBtn.style.display = 'none';
        if (previewEl) previewEl.classList.add('hidden');
        if (previewToggle) previewToggle.textContent = 'Preview';
    }

    function findNote(id) {
        for (var i = 0; i < notes.length; i++) {
            if (notes[i].id === id) return notes[i];
        }
        return null;
    }

    function escapeHTML(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose for external use
    window.ClawIDENotes = {
        setScope: setScope,
        reload: loadNotes,
        refreshGitStatus: loadGitStatus
    };
})();
