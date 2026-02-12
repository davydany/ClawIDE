// ClawIDE Bookmarks Manager
// Sidebar-based bookmarks with search, CRUD, emoji picker, star toggle, and tab bar integration.
(function() {
    'use strict';

    var API_BASE = '/api/bookmarks';
    var debounceTimer = null;
    var bookmarks = [];
    var editingID = null;
    var projectID = '';
    var emojiPickerVisible = false;
    var emojiPickerLoaded = false;

    // DOM references
    var container, list, searchInput, form, nameInput, urlInput, emojiInput;
    var formTitle, cancelBtn, starredBar, emojiBtn, emojiPickerEl;

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
        starredBar = document.getElementById('starred-bookmarks-bar');
        emojiBtn = document.getElementById('bookmarks-emoji-btn');
        emojiPickerEl = document.getElementById('bookmarks-emoji-picker');

        if (!container) return;

        projectID = container.getAttribute('data-project-id') || '';

        // Search with debounce
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                clearTimeout(debounceTimer);
                debounceTimer = setTimeout(function() {
                    loadBookmarks();
                }, 250);
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

        // Emoji picker toggle
        if (emojiBtn) {
            emojiBtn.addEventListener('click', function(e) {
                e.stopPropagation();
                toggleEmojiPicker();
            });
        }

        // Listen for emoji selection
        if (emojiPickerEl) {
            emojiPickerEl.addEventListener('emoji-click', function(e) {
                if (emojiInput && e.detail && e.detail.unicode) {
                    emojiInput.value = e.detail.unicode;
                }
                hideEmojiPicker();
            });
        }

        // Close emoji picker on outside click
        document.addEventListener('click', function(e) {
            if (emojiPickerVisible && emojiPickerEl && !emojiPickerEl.contains(e.target) && e.target !== emojiBtn) {
                hideEmojiPicker();
            }
        });

        // Close emoji picker on Escape
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape' && emojiPickerVisible) {
                hideEmojiPicker();
            }
        });

        // Initial load
        loadBookmarks();
    }

    function toggleEmojiPicker() {
        if (emojiPickerVisible) {
            hideEmojiPicker();
        } else {
            showEmojiPicker();
        }
    }

    function loadEmojiPickerScript() {
        if (emojiPickerLoaded) return Promise.resolve();
        return new Promise(function(resolve, reject) {
            var script = document.createElement('script');
            script.type = 'module';
            script.src = 'https://cdn.jsdelivr.net/npm/emoji-picker-element@^1/index.js';
            script.onload = function() {
                emojiPickerLoaded = true;
                resolve();
            };
            script.onerror = function() {
                reject(new Error('Failed to load emoji picker'));
            };
            document.head.appendChild(script);
        });
    }

    function showEmojiPicker() {
        if (!emojiPickerEl) return;
        loadEmojiPickerScript().then(function() {
            emojiPickerEl.style.display = 'block';
            emojiPickerVisible = true;
        }).catch(function(err) {
            console.error('Emoji picker load failed:', err);
        });
    }

    function hideEmojiPicker() {
        if (!emojiPickerEl) return;
        emojiPickerEl.style.display = 'none';
        emojiPickerVisible = false;
    }

    function loadBookmarks() {
        var params = [];
        if (projectID) {
            params.push('project_id=' + encodeURIComponent(projectID));
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
                bookmarks = data || [];
                renderList();
                updateTabBar();
            })
            .catch(function(err) {
                console.error('Failed to load bookmarks:', err);
            });
    }

    function renderList() {
        if (!list) return;

        if (bookmarks.length === 0) {
            list.innerHTML = '<div class="text-gray-500 text-xs p-3 text-center">No bookmarks yet</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < bookmarks.length; i++) {
            var bm = bookmarks[i];
            var domain = getDomain(bm.url);
            var faviconUrl = getFaviconUrl(bm.url);
            var starClass = bm.starred ? 'text-yellow-400' : 'text-gray-600 hover:text-yellow-400';
            var starFill = bm.starred ? 'currentColor' : 'none';

            html += '<div class="bookmark-item group" data-id="' + bm.id + '">';
            html += '  <div class="flex items-center gap-2">';

            // Star toggle
            html += '    <button class="flex-shrink-0 p-0.5 rounded transition-colors ' + starClass + '" title="' + (bm.starred ? 'Unstar' : 'Star') + '" data-bookmark-star="' + bm.id + '">';
            html += '      <svg class="w-3 h-3" fill="' + starFill + '" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"/></svg>';
            html += '    </button>';

            // Favicon or emoji
            if (bm.emoji) {
                html += '    <span class="text-sm flex-shrink-0">' + escapeHTML(bm.emoji) + '</span>';
            } else if (faviconUrl) {
                html += '    <img src="' + escapeHTML(faviconUrl) + '" class="w-4 h-4 flex-shrink-0" onerror="this.style.display=\'none\'">';
            }

            // Name as link
            html += '    <a href="' + escapeHTML(bm.url) + '" target="_blank" rel="noopener" class="text-xs text-white font-medium truncate hover:text-indigo-300 transition-colors" title="' + escapeHTML(bm.url) + '">' + escapeHTML(bm.name) + '</a>';

            // Actions
            html += '    <div class="flex items-center gap-1 flex-shrink-0 ml-auto opacity-0 group-hover:opacity-100 transition-opacity">';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-white transition-colors" title="Edit" data-bookmark-edit="' + bm.id + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M11 4H4a2 2 0 00-2 2v14a2 2 0 002 2h14a2 2 0 002-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 013 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>';
            html += '      </button>';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-red-400 transition-colors" title="Delete" data-bookmark-delete="' + bm.id + '">';
            html += '        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
            html += '      </button>';
            html += '    </div>';

            html += '  </div>';
            html += '  <div class="text-[10px] text-gray-500 mt-0.5 truncate pl-6">' + escapeHTML(domain) + '</div>';
            html += '</div>';
        }
        list.innerHTML = html;

        // Bind action buttons
        var starBtns = list.querySelectorAll('[data-bookmark-star]');
        for (var j = 0; j < starBtns.length; j++) {
            starBtns[j].addEventListener('click', function(e) {
                e.preventDefault();
                toggleStar(this.getAttribute('data-bookmark-star'));
            });
        }

        var editBtns = list.querySelectorAll('[data-bookmark-edit]');
        for (var k = 0; k < editBtns.length; k++) {
            editBtns[k].addEventListener('click', function() {
                startEdit(this.getAttribute('data-bookmark-edit'));
            });
        }

        var deleteBtns = list.querySelectorAll('[data-bookmark-delete]');
        for (var l = 0; l < deleteBtns.length; l++) {
            deleteBtns[l].addEventListener('click', function() {
                deleteBookmark(this.getAttribute('data-bookmark-delete'));
            });
        }
    }

    function updateTabBar() {
        if (!starredBar) return;

        var starred = [];
        for (var i = 0; i < bookmarks.length; i++) {
            if (bookmarks[i].starred) {
                starred.push(bookmarks[i]);
            }
        }

        // Sort starred alphabetically
        starred.sort(function(a, b) {
            return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
        });

        if (starred.length === 0) {
            starredBar.innerHTML = '';
            return;
        }

        var html = '';
        for (var j = 0; j < starred.length; j++) {
            var bm = starred[j];
            var faviconUrl = getFaviconUrl(bm.url);

            html += '<a href="' + escapeHTML(bm.url) + '" target="_blank" rel="noopener"';
            html += '   class="flex items-center gap-1 px-2 py-1.5 text-xs text-gray-400 hover:text-white transition-colors rounded hover:bg-gray-800"';
            html += '   title="' + escapeHTML(bm.name) + '">';

            if (bm.emoji) {
                html += '<span class="text-sm">' + escapeHTML(bm.emoji) + '</span>';
            } else if (faviconUrl) {
                html += '<img src="' + escapeHTML(faviconUrl) + '" class="w-4 h-4" onerror="this.style.display=\'none\';this.nextElementSibling.style.display=\'block\'">';
                html += '<svg style="display:none" class="w-3.5 h-3.5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>';
            } else {
                html += '<svg class="w-3.5 h-3.5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1"/></svg>';
            }

            html += '<span class="max-w-[80px] truncate">' + escapeHTML(bm.name) + '</span>';
            html += '</a>';
        }
        starredBar.innerHTML = html;
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
        } else {
            method = 'POST';
            url = API_BASE;
        }

        var body = { name: name, url: bookmarkUrl, emoji: emoji };
        if (!editingID) {
            body.project_id = projectID;
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
        nameInput.value = bm.name;
        urlInput.value = bm.url;
        if (emojiInput) emojiInput.value = bm.emoji || '';
        if (formTitle) formTitle.textContent = 'Edit Bookmark';
        if (cancelBtn) cancelBtn.style.display = '';
    }

    function deleteBookmark(id) {
        if (!confirm('Delete this bookmark?')) return;

        fetch(API_BASE + '/' + id, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) throw new Error('Delete failed');
                loadBookmarks();
            })
            .catch(function(err) {
                console.error('Failed to delete bookmark:', err);
            });
    }

    function toggleStar(id) {
        fetch(API_BASE + '/' + id + '/star', { method: 'PATCH' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function() {
                loadBookmarks();
            })
            .catch(function(err) {
                console.error('Failed to toggle star:', err);
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

    function getDomain(u) {
        try {
            var parsed = new URL(u);
            return parsed.hostname;
        } catch(e) {
            return u;
        }
    }

    function getFaviconUrl(u) {
        try {
            var parsed = new URL(u);
            if (parsed.hostname) {
                return 'https://www.google.com/s2/favicons?domain=' + parsed.hostname + '&sz=32';
            }
        } catch(e) {}
        return '';
    }

    function escapeHTML(str) {
        if (!str) return '';
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
    window.ClawIDEBookmarks = {
        reload: loadBookmarks,
        updateTabBar: updateTabBar
    };
})();
