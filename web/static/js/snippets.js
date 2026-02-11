// ClawIDE Snippets Manager
// Floating drawer UI for global code snippets with CRUD, search, and terminal insert.
(function() {
    'use strict';

    var API_BASE = '/api/snippets';
    var debounceTimer = null;
    var snippets = [];
    var editingID = null;

    // DOM references (populated in init)
    var drawer, overlay, list, searchInput, form, nameInput, contentInput, formTitle, cancelBtn;

    function init() {
        drawer = document.getElementById('snippet-drawer');
        overlay = document.getElementById('snippet-overlay');
        list = document.getElementById('snippet-list');
        searchInput = document.getElementById('snippet-search');
        form = document.getElementById('snippet-form');
        nameInput = document.getElementById('snippet-name');
        contentInput = document.getElementById('snippet-content');
        formTitle = document.getElementById('snippet-form-title');
        cancelBtn = document.getElementById('snippet-cancel');

        if (!drawer) return;

        // Toggle button
        var toggleBtn = document.getElementById('snippet-toggle');
        if (toggleBtn) {
            toggleBtn.addEventListener('click', toggle);
        }

        // Overlay close
        if (overlay) {
            overlay.addEventListener('click', close);
        }

        // Search with debounce
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(function() {
                    loadSnippets(searchInput.value.trim());
                }, 250);
            });
        }

        // Form submit
        if (form) {
            form.addEventListener('submit', function(e) {
                e.preventDefault();
                saveSnippet();
            });
        }

        // Cancel edit
        if (cancelBtn) {
            cancelBtn.addEventListener('click', resetForm);
        }

        // Initial load
        loadSnippets('');
    }

    function toggle() {
        if (drawer.classList.contains('open')) {
            close();
        } else {
            open();
        }
    }

    function open() {
        drawer.classList.add('open');
        if (overlay) overlay.classList.add('open');
        loadSnippets(searchInput ? searchInput.value.trim() : '');
    }

    function close() {
        drawer.classList.remove('open');
        if (overlay) overlay.classList.remove('open');
        resetForm();
    }

    function loadSnippets(query) {
        var url = API_BASE;
        if (query) {
            url += '?q=' + encodeURIComponent(query);
        }
        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                snippets = data || [];
                renderList();
            })
            .catch(function(err) {
                console.error('Failed to load snippets:', err);
            });
    }

    function renderList() {
        if (!list) return;

        if (snippets.length === 0) {
            list.innerHTML = '<div class="text-gray-500 text-sm p-3 text-center">No snippets yet</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < snippets.length; i++) {
            var sn = snippets[i];
            var preview = sn.content.length > 60 ? sn.content.substring(0, 60) + '...' : sn.content;
            html += '<div class="snippet-item" data-id="' + sn.id + '">';
            html += '  <div class="snippet-item-header">';
            html += '    <span class="snippet-item-name">' + escapeHTML(sn.name) + '</span>';
            html += '    <div class="snippet-item-actions">';
            html += '      <button class="snippet-btn-insert" title="Insert into terminal" data-insert="' + sn.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 10 4 15 9 20"/><path d="M20 4v7a4 4 0 01-4 4H4"/></svg>';
            html += '      </button>';
            html += '      <button class="snippet-btn-edit" title="Edit" data-edit="' + sn.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>';
            html += '      </button>';
            html += '      <button class="snippet-btn-delete" title="Delete" data-delete="' + sn.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
            html += '      </button>';
            html += '    </div>';
            html += '  </div>';
            html += '  <div class="snippet-item-preview">' + escapeHTML(preview) + '</div>';
            html += '</div>';
        }
        list.innerHTML = html;

        // Bind action buttons
        var insertBtns = list.querySelectorAll('[data-insert]');
        for (var j = 0; j < insertBtns.length; j++) {
            insertBtns[j].addEventListener('click', function() {
                insertSnippet(this.getAttribute('data-insert'));
            });
        }

        var editBtns = list.querySelectorAll('[data-edit]');
        for (var k = 0; k < editBtns.length; k++) {
            editBtns[k].addEventListener('click', function() {
                startEdit(this.getAttribute('data-edit'));
            });
        }

        var deleteBtns = list.querySelectorAll('[data-delete]');
        for (var l = 0; l < deleteBtns.length; l++) {
            deleteBtns[l].addEventListener('click', function() {
                deleteSnippet(this.getAttribute('data-delete'));
            });
        }
    }

    function saveSnippet() {
        var name = nameInput.value.trim();
        var content = contentInput.value;
        if (!name) return;

        var method, url;
        if (editingID) {
            method = 'PUT';
            url = API_BASE + '/' + editingID;
        } else {
            method = 'POST';
            url = API_BASE;
        }

        fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: name, content: content })
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Save failed');
            return r.json();
        })
        .then(function() {
            resetForm();
            loadSnippets(searchInput ? searchInput.value.trim() : '');
        })
        .catch(function(err) {
            console.error('Failed to save snippet:', err);
        });
    }

    function startEdit(id) {
        var sn = findSnippet(id);
        if (!sn) return;

        editingID = id;
        nameInput.value = sn.name;
        contentInput.value = sn.content;
        if (formTitle) formTitle.textContent = 'Edit Snippet';
        if (cancelBtn) cancelBtn.style.display = '';
    }

    function deleteSnippet(id) {
        if (!confirm('Delete this snippet?')) return;

        fetch(API_BASE + '/' + id, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) throw new Error('Delete failed');
                loadSnippets(searchInput ? searchInput.value.trim() : '');
            })
            .catch(function(err) {
                console.error('Failed to delete snippet:', err);
            });
    }

    function insertSnippet(id) {
        var sn = findSnippet(id);
        if (!sn) return;

        var paneID = window.ClawIDETerminal.getFocusedPaneID();
        if (!paneID) {
            var allPanes = window.ClawIDETerminal.getAllPaneIDs();
            if (allPanes.length === 0) return;
            paneID = allPanes[0];
        }
        window.ClawIDETerminal.sendInput(paneID, sn.content);
    }

    function resetForm() {
        editingID = null;
        if (nameInput) nameInput.value = '';
        if (contentInput) contentInput.value = '';
        if (formTitle) formTitle.textContent = 'New Snippet';
        if (cancelBtn) cancelBtn.style.display = 'none';
    }

    function findSnippet(id) {
        for (var i = 0; i < snippets.length; i++) {
            if (snippets[i].id === id) return snippets[i];
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
    window.ClawIDESnippets = {
        open: open,
        close: close,
        toggle: toggle
    };
})();
