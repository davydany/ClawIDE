// ClawIDE Bookmarks Manager
// Full-featured bookmarks panel: folder tree, drag-and-drop, context menus,
// toggleable bookmark bar, and CRUD.  Replaces the legacy bookmarks.js.
(function() {
    'use strict';

    var API_BASE = '/api/bookmarks';
    var FOLDER_API = API_BASE + '/folders';
    var debounceTimer = null;
    var bookmarks = [];
    var folders = [];
    var editingID = null;
    var projectID = '';
    var selectedFolderID = '';
    var emojiPickerVisible = false;
    var emojiPickerLoaded = false;
    var folderTree = null;
    var contextMenuEl = null;
    var gitStatusLoaded = false;

    // Drag state
    var dragSourceID = null;
    var dragSourceType = null; // 'bookmark' or 'folder'

    // DOM refs
    var container, list, searchInput, form, nameInput, urlInput, emojiInput;
    var formTitle, cancelBtn, emojiBtn, emojiPickerEl;
    var bookmarkBarContent, folderTreeContainer;

    // ─── Initialization ───

    function init() {
        container = document.getElementById('bookmarks-container');
        list = document.getElementById('bookmarks-list');
        searchInput = document.getElementById('bookmarks-search');
        form = document.getElementById('bookmarks-form');
        nameInput = document.getElementById('bookmarks-name');
        urlInput = document.getElementById('bookmarks-url');
        emojiInput = document.getElementById('bookmarks-emoji');
        formTitle = document.getElementById('bookmarks-form-title');
        cancelBtn = document.getElementById('bookmarks-cancel');
        emojiBtn = document.getElementById('bookmarks-emoji-btn');
        emojiPickerEl = document.getElementById('bookmarks-emoji-picker');
        bookmarkBarContent = document.getElementById('bookmarks-bar-content');
        folderTreeContainer = document.getElementById('bookmarks-folder-tree');

        if (!container) return;
        projectID = container.getAttribute('data-project-id') || '';

        // Search with debounce
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(loadBookmarks, 250);
            });
        }

        // Form submit
        if (form) {
            form.addEventListener('submit', function(e) {
                e.preventDefault();
                saveBookmark();
            });
        }

        // Cancel edit
        if (cancelBtn) {
            cancelBtn.addEventListener('click', resetForm);
        }

        // Emoji picker
        if (emojiBtn) {
            emojiBtn.addEventListener('click', function(e) {
                e.stopPropagation();
                toggleEmojiPicker();
            });
        }
        if (emojiPickerEl) {
            emojiPickerEl.addEventListener('emoji-click', function(e) {
                if (emojiInput && e.detail && e.detail.unicode) {
                    emojiInput.value = e.detail.unicode;
                }
                hideEmojiPicker();
            });
        }

        // Close emoji picker on outside click / Escape
        document.addEventListener('click', function(e) {
            if (emojiPickerVisible && emojiPickerEl && !emojiPickerEl.contains(e.target) && e.target !== emojiBtn) {
                hideEmojiPicker();
            }
            hideContextMenu();
        });
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                if (emojiPickerVisible) hideEmojiPicker();
                hideContextMenu();
            }
        });

        // Initialize folder tree
        initFolderTree();

        // Favicon preview next to URL input
        if (urlInput) {
            var faviconPreview = document.createElement('img');
            faviconPreview.id = 'bookmarks-favicon-preview';
            faviconPreview.className = 'w-5 h-5 flex-shrink-0 ml-1';
            faviconPreview.style.display = 'none';
            faviconPreview.onerror = function() { this.style.display = 'none'; };
            urlInput.parentNode.style.display = 'flex';
            urlInput.parentNode.style.alignItems = 'center';
            urlInput.style.flex = '1';
            urlInput.parentNode.appendChild(faviconPreview);

            var faviconTimer = null;
            urlInput.addEventListener('input', function() {
                clearTimeout(faviconTimer);
                faviconTimer = setTimeout(function() {
                    var fUrl = getFaviconUrl(urlInput.value.trim());
                    if (fUrl) {
                        faviconPreview.src = fUrl;
                        faviconPreview.style.display = '';
                    } else {
                        faviconPreview.style.display = 'none';
                    }
                }, 400);
            });
        }

        // Persist bookmark bar visibility
        var savedBarState = localStorage.getItem('clawide-bookmark-bar');
        if (savedBarState !== null) {
            // Alpine manages visibility; we sync via Alpine's showBookmarkBar
            try {
                var alpineData = Alpine.$data(container.closest('[x-data]'));
                if (alpineData && savedBarState === 'false') {
                    alpineData.showBookmarkBar = false;
                }
            } catch (e) { /* ignore */ }
        }

        // Watch for bookmark bar toggle to persist
        observeBookmarkBarToggle();

        // Create context menu container
        createContextMenuElement();

        // Listen for git status updates
        document.addEventListener('clawide-git-status-update', function(e) {
            if (e.detail && e.detail.type === 'bookmarks') {
                renderGitUI();
                renderList();
            }
        });

        // Initial load
        loadBookmarks();
        loadGitStatus();
    }

    // ─── Folder Tree ───

    function initFolderTree() {
        if (!folderTreeContainer || typeof FolderTree === 'undefined') return;

        // Clear default content and create tree container
        var treeDiv = document.createElement('div');
        treeDiv.id = 'bookmarks-folder-tree-inner';
        folderTreeContainer.insertBefore(treeDiv, folderTreeContainer.firstChild);

        folderTree = new FolderTree({
            container: treeDiv,
            projectID: projectID,
            apiBase: FOLDER_API,
            allowDrag: true,
            onSelect: function(folderID) {
                selectedFolderID = folderID;
                loadBookmarks();
            },
            onDrop: function(itemID, targetFolderID, itemType) {
                if (itemType === 'folder') {
                    // Move folder under new parent
                    moveFolder(itemID, targetFolderID);
                } else {
                    // Move bookmark to folder
                    moveBookmarkToFolder(itemID, targetFolderID);
                }
            }
        });
        folderTree.load();
    }

    function moveFolder(folderID, newParentID) {
        fetch(FOLDER_API + '/' + folderID + '?project_id=' + encodeURIComponent(projectID), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ parent_id: newParentID })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            if (folderTree) folderTree.load();
        })
        .catch(function(err) {
            console.error('Failed to move folder:', err);
        });
    }

    function moveBookmarkToFolder(bookmarkID, folderID) {
        var url = API_BASE + '/' + bookmarkID;
        if (projectID) url += '?project_id=' + encodeURIComponent(projectID);

        fetch(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ project_id: projectID, folder_id: folderID })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            loadBookmarks();
        })
        .catch(function(err) {
            console.error('Failed to move bookmark:', err);
        });
    }

    // ─── Git Status ───

    function loadGitStatus() {
        if (!projectID || typeof ClawIDEGit === 'undefined') return;
        ClawIDEGit.fetchStatus('bookmarks', projectID).then(function() {
            gitStatusLoaded = true;
            renderGitUI();
            renderList();
        });
    }

    function renderGitUI() {
        if (typeof ClawIDEGit === 'undefined') return;

        // Render warning banner
        var bannerEl = document.getElementById('bookmarks-git-banner');
        if (bannerEl) {
            bannerEl.innerHTML = ClawIDEGit.renderWarningBanner('bookmarks');
        }

        // Render git toolbar (refresh + commit button)
        var toolbarEl = document.getElementById('bookmarks-git-toolbar');
        if (toolbarEl) {
            var status = ClawIDEGit.getCachedStatus('bookmarks');
            if (status && status.is_git_repo && !status.is_ignored) {
                toolbarEl.innerHTML = ClawIDEGit.renderRefreshButton('bookmarks') +
                    ClawIDEGit.renderCommitButton('bookmarks');
                toolbarEl.style.display = '';
            } else {
                toolbarEl.innerHTML = '';
                toolbarEl.style.display = 'none';
            }
        }
    }

    // ─── Bookmark CRUD ───

    function loadBookmarks() {
        var params = [];
        if (projectID) {
            params.push('project_id=' + encodeURIComponent(projectID));
        }
        var query = searchInput ? searchInput.value.trim() : '';
        if (query) {
            params.push('q=' + encodeURIComponent(query));
        } else if (selectedFolderID !== undefined) {
            params.push('folder_id=' + encodeURIComponent(selectedFolderID));
        }

        var url = API_BASE;
        if (params.length > 0) url += '?' + params.join('&');

        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                bookmarks = data || [];
                renderList();
                updateBookmarkBar();
            })
            .catch(function(err) {
                console.error('Failed to load bookmarks:', err);
            });
    }

    function saveBookmark() {
        var name = nameInput.value.trim();
        var bookmarkUrl = urlInput.value.trim();
        var emoji = emojiInput ? emojiInput.value.trim() : '';

        if (!name || !bookmarkUrl) return;

        var method, url;
        if (editingID) {
            method = 'PUT';
            url = API_BASE + '/' + editingID;
            if (projectID) url += '?project_id=' + encodeURIComponent(projectID);
        } else {
            method = 'POST';
            url = API_BASE;
        }

        var body = { name: name, url: bookmarkUrl, emoji: emoji };
        if (!editingID) {
            body.project_id = projectID;
            body.folder_id = selectedFolderID || '';
        }

        fetch(url, {
            method: method,
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body)
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            resetForm();
            loadBookmarks();
        })
        .catch(function(err) {
            console.error('Failed to save bookmark:', err);
        });
    }

    function startEdit(id) {
        var bm = findBookmark(id);
        if (!bm) return;

        editingID = id;
        if (nameInput) nameInput.value = bm.name;
        if (urlInput) urlInput.value = bm.url;
        if (emojiInput) emojiInput.value = bm.emoji || '';
        if (formTitle) formTitle.textContent = 'Edit Bookmark';
        if (cancelBtn) cancelBtn.style.display = '';
    }

    function deleteBookmark(id) {
        var doDelete = function() {
            var url = API_BASE + '/' + id;
            if (projectID) url += '?project_id=' + encodeURIComponent(projectID);

            fetch(url, { method: 'DELETE' })
                .then(function(r) {
                    if (!r.ok) throw new Error('Delete failed');
                    loadBookmarks();
                })
                .catch(function(err) {
                    console.error('Failed to delete bookmark:', err);
                });
        };

        if (typeof ClawIDEDialog !== 'undefined') {
            ClawIDEDialog.confirm('Delete Bookmark', 'Are you sure you want to delete this bookmark?', { destructive: true }).then(function(ok) {
                if (ok) doDelete();
            });
        } else {
            doDelete();
        }
    }

    function toggleInBar(id) {
        var url = API_BASE + '/' + id + '/star';
        if (projectID) url += '?project_id=' + encodeURIComponent(projectID);

        fetch(url, { method: 'PATCH' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function() {
                loadBookmarks();
            })
            .catch(function(err) {
                console.error('Failed to toggle bar:', err);
            });
    }

    function resetForm() {
        editingID = null;
        if (nameInput) nameInput.value = '';
        if (urlInput) urlInput.value = '';
        if (emojiInput) emojiInput.value = '';
        if (formTitle) formTitle.textContent = 'New Bookmark';
        if (cancelBtn) cancelBtn.style.display = 'none';
        hideEmojiPicker();
    }

    function findBookmark(id) {
        for (var i = 0; i < bookmarks.length; i++) {
            if (bookmarks[i].id === id) return bookmarks[i];
        }
        return null;
    }

    // ─── Rendering ───

    function renderList() {
        if (!list) return;

        if (bookmarks.length === 0) {
            var msg = selectedFolderID ? 'No bookmarks in this folder' : 'No bookmarks yet';
            list.innerHTML = '<div class="text-gray-500 text-xs p-3 text-center">' + msg + '</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < bookmarks.length; i++) {
            var bm = bookmarks[i];
            var domain = getDomain(bm.url);
            var faviconUrl = getFaviconUrl(bm.url);
            var inBar = bm.in_bar || bm.starred;
            var barClass = inBar ? 'text-yellow-400' : 'text-gray-600 hover:text-yellow-400';
            var barFill = inBar ? 'currentColor' : 'none';

            html += '<div class="bookmark-item group" data-id="' + escapeAttr(bm.id) + '" draggable="true">';
            html += '  <div class="flex items-center gap-2">';

            // Bar toggle (star icon)
            html += '    <button class="flex-shrink-0 p-0.5 rounded transition-colors ' + barClass + '" title="' + (inBar ? 'Remove from bar' : 'Add to bar') + '" data-bookmark-bar="' + escapeAttr(bm.id) + '">';
            html += '      <svg class="w-3 h-3" fill="' + barFill + '" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"/></svg>';
            html += '    </button>';

            // Favicon or emoji
            if (bm.emoji) {
                html += '    <span class="text-sm flex-shrink-0">' + escapeHTML(bm.emoji) + '</span>';
            } else if (faviconUrl) {
                html += '    <img src="' + escapeAttr(faviconUrl) + '" class="w-4 h-4 flex-shrink-0" onerror="this.style.display=\'none\'">';
            }

            // Name as link
            html += '    <a href="' + escapeAttr(bm.url) + '" target="_blank" rel="noopener" class="text-xs text-white font-medium truncate hover:text-indigo-300 transition-colors" title="' + escapeAttr(bm.url) + '">' + escapeHTML(bm.name) + '</a>';

            // Git status badge (bookmarks share a single file)
            if (gitStatusLoaded && typeof ClawIDEGit !== 'undefined') {
                var bmPath = '.clawide/bookmarks/bookmarks.json';
                var gitSt = ClawIDEGit.getFileStatus('bookmarks', bmPath);
                if (gitSt) {
                    html += '    ' + ClawIDEGit.renderBadge(gitSt);
                }
            }

            // Hover actions
            html += '    <div class="flex items-center gap-1 flex-shrink-0 ml-auto opacity-0 group-hover:opacity-100 transition-opacity">';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-white transition-colors" title="Edit" data-bookmark-edit="' + escapeAttr(bm.id) + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>';
            html += '      </button>';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-red-400 transition-colors" title="Delete" data-bookmark-delete="' + escapeAttr(bm.id) + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
            html += '      </button>';
            html += '    </div>';

            html += '  </div>';
            html += '  <div class="text-[10px] text-gray-500 mt-0.5 truncate pl-6">' + escapeHTML(domain) + '</div>';
            html += '</div>';
        }
        list.innerHTML = html;
        bindListEvents();
    }

    function bindListEvents() {
        if (!list) return;

        // Bar toggle (star)
        var barBtns = list.querySelectorAll('[data-bookmark-bar]');
        for (var i = 0; i < barBtns.length; i++) {
            barBtns[i].addEventListener('click', function(e) {
                e.preventDefault();
                toggleInBar(this.getAttribute('data-bookmark-bar'));
            });
        }

        // Edit
        var editBtns = list.querySelectorAll('[data-bookmark-edit]');
        for (var j = 0; j < editBtns.length; j++) {
            editBtns[j].addEventListener('click', function() {
                startEdit(this.getAttribute('data-bookmark-edit'));
            });
        }

        // Delete
        var deleteBtns = list.querySelectorAll('[data-bookmark-delete]');
        for (var k = 0; k < deleteBtns.length; k++) {
            deleteBtns[k].addEventListener('click', function() {
                deleteBookmark(this.getAttribute('data-bookmark-delete'));
            });
        }

        // Drag and drop for bookmark items
        var items = list.querySelectorAll('.bookmark-item');
        for (var m = 0; m < items.length; m++) {
            bindDragEvents(items[m]);
        }

        // Context menu
        var allItems = list.querySelectorAll('.bookmark-item');
        for (var n = 0; n < allItems.length; n++) {
            allItems[n].addEventListener('contextmenu', function(e) {
                e.preventDefault();
                var id = this.getAttribute('data-id');
                showBookmarkContextMenu(e, id);
            });
        }
    }

    // ─── Drag and Drop ───

    function bindDragEvents(el) {
        el.addEventListener('dragstart', function(e) {
            var id = this.getAttribute('data-id');
            dragSourceID = id;
            dragSourceType = 'bookmark';
            e.dataTransfer.setData('text/tree-item-id', id);
            e.dataTransfer.setData('text/item-type', 'bookmark');
            e.dataTransfer.effectAllowed = 'move';
            this.classList.add('bookmark-dragging');
        });

        el.addEventListener('dragend', function() {
            this.classList.remove('bookmark-dragging');
            clearDropIndicators();
            dragSourceID = null;
            dragSourceType = null;
        });

        el.addEventListener('dragover', function(e) {
            e.preventDefault();
            if (!dragSourceID) return;
            e.dataTransfer.dropEffect = 'move';

            // Show drop indicator
            var rect = this.getBoundingClientRect();
            var midY = rect.top + rect.height / 2;
            clearDropIndicators();
            if (e.clientY < midY) {
                this.classList.add('bookmark-drop-above');
            } else {
                this.classList.add('bookmark-drop-below');
            }
        });

        el.addEventListener('dragleave', function() {
            this.classList.remove('bookmark-drop-above', 'bookmark-drop-below');
        });

        el.addEventListener('drop', function(e) {
            e.preventDefault();
            clearDropIndicators();
            var targetID = this.getAttribute('data-id');
            var sourceID = e.dataTransfer.getData('text/tree-item-id');
            var itemType = e.dataTransfer.getData('text/item-type');
            if (!sourceID || sourceID === targetID || itemType !== 'bookmark') return;

            // Reorder: build new order
            var ids = [];
            for (var i = 0; i < bookmarks.length; i++) {
                ids.push(bookmarks[i].id);
            }

            // Remove source from current position
            var srcIdx = ids.indexOf(sourceID);
            if (srcIdx === -1) return;
            ids.splice(srcIdx, 1);

            // Insert at target position
            var rect = this.getBoundingClientRect();
            var midY = rect.top + rect.height / 2;
            var targetIdx = ids.indexOf(targetID);
            if (e.clientY >= midY) targetIdx++;
            ids.splice(targetIdx, 0, sourceID);

            reorderBookmarks(ids);
        });
    }

    function clearDropIndicators() {
        if (!list) return;
        var items = list.querySelectorAll('.bookmark-item');
        for (var i = 0; i < items.length; i++) {
            items[i].classList.remove('bookmark-drop-above', 'bookmark-drop-below');
        }
    }

    function reorderBookmarks(bookmarkIDs) {
        fetch(API_BASE + '/reorder', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                project_id: projectID,
                bookmark_ids: bookmarkIDs
            })
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Reorder failed');
            loadBookmarks();
        })
        .catch(function(err) {
            console.error('Failed to reorder bookmarks:', err);
        });
    }

    // ─── Bookmark Bar ───

    function updateBookmarkBar() {
        if (!bookmarkBarContent) return;

        var barItems = [];
        for (var i = 0; i < bookmarks.length; i++) {
            if (bookmarks[i].in_bar || bookmarks[i].starred) {
                barItems.push(bookmarks[i]);
            }
        }

        // Also fetch all bookmarks to get bar items from other folders
        if (selectedFolderID) {
            fetchAllBarBookmarks();
            return;
        }

        renderBarItems(barItems);
    }

    function fetchAllBarBookmarks() {
        var url = API_BASE + '?project_id=' + encodeURIComponent(projectID);
        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                var barItems = [];
                for (var i = 0; i < (data || []).length; i++) {
                    if (data[i].in_bar || data[i].starred) {
                        barItems.push(data[i]);
                    }
                }
                renderBarItems(barItems);
            })
            .catch(function() {});
    }

    function renderBarItems(barItems) {
        if (!bookmarkBarContent) return;

        barItems.sort(function(a, b) {
            if (a.order !== b.order) return (a.order || 0) - (b.order || 0);
            return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
        });

        if (barItems.length === 0) {
            bookmarkBarContent.innerHTML = '<span class="text-xs text-gray-500 italic">No bookmarks in bar</span>';
            return;
        }

        var html = '';
        for (var j = 0; j < barItems.length; j++) {
            var bm = barItems[j];
            var faviconUrl = getFaviconUrl(bm.url);

            html += '<a href="' + escapeAttr(bm.url) + '" target="_blank" rel="noopener"';
            html += '   class="bookmark-bar-item flex items-center gap-1 px-2 py-1 text-xs text-gray-400 hover:text-white transition-colors rounded hover:bg-gray-800"';
            html += '   title="' + escapeAttr(bm.name) + '"';
            html += '   data-bar-id="' + escapeAttr(bm.id) + '">';

            if (bm.emoji) {
                html += '<span class="text-sm">' + escapeHTML(bm.emoji) + '</span>';
            } else if (faviconUrl) {
                html += '<img src="' + escapeAttr(faviconUrl) + '" class="w-4 h-4" onerror="this.style.display=\'none\'">';
            } else {
                html += '<svg class="w-3.5 h-3.5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>';
            }

            html += '<span class="max-w-[80px] truncate">' + escapeHTML(bm.name) + '</span>';
            html += '</a>';
        }
        bookmarkBarContent.innerHTML = html;

        // Context menu for bar items
        var barLinks = bookmarkBarContent.querySelectorAll('.bookmark-bar-item');
        for (var k = 0; k < barLinks.length; k++) {
            barLinks[k].addEventListener('contextmenu', function(e) {
                e.preventDefault();
                var id = this.getAttribute('data-bar-id');
                showBookmarkContextMenu(e, id);
            });
        }
    }

    function observeBookmarkBarToggle() {
        // Watch Alpine state changes to persist bar visibility
        var interval = setInterval(function() {
            try {
                var el = container ? container.closest('[x-data]') : null;
                if (!el) return;
                var data = Alpine.$data(el);
                if (data) {
                    var current = data.showBookmarkBar;
                    localStorage.setItem('clawide-bookmark-bar', current ? 'true' : 'false');
                }
            } catch (e) { /* Alpine not ready */ }
        }, 1000);
        // Clean up on page unload
        window.addEventListener('beforeunload', function() {
            clearInterval(interval);
        });
    }

    // ─── Context Menu ───

    function createContextMenuElement() {
        contextMenuEl = document.createElement('div');
        contextMenuEl.id = 'bookmarks-context-menu';
        contextMenuEl.className = 'context-menu';
        contextMenuEl.style.display = 'none';
        document.body.appendChild(contextMenuEl);
    }

    function showBookmarkContextMenu(e, bookmarkID) {
        var bm = findBookmark(bookmarkID);
        if (!bm || !contextMenuEl) return;

        var inBar = bm.in_bar || bm.starred;
        var items = [
            { label: 'Open in new tab', icon: 'external', action: function() { window.open(bm.url, '_blank'); } },
            { label: 'Edit', icon: 'edit', action: function() { startEdit(bookmarkID); } },
            { separator: true },
            { label: inBar ? 'Remove from bar' : 'Add to bar', icon: 'star', action: function() { toggleInBar(bookmarkID); } },
            { separator: true },
            { label: 'Delete', icon: 'delete', danger: true, action: function() { deleteBookmark(bookmarkID); } }
        ];

        renderContextMenu(e, items);
    }

    function showFolderContextMenu(e, folderID) {
        if (!contextMenuEl) return;

        var items = [
            { label: 'Rename', icon: 'edit', action: function() { promptRenameFolder(folderID); } },
            { label: 'New Bookmark', icon: 'plus', action: function() {
                selectedFolderID = folderID;
                if (folderTree) folderTree.select(folderID);
                if (nameInput) nameInput.focus();
            }},
            { label: 'New Subfolder', icon: 'folder', action: function() { promptNewFolder(folderID); } },
            { separator: true },
            { label: 'Delete', icon: 'delete', danger: true, action: function() {
                if (typeof ClawIDEDialog !== 'undefined') {
                    ClawIDEDialog.confirm('Delete Folder', 'Delete this folder? Bookmarks will be moved to root.', { destructive: true }).then(function(ok) {
                        if (ok && folderTree) folderTree.deleteFolder(folderID).then(loadBookmarks);
                    });
                }
            }}
        ];

        renderContextMenu(e, items);
    }

    function renderContextMenu(e, items) {
        if (!contextMenuEl) return;

        var html = '';
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            if (item.separator) {
                html += '<div class="context-menu-separator"></div>';
                continue;
            }
            var cls = 'context-menu-item' + (item.danger ? ' context-menu-danger' : '');
            html += '<button class="' + cls + '" data-ctx-idx="' + i + '">';
            html += getContextMenuIcon(item.icon);
            html += '<span>' + escapeHTML(item.label) + '</span>';
            html += '</button>';
        }
        contextMenuEl.innerHTML = html;
        contextMenuEl.style.display = 'block';

        // Position near cursor, keep within viewport
        var menuW = 180;
        var menuH = contextMenuEl.offsetHeight || 150;
        var x = e.clientX;
        var y = e.clientY;
        if (x + menuW > window.innerWidth) x = window.innerWidth - menuW - 8;
        if (y + menuH > window.innerHeight) y = window.innerHeight - menuH - 8;
        contextMenuEl.style.left = x + 'px';
        contextMenuEl.style.top = y + 'px';

        // Bind click handlers
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

    function getContextMenuIcon(name) {
        var icons = {
            external: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"/></svg>',
            edit: '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>',
            star: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"/></svg>',
            'delete': '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>',
            plus: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>',
            folder: '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z"/></svg>'
        };
        return icons[name] || '';
    }

    // ─── Folder Prompts ───

    function promptNewFolder(parentID) {
        if (typeof ClawIDEDialog === 'undefined') return;
        ClawIDEDialog.prompt('New Folder', 'Folder name', '').then(function(name) {
            if (!name || !name.trim()) return;
            if (folderTree) {
                folderTree.createFolder(name.trim(), parentID || '');
            }
        });
    }

    function promptRenameFolder(folderID) {
        if (typeof ClawIDEDialog === 'undefined') return;
        var folder = folderTree ? folderTree.getFolder(folderID) : null;
        var current = folder ? folder.name : '';
        ClawIDEDialog.prompt('Rename Folder', 'Folder name', current).then(function(name) {
            if (!name || !name.trim() || name.trim() === current) return;
            if (folderTree) {
                folderTree.renameFolder(folderID, name.trim());
            }
        });
    }

    // ─── Emoji Picker ───

    function toggleEmojiPicker() {
        if (emojiPickerVisible) hideEmojiPicker();
        else showEmojiPicker();
    }

    var FALLBACK_EMOJIS = [
        '\u{1F600}','\u{1F60D}','\u{1F60E}','\u{1F914}','\u{1F4A1}','\u{1F525}','\u{2B50}','\u{1F680}',
        '\u{1F4BB}','\u{1F4F1}','\u{1F310}','\u{1F512}','\u{1F4E6}','\u{1F4DA}','\u{1F4DD}','\u{2705}',
        '\u{274C}','\u{26A0}','\u{1F6A7}','\u{1F3E0}','\u{1F4C1}','\u{1F4CA}','\u{1F4C8}','\u{1F527}',
        '\u{2699}','\u{1F50D}','\u{1F4AC}','\u{1F4E7}','\u{1F465}','\u{1F3AF}','\u{1F3C6}','\u{1F389}',
        '\u{2764}','\u{1F44D}','\u{1F44E}','\u{270F}','\u{1F4CC}','\u{1F516}','\u{1F4D6}','\u{1F30D}'
    ];

    function loadEmojiPickerScript() {
        if (emojiPickerLoaded) return Promise.resolve();
        return new Promise(function(resolve, reject) {
            var timer = setTimeout(function() {
                reject(new Error('Emoji picker load timed out'));
            }, 5000);

            var script = document.createElement('script');
            script.type = 'module';
            script.src = 'https://cdn.jsdelivr.net/npm/emoji-picker-element@^1/index.js';
            script.onload = function() {
                clearTimeout(timer);
                emojiPickerLoaded = true;
                resolve();
            };
            script.onerror = function() {
                clearTimeout(timer);
                reject(new Error('Failed to load emoji picker'));
            };
            document.head.appendChild(script);
        });
    }

    function showFallbackEmojiGrid() {
        if (!emojiPickerEl) return;
        var html = '<div class="bg-gray-900 border border-gray-700 rounded-lg p-2 shadow-xl" style="width:240px">';
        html += '<div class="text-[10px] text-gray-500 mb-1.5 px-1">Quick Emojis</div>';
        html += '<div class="grid grid-cols-8 gap-0.5">';
        for (var i = 0; i < FALLBACK_EMOJIS.length; i++) {
            html += '<button type="button" class="emoji-fallback-btn p-1 text-lg hover:bg-gray-800 rounded cursor-pointer text-center leading-none" data-emoji="' + FALLBACK_EMOJIS[i] + '">' + FALLBACK_EMOJIS[i] + '</button>';
        }
        html += '</div></div>';

        // Replace the picker element with fallback
        var wrapper = emojiPickerEl.parentNode;
        var fallback = document.createElement('div');
        fallback.id = 'bookmarks-emoji-fallback';
        fallback.style.position = 'absolute';
        fallback.style.zIndex = '50';
        fallback.style.bottom = '100%';
        fallback.style.left = '0';
        fallback.innerHTML = html;
        wrapper.appendChild(fallback);

        // Bind click handlers
        var btns = fallback.querySelectorAll('.emoji-fallback-btn');
        for (var j = 0; j < btns.length; j++) {
            btns[j].addEventListener('click', function() {
                if (emojiInput) emojiInput.value = this.getAttribute('data-emoji');
                hideEmojiPicker();
            });
        }

        emojiPickerVisible = true;
    }

    function showEmojiPicker() {
        if (!emojiPickerEl) return;

        // Show loading spinner briefly
        emojiPickerEl.style.display = 'none';
        var loadingEl = document.createElement('div');
        loadingEl.id = 'emoji-loading-indicator';
        loadingEl.className = 'text-xs text-gray-500 py-1';
        loadingEl.textContent = 'Loading emoji picker...';
        emojiPickerEl.parentNode.appendChild(loadingEl);

        loadEmojiPickerScript().then(function() {
            var loading = document.getElementById('emoji-loading-indicator');
            if (loading) loading.remove();
            emojiPickerEl.style.display = 'block';
            emojiPickerVisible = true;
        }).catch(function() {
            var loading = document.getElementById('emoji-loading-indicator');
            if (loading) loading.remove();
            // Show fallback grid
            showFallbackEmojiGrid();
            if (typeof ClawIDENotifications !== 'undefined') {
                ClawIDENotifications.toast('Emoji picker unavailable, showing basic set', 'warning');
            }
        });
    }

    function hideEmojiPicker() {
        if (emojiPickerEl) emojiPickerEl.style.display = 'none';
        // Remove fallback grid if present
        var fallback = document.getElementById('bookmarks-emoji-fallback');
        if (fallback) fallback.remove();
        var loading = document.getElementById('emoji-loading-indicator');
        if (loading) loading.remove();
        emojiPickerVisible = false;
    }

    // ─── Helpers ───

    function getDomain(u) {
        try { return new URL(u).hostname; }
        catch(e) { return u; }
    }

    function getFaviconUrl(u) {
        try {
            var parsed = new URL(u);
            if (parsed.hostname) return 'https://www.google.com/s2/favicons?domain=' + parsed.hostname + '&sz=32';
        } catch(e) {}
        return '';
    }

    function escapeHTML(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    function escapeAttr(str) {
        if (!str) return '';
        return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // ─── Lifecycle ───

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Public API (backward-compatible with old bookmarks.js)
    window.ClawIDEBookmarks = {
        reload: function() { if (!container) init(); else loadBookmarks(); },
        updateTabBar: updateBookmarkBar,
        createFolder: function(name, parentID) {
            if (folderTree) return folderTree.createFolder(name, parentID);
        },
        showFolderContextMenu: showFolderContextMenu,
        promptNewFolder: promptNewFolder,
        refreshGitStatus: loadGitStatus
    };
})();
