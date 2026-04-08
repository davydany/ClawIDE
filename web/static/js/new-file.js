// ClawIDE New File — Modal dialog for creating new files and folders
// Includes autocomplete, context menu (new/rename/delete), and keyboard navigation
(function() {
    'use strict';

    var config = {
        projectID: '',
        featureID: null,
        filesAPI: '',
        fileAPI: '',
        mkdirAPI: '',
        renameAPI: '',
        fileTreeId: 'file-tree',
        onFileCreated: null,
        onRenamed: null,
        onDeleted: null,
        refreshDir: null,
    };

    var modalEl = null;
    var inputEl = null;
    var autocompleteEl = null;
    var errorEl = null;
    var createBtnEl = null;
    var isFolder = false;
    var suggestions = [];
    var selectedSuggestionIndex = -1;
    var contextMenuEl = null;
    var longPressTimer = null;

    // --- Validation ---
    function validatePath(path) {
        if (!path || !path.trim()) return 'Path cannot be empty';
        if (/\.\./.test(path)) return 'Path cannot contain ".."';
        if (/[\x00-\x1f]/.test(path)) return 'Path contains invalid characters';
        if (!isFolder && path.charAt(path.length - 1) === '/') return 'File path cannot end with "/"';
        return null;
    }

    // --- Autocomplete ---
    function getDirectoryPart(path) {
        var lastSlash = path.lastIndexOf('/');
        if (lastSlash === -1) return '';
        return path.substring(0, lastSlash);
    }

    function getFilenamePart(path) {
        var lastSlash = path.lastIndexOf('/');
        if (lastSlash === -1) return path;
        return path.substring(lastSlash + 1);
    }

    function fetchSuggestions(dirPath) {
        var showHidden = localStorage.getItem('editor.preferences.showHidden') !== 'false';
        var url = config.filesAPI + '?hidden=' + showHidden + (dirPath ? '&path=' + encodeURIComponent(dirPath) : '');
        fetch(url)
            .then(function(r) {
                if (!r.ok) {
                    suggestions = [];
                    renderAutocomplete();
                    return;
                }
                return r.json();
            })
            .then(function(files) {
                if (!files) return;
                suggestions = files;
                renderAutocomplete();
            })
            .catch(function() {
                suggestions = [];
                renderAutocomplete();
            });
    }

    function renderAutocomplete() {
        if (!autocompleteEl || !inputEl) return;

        var currentPath = inputEl.value;
        var dirPart = getDirectoryPart(currentPath);
        var partial = getFilenamePart(currentPath).toLowerCase();

        var filtered = suggestions.filter(function(f) {
            if (!partial) return true;
            return f.name.toLowerCase().indexOf(partial) === 0;
        });

        if (filtered.length === 0) {
            autocompleteEl.style.display = 'none';
            selectedSuggestionIndex = -1;
            return;
        }

        autocompleteEl.innerHTML = '';
        autocompleteEl.style.display = 'block';
        selectedSuggestionIndex = -1;

        filtered.forEach(function(f, idx) {
            var item = document.createElement('div');
            item.className = 'new-file-suggestion';
            item.dataset.index = idx;
            var icon = f.is_dir ? '\u25B6 ' : '\u25AB ';
            item.textContent = icon + f.name;
            item.addEventListener('mousedown', function(e) {
                e.preventDefault();
                applySuggestion(f, dirPart);
            });
            autocompleteEl.appendChild(item);
        });
    }

    function applySuggestion(file, dirPart) {
        if (!inputEl) return;
        var prefix = dirPart ? dirPart + '/' : '';
        if (file.is_dir) {
            inputEl.value = prefix + file.name + '/';
            fetchSuggestions(prefix + file.name);
        } else {
            inputEl.value = prefix + file.name;
            autocompleteEl.style.display = 'none';
        }
        inputEl.focus();
    }

    function navigateSuggestions(direction) {
        if (!autocompleteEl || autocompleteEl.style.display === 'none') return;
        var items = autocompleteEl.querySelectorAll('.new-file-suggestion');
        if (items.length === 0) return;

        if (direction === 'down') {
            selectedSuggestionIndex = Math.min(selectedSuggestionIndex + 1, items.length - 1);
        } else {
            selectedSuggestionIndex = Math.max(selectedSuggestionIndex - 1, -1);
        }

        items.forEach(function(el, i) {
            if (i === selectedSuggestionIndex) {
                el.classList.add('selected');
            } else {
                el.classList.remove('selected');
            }
        });
    }

    function acceptSelectedSuggestion() {
        if (selectedSuggestionIndex < 0) return false;
        var items = autocompleteEl.querySelectorAll('.new-file-suggestion');
        if (selectedSuggestionIndex >= items.length) return false;

        var dirPart = getDirectoryPart(inputEl.value);
        var filtered = suggestions.filter(function(f) {
            var partial = getFilenamePart(inputEl.value).toLowerCase();
            if (!partial) return true;
            return f.name.toLowerCase().indexOf(partial) === 0;
        });

        if (filtered[selectedSuggestionIndex]) {
            applySuggestion(filtered[selectedSuggestionIndex], dirPart);
            return true;
        }
        return false;
    }

    // --- Modal ---
    function buildModal() {
        if (modalEl) return;

        // Backdrop
        var backdrop = document.createElement('div');
        backdrop.className = 'new-file-backdrop';
        backdrop.addEventListener('click', function() { closeModal(); });

        // Card
        var card = document.createElement('div');
        card.className = 'new-file-card';
        card.addEventListener('click', function(e) { e.stopPropagation(); });

        // Title
        var title = document.createElement('h3');
        title.className = 'text-sm font-semibold text-white mb-3';
        title.id = 'new-file-title';
        card.appendChild(title);

        // Input
        var input = document.createElement('input');
        input.type = 'text';
        input.className = 'new-file-input';
        input.placeholder = 'path/to/file.txt';
        input.autocomplete = 'off';
        input.spellcheck = false;
        inputEl = input;
        card.appendChild(input);

        // Autocomplete dropdown
        var ac = document.createElement('div');
        ac.className = 'new-file-autocomplete';
        ac.style.display = 'none';
        autocompleteEl = ac;
        card.appendChild(ac);

        // Error
        var err = document.createElement('div');
        err.className = 'new-file-error';
        err.style.display = 'none';
        errorEl = err;
        card.appendChild(err);

        // Buttons
        var btnRow = document.createElement('div');
        btnRow.className = 'flex items-center justify-end gap-2 mt-3';

        var cancelBtn = document.createElement('button');
        cancelBtn.className = 'px-3 py-1.5 text-xs text-gray-400 hover:text-white rounded transition-colors';
        cancelBtn.textContent = 'Cancel';
        cancelBtn.addEventListener('click', function() { closeModal(); });
        btnRow.appendChild(cancelBtn);

        var createBtn = document.createElement('button');
        createBtn.className = 'px-3 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded transition-colors font-medium';
        createBtn.textContent = 'Create';
        createBtn.addEventListener('click', function() { doCreate(); });
        createBtnEl = createBtn;
        btnRow.appendChild(createBtn);

        card.appendChild(btnRow);

        // Modal wrapper
        var modal = document.createElement('div');
        modal.className = 'new-file-modal';
        modal.style.display = 'none';
        modal.appendChild(backdrop);
        modal.appendChild(card);
        modalEl = modal;

        document.body.appendChild(modal);

        // Input events
        input.addEventListener('input', function() {
            hideError();
            var dirPart = getDirectoryPart(input.value);
            fetchSuggestions(dirPart);
        });

        input.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                e.preventDefault();
                if (autocompleteEl.style.display !== 'none') {
                    autocompleteEl.style.display = 'none';
                    selectedSuggestionIndex = -1;
                } else {
                    closeModal();
                }
                return;
            }
            if (e.key === 'ArrowDown') {
                e.preventDefault();
                navigateSuggestions('down');
                return;
            }
            if (e.key === 'ArrowUp') {
                e.preventDefault();
                navigateSuggestions('up');
                return;
            }
            if (e.key === 'Tab') {
                e.preventDefault();
                if (!acceptSelectedSuggestion()) {
                    // Auto-complete first suggestion
                    var dirPart = getDirectoryPart(input.value);
                    var partial = getFilenamePart(input.value).toLowerCase();
                    var filtered = suggestions.filter(function(f) {
                        if (!partial) return true;
                        return f.name.toLowerCase().indexOf(partial) === 0;
                    });
                    if (filtered.length > 0) {
                        applySuggestion(filtered[0], dirPart);
                    }
                }
                return;
            }
            if (e.key === 'Enter') {
                e.preventDefault();
                if (selectedSuggestionIndex >= 0 && autocompleteEl.style.display !== 'none') {
                    acceptSelectedSuggestion();
                } else {
                    doCreate();
                }
                return;
            }
        });
    }

    function showError(msg) {
        if (!errorEl) return;
        errorEl.textContent = msg;
        errorEl.style.display = 'block';
    }

    function hideError() {
        if (!errorEl) return;
        errorEl.style.display = 'none';
    }

    function openModal(prefillPath) {
        isFolder = false;
        buildModal();

        var titleEl = modalEl.querySelector('#new-file-title');
        if (titleEl) titleEl.textContent = 'New File';
        inputEl.placeholder = 'path/to/file.txt';
        inputEl.value = prefillPath || '';
        hideError();
        autocompleteEl.style.display = 'none';
        selectedSuggestionIndex = -1;
        suggestions = [];

        modalEl.style.display = '';
        closeContextMenu();

        setTimeout(function() {
            inputEl.focus();
            if (prefillPath) {
                fetchSuggestions(getDirectoryPart(prefillPath));
            }
        }, 50);
    }

    function openFolderModal(prefillPath) {
        isFolder = true;
        buildModal();

        var titleEl = modalEl.querySelector('#new-file-title');
        if (titleEl) titleEl.textContent = 'New Folder';
        inputEl.placeholder = 'path/to/folder';
        inputEl.value = prefillPath || '';
        hideError();
        autocompleteEl.style.display = 'none';
        selectedSuggestionIndex = -1;
        suggestions = [];

        modalEl.style.display = '';
        closeContextMenu();

        setTimeout(function() {
            inputEl.focus();
            if (prefillPath) {
                fetchSuggestions(getDirectoryPart(prefillPath));
            }
        }, 50);
    }

    function closeModal() {
        if (modalEl) modalEl.style.display = 'none';
        suggestions = [];
        selectedSuggestionIndex = -1;
    }

    // --- Create file/folder ---
    function doCreate() {
        if (!inputEl) return;
        var path = inputEl.value.trim();

        var err = validatePath(path);
        if (err) {
            showError(err);
            return;
        }

        if (isFolder) {
            createFolder(path);
        } else {
            createFile(path);
        }
    }

    function createFile(path) {
        var url = config.fileAPI + '?path=' + encodeURIComponent(path);
        fetch(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'text/plain' },
            body: '',
        })
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to create file (HTTP ' + resp.status + ')');
                closeModal();
                var dirPart = getDirectoryPart(path);
                if (config.refreshDir) config.refreshDir(dirPart);
                if (config.onFileCreated) config.onFileCreated(path);
            })
            .catch(function(err) {
                showError(err.message || 'Failed to create file');
            });
    }

    function createFolder(path) {
        // Remove trailing slash for the API call
        var cleanPath = path.replace(/\/+$/, '');
        var url = config.mkdirAPI + '?path=' + encodeURIComponent(cleanPath);
        fetch(url, { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to create folder (HTTP ' + resp.status + ')');
                closeModal();
                var dirPart = getDirectoryPart(cleanPath);
                if (config.refreshDir) config.refreshDir(dirPart);
            })
            .catch(function(err) {
                showError(err.message || 'Failed to create folder');
            });
    }

    // --- Rename ---
    function doRename(itemPath, itemIsDir) {
        if (!window.ClawIDEDialog) return;

        var currentName = getFilenamePart(itemPath);
        var typeLabel = itemIsDir ? 'folder' : 'file';

        window.ClawIDEDialog.prompt('Rename ' + typeLabel, 'New name:', currentName).then(function(newName) {
            if (!newName || newName === currentName) return;

            // Validate new name
            if (/\//.test(newName) || /\.\./.test(newName) || /[\x00-\x1f]/.test(newName)) {
                return; // silently reject invalid names
            }

            var parentDir = getDirectoryPart(itemPath);
            var newPath = parentDir ? parentDir + '/' + newName : newName;

            var url = config.renameAPI + '?path=' + encodeURIComponent(itemPath) + '&newPath=' + encodeURIComponent(newPath);
            fetch(url, { method: 'POST' })
                .then(function(resp) {
                    if (resp.status === 409) throw new Error('A file or folder with that name already exists');
                    if (!resp.ok) throw new Error('Failed to rename (HTTP ' + resp.status + ')');
                    // Refresh the parent directory in the tree
                    if (config.refreshDir) config.refreshDir(parentDir);
                    if (config.onRenamed) config.onRenamed(itemPath, newPath);
                })
                .catch(function(err) {
                    if (window.ClawIDEDialog) {
                        window.ClawIDEDialog.confirm('Rename Failed', err.message || 'Failed to rename', { confirmLabel: 'OK' });
                    }
                });
        });
    }

    // --- Delete ---
    function doDelete(itemPath, itemIsDir) {
        if (!window.ClawIDEDialog) return;

        var name = getFilenamePart(itemPath);
        var typeLabel = itemIsDir ? 'folder' : 'file';
        var message = 'Delete ' + typeLabel + ' "' + name + '"?' + (itemIsDir ? ' This will delete all contents.' : '');

        window.ClawIDEDialog.confirm('Delete ' + typeLabel, message, { destructive: true, confirmLabel: 'Delete' }).then(function(confirmed) {
            if (!confirmed) return;

            var url = config.fileAPI + '?path=' + encodeURIComponent(itemPath);
            fetch(url, { method: 'DELETE' })
                .then(function(resp) {
                    if (!resp.ok) throw new Error('Failed to delete (HTTP ' + resp.status + ')');
                    var parentDir = getDirectoryPart(itemPath);
                    if (config.refreshDir) config.refreshDir(parentDir);
                    if (config.onDeleted) config.onDeleted(itemPath, itemIsDir);
                })
                .catch(function(err) {
                    if (window.ClawIDEDialog) {
                        window.ClawIDEDialog.confirm('Delete Failed', err.message || 'Failed to delete', { confirmLabel: 'OK' });
                    }
                });
        });
    }

    // --- Context menu ---
    function buildContextMenu() {
        if (contextMenuEl) return;

        var menu = document.createElement('div');
        menu.className = 'file-tree-context-menu';
        menu.style.display = 'none';

        // New File
        var newFileItem = document.createElement('div');
        newFileItem.className = 'file-tree-context-menu-item';
        newFileItem.dataset.action = 'newFile';
        newFileItem.textContent = 'New File';
        newFileItem.addEventListener('click', function() {
            var path = contextMenuEl.dataset.folderPath || '';
            closeContextMenu();
            openModal(path ? path + '/' : '');
        });
        menu.appendChild(newFileItem);

        // New Folder
        var newFolderItem = document.createElement('div');
        newFolderItem.className = 'file-tree-context-menu-item';
        newFolderItem.dataset.action = 'newFolder';
        newFolderItem.textContent = 'New Folder';
        newFolderItem.addEventListener('click', function() {
            var path = contextMenuEl.dataset.folderPath || '';
            closeContextMenu();
            openFolderModal(path ? path + '/' : '');
        });
        menu.appendChild(newFolderItem);

        // Separator
        var sep1 = document.createElement('div');
        sep1.className = 'file-tree-context-menu-separator';
        menu.appendChild(sep1);

        // Rename
        var renameItem = document.createElement('div');
        renameItem.className = 'file-tree-context-menu-item';
        renameItem.dataset.action = 'rename';
        renameItem.textContent = 'Rename';
        renameItem.addEventListener('click', function() {
            var itemPath = contextMenuEl.dataset.itemPath;
            var itemIsDir = contextMenuEl.dataset.itemIsDir === 'true';
            closeContextMenu();
            if (itemPath) doRename(itemPath, itemIsDir);
        });
        menu.appendChild(renameItem);

        // Separator
        var sep2 = document.createElement('div');
        sep2.className = 'file-tree-context-menu-separator';
        menu.appendChild(sep2);

        // Delete
        var deleteItem = document.createElement('div');
        deleteItem.className = 'file-tree-context-menu-item file-tree-context-menu-item-danger';
        deleteItem.dataset.action = 'delete';
        deleteItem.textContent = 'Delete';
        deleteItem.addEventListener('click', function() {
            var itemPath = contextMenuEl.dataset.itemPath;
            var itemIsDir = contextMenuEl.dataset.itemIsDir === 'true';
            closeContextMenu();
            if (itemPath) doDelete(itemPath, itemIsDir);
        });
        menu.appendChild(deleteItem);

        contextMenuEl = menu;
        document.body.appendChild(menu);
    }

    function showContextMenu(x, y, itemPath, itemIsDir) {
        buildContextMenu();
        contextMenuEl.dataset.itemPath = itemPath || '';
        contextMenuEl.dataset.itemIsDir = itemIsDir ? 'true' : 'false';
        // For new file/folder, use folder path (the item if dir, or parent if file)
        contextMenuEl.dataset.folderPath = itemIsDir ? itemPath : getDirectoryPart(itemPath);

        // Show/hide "New File" and "New Folder" only for directories
        var items = contextMenuEl.querySelectorAll('.file-tree-context-menu-item');
        items.forEach(function(el) {
            var action = el.dataset.action;
            if (action === 'newFile' || action === 'newFolder') {
                el.style.display = itemIsDir ? '' : 'none';
            }
        });
        // Show/hide first separator (between new items and rename)
        var seps = contextMenuEl.querySelectorAll('.file-tree-context-menu-separator');
        if (seps[0]) seps[0].style.display = itemIsDir ? '' : 'none';

        contextMenuEl.style.left = x + 'px';
        contextMenuEl.style.top = y + 'px';
        contextMenuEl.style.display = 'block';

        // Ensure menu stays within viewport
        requestAnimationFrame(function() {
            var rect = contextMenuEl.getBoundingClientRect();
            if (rect.right > window.innerWidth) {
                contextMenuEl.style.left = (window.innerWidth - rect.width - 4) + 'px';
            }
            if (rect.bottom > window.innerHeight) {
                contextMenuEl.style.top = (window.innerHeight - rect.height - 4) + 'px';
            }
        });
    }

    function closeContextMenu() {
        if (contextMenuEl) contextMenuEl.style.display = 'none';
    }

    function setupContextMenu() {
        var treeEl = document.getElementById(config.fileTreeId);
        if (!treeEl) return;

        // Right-click on any file tree item (files and folders)
        treeEl.addEventListener('contextmenu', function(e) {
            // Match folders (data-path) or files (data-filepath)
            var item = e.target.closest('.file-tree-item');
            if (!item) return;
            e.preventDefault();

            var isDir = item.hasAttribute('data-path');
            var itemPath = isDir ? item.dataset.path : item.dataset.filepath;
            if (!itemPath) return;

            showContextMenu(e.clientX, e.clientY, itemPath, isDir);
        });

        // Long-press for mobile (500ms)
        treeEl.addEventListener('touchstart', function(e) {
            var item = e.target.closest('.file-tree-item');
            if (!item) return;

            longPressTimer = setTimeout(function() {
                e.preventDefault();
                var touch = e.touches[0];
                var isDir = item.hasAttribute('data-path');
                var itemPath = isDir ? item.dataset.path : item.dataset.filepath;
                if (itemPath) {
                    showContextMenu(touch.clientX, touch.clientY, itemPath, isDir);
                }
            }, 500);
        }, { passive: false });

        treeEl.addEventListener('touchend', function() {
            clearTimeout(longPressTimer);
        });

        treeEl.addEventListener('touchmove', function() {
            clearTimeout(longPressTimer);
        });

        // Close context menu on click elsewhere
        document.addEventListener('click', function(e) {
            if (contextMenuEl && !contextMenuEl.contains(e.target)) {
                closeContextMenu();
            }
        });

        document.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') closeContextMenu();
        });
    }

    // --- Init ---
    function init(opts) {
        config.projectID = opts.projectID || '';
        config.featureID = opts.featureID || null;
        config.filesAPI = opts.filesAPI || '';
        config.fileAPI = opts.fileAPI || '';
        config.mkdirAPI = opts.mkdirAPI || '';
        config.renameAPI = opts.renameAPI || '';
        config.fileTreeId = opts.fileTreeId || 'file-tree';
        config.onFileCreated = opts.onFileCreated || null;
        config.onRenamed = opts.onRenamed || null;
        config.onDeleted = opts.onDeleted || null;
        config.refreshDir = opts.refreshDir || null;

        setupContextMenu();
    }

    // --- Public API ---
    window.ClawIDENewFile = {
        init: init,
        openModal: openModal,
        openFolderModal: openFolderModal,
        closeModal: closeModal,
    };
})();
