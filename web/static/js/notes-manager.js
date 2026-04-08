// ClawIDE Notes Manager
// Tab-based notes with folder tree (notes as files), edit/preview modes,
// markdown rendering, Mermaid diagram support, context menus, and auto-save.
(function() {
    'use strict';

    var API_BASE = '/api/notes';
    var AUTOSAVE_DELAY = 500; // ms

    var projectID = '';
    var notes = [];
    var currentNoteID = null;
    var currentFolderID = '';
    var folderTree = null;
    var saveTimer = null;
    var saveState = 'idle'; // 'idle' | 'saving' | 'saved'
    var isDirty = false;
    var gitStatusLoaded = false;

    // Context menu
    var contextMenuEl = null;

    // Title validation
    var titleErrorEl = null;
    var VALID_TITLE_REGEX = /^[a-zA-Z0-9._-]*$/;

    function validateTitle(title) {
        if (!title) return ''; // empty is handled by required check
        if (!VALID_TITLE_REGEX.test(title)) {
            return 'Only letters, numbers, dots, hyphens, and underscores allowed';
        }
        if (title === '.' || title === '..') {
            return 'Title cannot be "." or ".."';
        }
        if (title.length > 255) {
            return 'Title exceeds maximum length of 255 characters';
        }
        return '';
    }

    function showTitleError(msg) {
        if (!titleErrorEl) {
            titleErrorEl = document.createElement('div');
            titleErrorEl.className = 'text-[10px] text-red-400 mt-0.5';
            if (titleInput && titleInput.parentNode) {
                titleInput.parentNode.insertBefore(titleErrorEl, titleInput.nextSibling);
            }
        }
        if (msg) {
            titleErrorEl.textContent = msg;
            titleErrorEl.style.display = '';
            if (titleInput) titleInput.classList.add('border-red-500');
        } else {
            titleErrorEl.textContent = '';
            titleErrorEl.style.display = 'none';
            if (titleInput) titleInput.classList.remove('border-red-500');
        }
    }

    // DOM refs
    var container, searchInput, titleInput, contentInput;
    var previewEl, formEl, cancelBtn, formTitleEl;
    var saveIndicator, newNoteBtn, deleteNoteBtn;
    var toolbarBold, toolbarItalic, toolbarLink, toolbarMermaid;
    var tabProject, tabGlobal;
    var scope = 'project';
    var newFolderBtn;

    function init() {
        container = document.getElementById('notes-container');
        if (!container) return;

        projectID = container.getAttribute('data-project-id') || '';
        searchInput = document.getElementById('notes-search');
        titleInput = document.getElementById('notes-title');
        contentInput = document.getElementById('notes-content');
        previewEl = document.getElementById('notes-preview');
        formEl = document.getElementById('notes-form');
        cancelBtn = document.getElementById('notes-cancel');
        formTitleEl = document.getElementById('notes-form-title');
        saveIndicator = document.getElementById('notes-save-indicator');
        newNoteBtn = document.getElementById('notes-new-btn');
        deleteNoteBtn = document.getElementById('notes-delete-btn');
        tabProject = document.getElementById('notes-tab-project');
        tabGlobal = document.getElementById('notes-tab-global');
        newFolderBtn = document.getElementById('notes-new-folder-btn');

        // Toolbar buttons
        toolbarBold = document.getElementById('notes-toolbar-bold');
        toolbarItalic = document.getElementById('notes-toolbar-italic');
        toolbarLink = document.getElementById('notes-toolbar-link');
        toolbarMermaid = document.getElementById('notes-toolbar-mermaid');

        // Init folder tree with file support and context menus
        var treeContainer = document.getElementById('notes-folder-tree-inner');
        if (treeContainer) {
            folderTree = new FolderTree({
                container: treeContainer,
                projectID: projectID,
                apiBase: API_BASE + '/folders',
                rootLabel: 'All Notes',
                allowDrag: true,
                onSelect: function(folderID) {
                    currentFolderID = folderID;
                    // Deselect any file selection, clear editor
                    resetForm();
                },
                onFileSelect: function(fileID) {
                    selectNote(fileID);
                },
                onDrop: function(itemID, targetFolderID, itemType) {
                    if (itemType === 'file') {
                        moveNoteToFolder(itemID, targetFolderID);
                    } else if (itemType === 'folder') {
                        moveFolderToParent(itemID, targetFolderID);
                    }
                },
                onContextMenu: function(type, id, event) {
                    showContextMenu(type, id, event);
                }
            });
            folderTree.load();
        }

        // Search with debounce - filters the tree
        if (searchInput) {
            var searchTimer = null;
            searchInput.addEventListener('input', function() {
                clearTimeout(searchTimer);
                var query = searchInput.value.trim();
                searchTimer = setTimeout(function() {
                    if (folderTree) {
                        folderTree.setFilter(query);
                    }
                }, 250);
            });
        }

        // Form submit (manual save)
        if (formEl) {
            formEl.addEventListener('submit', function(e) {
                e.preventDefault();
                saveNote();
            });
        }

        // Cancel edit
        if (cancelBtn) {
            cancelBtn.addEventListener('click', resetForm);
        }

        // New note button
        if (newNoteBtn) {
            newNoteBtn.addEventListener('click', function() {
                promptNewFile(currentFolderID);
            });
        }

        // Delete note button
        if (deleteNoteBtn) {
            deleteNoteBtn.addEventListener('click', function() {
                if (currentNoteID) deleteNote(currentNoteID);
            });
        }

        // New folder button
        if (newFolderBtn && folderTree) {
            newFolderBtn.addEventListener('click', function() {
                promptNewFolder(folderTree.selectedID || '');
            });
        }

        // Auto-save: debounced on input, save on blur
        if (contentInput) {
            contentInput.addEventListener('input', function() {
                isDirty = true;
                scheduleSave();
            });
            contentInput.addEventListener('blur', function() {
                if (isDirty && currentNoteID) {
                    clearTimeout(saveTimer);
                    autoSave();
                }
            });
        }
        if (titleInput) {
            titleInput.addEventListener('input', function() {
                var error = validateTitle(titleInput.value);
                showTitleError(error);
                isDirty = true;
                if (!error) scheduleSave();
            });
            titleInput.addEventListener('blur', function() {
                if (isDirty && currentNoteID && !validateTitle(titleInput.value)) {
                    clearTimeout(saveTimer);
                    autoSave();
                }
            });
        }

        // Toolbar actions
        if (toolbarBold) toolbarBold.addEventListener('click', function() { wrapSelection('**', '**'); });
        if (toolbarItalic) toolbarItalic.addEventListener('click', function() { wrapSelection('*', '*'); });
        if (toolbarLink) toolbarLink.addEventListener('click', insertLink);
        if (toolbarMermaid) toolbarMermaid.addEventListener('click', insertMermaid);

        // Create context menu element
        createContextMenuElement();

        // Close context menu on outside click / Escape
        document.addEventListener('click', function() { hideContextMenu(); });
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') hideContextMenu();
        });

        // Listen for git status updates
        document.addEventListener('clawide-git-status-update', function(e) {
            if (e.detail && e.detail.type === 'notes') {
                renderGitUI();
            }
        });

        // Load initial notes and git status
        loadAllNotes();
        loadGitStatus();
    }

    function setScope(newScope) {
        scope = newScope;
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
        loadAllNotes();
    }

    // ─── Git Status ───

    function loadGitStatus() {
        if (!projectID || typeof ClawIDEGit === 'undefined') return;
        ClawIDEGit.fetchStatus('notes', projectID).then(function() {
            gitStatusLoaded = true;
            renderGitUI();
        });
    }

    function renderGitUI() {
        if (typeof ClawIDEGit === 'undefined') return;

        var bannerEl = document.getElementById('notes-git-banner');
        if (bannerEl) {
            bannerEl.innerHTML = ClawIDEGit.renderWarningBanner('notes');
        }

        var toolbarEl = document.getElementById('notes-git-toolbar');
        if (toolbarEl) {
            var status = ClawIDEGit.getCachedStatus('notes');
            if (status && status.is_git_repo && !status.is_ignored) {
                toolbarEl.innerHTML = ClawIDEGit.renderRefreshButton('notes') +
                    ClawIDEGit.renderCommitButton('notes');
                toolbarEl.style.display = '';
            } else {
                toolbarEl.innerHTML = '';
                toolbarEl.style.display = 'none';
            }
        }
    }

    // ─── Notes CRUD ───

    // Load ALL notes (no folder filter) and feed them to the tree as files
    function loadAllNotes() {
        var params = [];
        var pid = scope === 'project' ? projectID : '';
        if (pid) params.push('project_id=' + encodeURIComponent(pid));

        var url = API_BASE;
        if (params.length > 0) url += '?' + params.join('&');

        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                notes = data || [];
                // Feed notes as files to the folder tree
                if (folderTree) {
                    folderTree.setFiles(notes);
                    // Re-select current file if still present
                    if (currentNoteID) {
                        folderTree.selectedFileID = currentNoteID;
                        folderTree.render();
                    }
                }
            })
            .catch(function(err) {
                console.error('Notes: failed to load:', err);
            });
    }

    function selectNote(id) {
        // Auto-save current note if dirty before switching
        if (isDirty && currentNoteID) {
            autoSave();
        }

        var note = findNote(id);
        if (!note) return;

        currentNoteID = id;
        if (titleInput) titleInput.value = note.title;
        if (contentInput) contentInput.value = note.content;
        if (formTitleEl) formTitleEl.textContent = 'Editing';
        if (cancelBtn) cancelBtn.style.display = '';
        if (deleteNoteBtn) deleteNoteBtn.style.display = '';
        isDirty = false;
        updateSaveIndicator('idle');

        // Update preview if in preview mode
        updatePreview();
    }

    function saveNote(overrideTitle, overrideFolderID) {
        var title = overrideTitle || (titleInput ? titleInput.value.trim() : '');
        var content = contentInput ? contentInput.value : '';
        if (!title) return;

        var titleError = validateTitle(title);
        if (titleError) {
            showTitleError(titleError);
            return;
        }

        var method, url;
        if (currentNoteID) {
            method = 'PUT';
            url = API_BASE + '/' + currentNoteID;
        } else {
            method = 'POST';
            url = API_BASE;
        }

        var body = { title: title, content: content };
        if (!currentNoteID) {
            body.project_id = scope === 'project' ? projectID : '';
            body.folder_id = (overrideFolderID !== undefined ? overrideFolderID : currentFolderID) || '';
        }

        updateSaveIndicator('saving');

        fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Save failed');
            return r.json();
        })
        .then(function(saved) {
            isDirty = false;
            updateSaveIndicator('saved');
            if (!currentNoteID) {
                currentNoteID = saved.id;
                if (formTitleEl) formTitleEl.textContent = 'Editing';
                if (cancelBtn) cancelBtn.style.display = '';
                if (deleteNoteBtn) deleteNoteBtn.style.display = '';
            }
            loadAllNotes();
        })
        .catch(function(err) {
            console.error('Notes: save failed:', err);
            updateSaveIndicator('idle');
        });
    }

    function autoSave() {
        if (!currentNoteID || !isDirty) return;

        var title = titleInput ? titleInput.value.trim() : '';
        var content = contentInput ? contentInput.value : '';
        if (!title) return;

        updateSaveIndicator('saving');

        fetch(API_BASE + '/' + currentNoteID, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ title: title, content: content })
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Auto-save failed');
            return r.json();
        })
        .then(function() {
            isDirty = false;
            updateSaveIndicator('saved');
            loadAllNotes();
        })
        .catch(function(err) {
            console.error('Notes: auto-save failed:', err);
            updateSaveIndicator('idle');
        });
    }

    function scheduleSave() {
        clearTimeout(saveTimer);
        if (!currentNoteID) return;
        saveTimer = setTimeout(autoSave, AUTOSAVE_DELAY);
    }

    function deleteNote(id) {
        var doDelete = function() {
            fetch(API_BASE + '/' + id, { method: 'DELETE' })
                .then(function(r) {
                    if (!r.ok) throw new Error('Delete failed');
                    if (currentNoteID === id) resetForm();
                    loadAllNotes();
                })
                .catch(function(err) {
                    console.error('Notes: delete failed:', err);
                });
        };

        if (typeof ClawIDEDialog !== 'undefined') {
            ClawIDEDialog.confirm('Delete Note', 'Are you sure you want to delete this note?', { destructive: true }).then(function(ok) {
                if (ok) doDelete();
            });
        } else {
            doDelete();
        }
    }

    function moveNoteToFolder(noteID, folderID) {
        var note = findNote(noteID);
        if (!note) return;

        fetch(API_BASE + '/' + noteID, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                project_id: scope === 'project' ? projectID : '',
                title: note.title,
                content: note.content,
                folder_id: folderID
            })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            loadAllNotes();
            if (folderTree) folderTree.load();
        })
        .catch(function(err) {
            console.error('Notes: move failed:', err);
        });
    }

    function moveFolderToParent(folderID, parentID) {
        var url = API_BASE + '/folders/' + folderID + '?project_id=' + encodeURIComponent(projectID);
        fetch(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ parent_id: parentID })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            if (folderTree) folderTree.load();
            loadAllNotes();
        })
        .catch(function(err) {
            console.error('Notes: move folder failed:', err);
        });
    }

    function resetForm() {
        currentNoteID = null;
        isDirty = false;
        clearTimeout(saveTimer);
        if (titleInput) titleInput.value = '';
        if (contentInput) contentInput.value = '';
        if (formTitleEl) formTitleEl.textContent = 'New Note';
        if (cancelBtn) cancelBtn.style.display = 'none';
        if (deleteNoteBtn) deleteNoteBtn.style.display = 'none';
        updateSaveIndicator('idle');
        updatePreview();
    }

    // ─── Context Menu ───

    function createContextMenuElement() {
        contextMenuEl = document.createElement('div');
        contextMenuEl.id = 'notes-context-menu';
        contextMenuEl.className = 'context-menu';
        contextMenuEl.style.display = 'none';
        document.body.appendChild(contextMenuEl);
    }

    function showContextMenu(type, id, event) {
        if (!contextMenuEl) return;
        var items = [];

        if (type === 'file') {
            // Right-click on a note file
            items = [
                { label: 'Rename', icon: 'edit', action: function() { promptRenameNote(id); } },
                { separator: true },
                { label: 'Delete', icon: 'delete', danger: true, action: function() { deleteNote(id); } }
            ];
        } else if (type === 'folder' && id) {
            // Right-click on a folder (non-root)
            items = [
                { label: 'New Note', icon: 'plus', action: function() { promptNewFile(id); } },
                { label: 'New Folder', icon: 'folder', action: function() { promptNewFolder(id); } },
                { separator: true },
                { label: 'Rename', icon: 'edit', action: function() { promptRenameFolder(id); } },
                { separator: true },
                { label: 'Delete', icon: 'delete', danger: true, action: function() {
                    if (typeof ClawIDEDialog !== 'undefined') {
                        ClawIDEDialog.confirm('Delete Folder', 'Delete this folder? Notes will be moved to root.', { destructive: true }).then(function(ok) {
                            if (ok && folderTree) folderTree.deleteFolder(id).then(loadAllNotes);
                        });
                    }
                }}
            ];
        } else {
            // Right-click on root or empty area
            items = [
                { label: 'New Note', icon: 'plus', action: function() { promptNewFile(''); } },
                { label: 'New Folder', icon: 'folder', action: function() { promptNewFolder(''); } }
            ];
        }

        renderContextMenu(event, items);
    }

    function renderContextMenu(e, items) {
        if (!contextMenuEl) return;

        var icons = {
            edit: '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
            'delete': '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
            plus: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>',
            folder: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>'
        };

        var html = '';
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            if (item.separator) {
                html += '<div class="context-menu-separator"></div>';
                continue;
            }
            var cls = 'context-menu-item' + (item.danger ? ' context-menu-danger' : '');
            html += '<button class="' + cls + '" data-ctx-idx="' + i + '">';
            html += (icons[item.icon] || '');
            html += '<span>' + escapeHTML(item.label) + '</span>';
            html += '</button>';
        }
        contextMenuEl.innerHTML = html;
        contextMenuEl.style.display = 'block';

        var menuW = 180;
        var menuH = contextMenuEl.offsetHeight || 150;
        var x = e.clientX;
        var y = e.clientY;
        if (x + menuW > window.innerWidth) x = window.innerWidth - menuW - 8;
        if (y + menuH > window.innerHeight) y = window.innerHeight - menuH - 8;
        contextMenuEl.style.left = x + 'px';
        contextMenuEl.style.top = y + 'px';

        var buttons = contextMenuEl.querySelectorAll('[data-ctx-idx]');
        for (var j = 0; j < buttons.length; j++) {
            buttons[j].addEventListener('click', function() {
                var idx = parseInt(this.getAttribute('data-ctx-idx'), 10);
                hideContextMenu();
                if (items[idx] && items[idx].action) items[idx].action();
            });
        }
    }

    function hideContextMenu() {
        if (contextMenuEl) contextMenuEl.style.display = 'none';
    }

    // ─── Prompt Helpers ───

    function promptNewFolder(parentID) {
        if (typeof ClawIDEDialog === 'undefined') return;
        ClawIDEDialog.prompt('New Folder', 'Folder name', '').then(function(name) {
            if (name && name.trim() && folderTree) {
                folderTree.createFolder(name.trim(), parentID || '').then(function() {
                    loadAllNotes();
                });
            }
        });
    }

    function promptRenameFolder(folderID) {
        if (typeof ClawIDEDialog === 'undefined' || !folderTree) return;
        var folder = folderTree.getFolder(folderID);
        var current = folder ? folder.name : '';
        ClawIDEDialog.prompt('Rename Folder', 'Folder name', current).then(function(name) {
            if (name && name.trim() && name.trim() !== current) {
                folderTree.renameFolder(folderID, name.trim());
            }
        });
    }

    function promptNewFile(folderID) {
        if (typeof ClawIDEDialog === 'undefined') return;
        ClawIDEDialog.prompt('New Note', 'File name', '', { suffix: '.md' }).then(function(name) {
            if (!name || !name.trim()) return;
            var title = name.trim();
            // Remove .md suffix if user added it
            if (title.toLowerCase().endsWith('.md')) {
                title = title.substring(0, title.length - 3);
            }
            if (!title) return;

            var titleError = validateTitle(title);
            if (titleError) {
                if (typeof ClawIDENotifications !== 'undefined') {
                    ClawIDENotifications.toast(titleError, 'error');
                }
                return;
            }

            // Create the note via API and then open it
            var body = {
                title: title,
                content: '',
                project_id: scope === 'project' ? projectID : '',
                folder_id: folderID || ''
            };

            fetch(API_BASE, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(body)
            })
            .then(function(r) {
                if (!r.ok) throw new Error('Create failed');
                return r.json();
            })
            .then(function(saved) {
                currentNoteID = saved.id;
                if (titleInput) titleInput.value = saved.title;
                if (contentInput) contentInput.value = '';
                if (formTitleEl) formTitleEl.textContent = 'Editing';
                if (cancelBtn) cancelBtn.style.display = '';
                if (deleteNoteBtn) deleteNoteBtn.style.display = '';
                isDirty = false;
                updateSaveIndicator('idle');
                if (contentInput) contentInput.focus();
                loadAllNotes();
            })
            .catch(function(err) {
                console.error('Notes: create failed:', err);
            });
        });
    }

    function promptRenameNote(noteID) {
        if (typeof ClawIDEDialog === 'undefined') return;
        var note = findNote(noteID);
        if (!note) return;
        ClawIDEDialog.prompt('Rename Note', 'Note title', note.title).then(function(name) {
            if (!name || !name.trim() || name.trim() === note.title) return;

            var titleError = validateTitle(name.trim());
            if (titleError) {
                if (typeof ClawIDENotifications !== 'undefined') {
                    ClawIDENotifications.toast(titleError, 'error');
                }
                return;
            }

            fetch(API_BASE + '/' + noteID, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ title: name.trim(), content: note.content })
            })
            .then(function(r) {
                if (!r.ok) throw new Error('Rename failed');
                return r.json();
            })
            .then(function() {
                // Update title in editor if this note is open
                if (currentNoteID === noteID && titleInput) {
                    titleInput.value = name.trim();
                }
                loadAllNotes();
            })
            .catch(function(err) {
                console.error('Notes: rename failed:', err);
            });
        });
    }

    // ─── Preview ───

    function updatePreview() {
        if (!previewEl || !contentInput) return;
        var content = contentInput.value || '';
        if (typeof ClawIDEMarkdown !== 'undefined') {
            ClawIDEMarkdown.renderInto(previewEl, content);
        } else {
            previewEl.innerHTML = content ? escapeHTML(content).replace(/\n/g, '<br>') : '<span class="text-th-text-faint italic">Nothing to preview</span>';
        }
    }

    function onModeChange(isEditMode) {
        if (!isEditMode) {
            updatePreview();
        }
    }

    // ─── Save Indicator ───

    var saveFadeTimer = null;

    function updateSaveIndicator(state) {
        saveState = state;
        if (!saveIndicator) return;
        clearTimeout(saveFadeTimer);

        // Ensure transition style is set
        saveIndicator.style.transition = 'opacity 0.4s ease';

        if (state === 'saving') {
            saveIndicator.innerHTML = '<span class="flex items-center gap-1 text-[10px] text-amber-400"><svg class="w-3 h-3 animate-spin" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>Saving...</span>';
            saveIndicator.style.display = '';
            saveIndicator.style.opacity = '1';
        } else if (state === 'saved') {
            saveIndicator.innerHTML = '<span class="flex items-center gap-1 text-[10px] text-green-400"><svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>Saved</span>';
            saveIndicator.style.display = '';
            saveIndicator.style.opacity = '1';
            saveFadeTimer = setTimeout(function() {
                if (saveState === 'saved') {
                    saveIndicator.style.opacity = '0';
                    // Hide after fade completes
                    saveFadeTimer = setTimeout(function() {
                        if (saveState === 'saved') {
                            saveIndicator.style.display = 'none';
                        }
                    }, 400);
                }
            }, 1500);
        } else {
            saveIndicator.style.opacity = '0';
            saveIndicator.style.display = 'none';
        }
    }

    // ─── Toolbar Helpers ───

    function wrapSelection(before, after) {
        if (!contentInput) return;
        var start = contentInput.selectionStart;
        var end = contentInput.selectionEnd;
        var text = contentInput.value;
        var selected = text.substring(start, end);

        if (!selected) {
            var placeholder = before === '**' ? 'bold text' : (before === '*' ? 'italic text' : 'text');
            contentInput.value = text.substring(0, start) + before + placeholder + after + text.substring(end);
            contentInput.selectionStart = start + before.length;
            contentInput.selectionEnd = start + before.length + placeholder.length;
        } else {
            contentInput.value = text.substring(0, start) + before + selected + after + text.substring(end);
            contentInput.selectionStart = start + before.length;
            contentInput.selectionEnd = start + before.length + selected.length;
        }
        contentInput.focus();
        isDirty = true;
        scheduleSave();
    }

    function insertLink() {
        if (!contentInput) return;
        var start = contentInput.selectionStart;
        var end = contentInput.selectionEnd;
        var text = contentInput.value;
        var selected = text.substring(start, end);

        var linkText = selected || 'link text';
        var insert = '[' + linkText + '](https://)';
        contentInput.value = text.substring(0, start) + insert + text.substring(end);

        var urlStart = start + linkText.length + 3;
        contentInput.selectionStart = urlStart;
        contentInput.selectionEnd = urlStart + 8;
        contentInput.focus();
        isDirty = true;
        scheduleSave();
    }

    function insertMermaid() {
        if (!contentInput) return;
        var start = contentInput.selectionStart;
        var text = contentInput.value;

        var template = '\n```mermaid\ngraph TD\n    A[Start] --> B{Decision}\n    B -->|Yes| C[Action]\n    B -->|No| D[End]\n```\n';

        contentInput.value = text.substring(0, start) + template + text.substring(start);
        contentInput.selectionStart = start + 1;
        contentInput.selectionEnd = start + template.length - 1;
        contentInput.focus();
        isDirty = true;
        scheduleSave();
    }

    // ─── Helpers ───

    function findNote(id) {
        for (var i = 0; i < notes.length; i++) {
            if (notes[i].id === id) return notes[i];
        }
        return null;
    }

    function escapeHTML(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str || ''));
        return div.innerHTML;
    }

    function escapeAttr(str) {
        return (str || '').replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // ─── Init ───

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose for external use
    window.ClawIDENotes = {
        setScope: setScope,
        reload: function() { if (!container) init(); else loadAllNotes(); },
        selectNote: selectNote,
        updatePreview: updatePreview,
        onModeChange: onModeChange,
        refreshGitStatus: loadGitStatus
    };
})();
