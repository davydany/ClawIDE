// ClawIDE Shared Folder Tree Component
// Reusable tree view for folder hierarchies with optional file items (bookmarks, notes).
(function() {
    'use strict';

    var MAX_DEPTH = 5;
    var LONG_PRESS_MS = 500;

    /**
     * FolderTree renders a recursive folder tree with expand/collapse, selection,
     * drag-drop reorder, optional file items, and context menu support.
     *
     * @param {Object} opts
     * @param {HTMLElement} opts.container    - DOM element to render into
     * @param {string}     opts.projectID    - current project UUID
     * @param {string}     opts.apiBase      - API prefix, e.g. '/api/bookmarks/folders'
     * @param {Function}  [opts.onSelect]    - callback(folderID) when a folder is clicked
     * @param {Function}  [opts.onFileSelect]- callback(fileID) when a file is clicked
     * @param {Function}  [opts.onDrop]      - callback(droppedItemID, targetFolderID, type) on DnD
     * @param {Function}  [opts.onContextMenu] - callback(type, id, event) for right-click/long-press
     * @param {boolean}   [opts.allowDrag]   - enable folder drag reorder (default false)
     * @param {string}    [opts.rootLabel]   - label for root node (default 'All')
     * @param {string}    [opts.fileIcon]    - 'document' (default) or 'bookmark'
     */
    function FolderTree(opts) {
        this.container = opts.container;
        this.projectID = opts.projectID;
        this.apiBase = opts.apiBase;
        this.onSelect = opts.onSelect || function() {};
        this.onFileSelect = opts.onFileSelect || function() {};
        this.onDrop = opts.onDrop || function() {};
        this.onContextMenu = opts.onContextMenu || null;
        this.allowDrag = opts.allowDrag || false;
        this.rootLabel = opts.rootLabel || 'All';
        this.fileIcon = opts.fileIcon || 'document';
        this.folders = [];
        this.files = []; // file items: { id, title/name, folder_id }
        this.expanded = {}; // folderID -> bool
        this.selectedID = ''; // '' = root / all
        this.selectedFileID = ''; // currently selected file
        this._filterText = '';
    }

    FolderTree.prototype.load = function() {
        var self = this;
        var url = this.apiBase + '?project_id=' + encodeURIComponent(this.projectID);
        return fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                self.folders = data || [];
                self.render();
            })
            .catch(function(err) {
                console.error('FolderTree load error:', err);
            });
    };

    FolderTree.prototype.setFiles = function(files) {
        this.files = files || [];
        this.render();
    };

    FolderTree.prototype.setFilter = function(text) {
        this._filterText = (text || '').toLowerCase().trim();
        this.render();
    };

    FolderTree.prototype.getFolder = function(id) {
        for (var i = 0; i < this.folders.length; i++) {
            if (this.folders[i].id === id) return this.folders[i];
        }
        return null;
    };

    FolderTree.prototype.select = function(folderID) {
        this.selectedID = folderID;
        this.selectedFileID = '';
        this.render();
        this.onSelect(folderID);
    };

    FolderTree.prototype.selectFile = function(fileID) {
        this.selectedFileID = fileID;
        this.render();
        this.onFileSelect(fileID);
    };

    FolderTree.prototype.toggleExpand = function(folderID) {
        this.expanded[folderID] = !this.expanded[folderID];
        this.render();
    };

    FolderTree.prototype.getDepth = function(folderID) {
        var depth = 0;
        var current = folderID;
        var seen = {};
        while (current) {
            if (seen[current]) break;
            seen[current] = true;
            depth++;
            var folder = this.getFolder(current);
            if (!folder) break;
            current = folder.parent_id;
        }
        return depth;
    };

    // ─── Filtering ───

    FolderTree.prototype._getVisibleFolderIDs = function() {
        if (!this._filterText) return null; // no filter, show all
        var matchFolders = {};
        var matchFiles = {};
        var self = this;

        // Check which files match
        for (var i = 0; i < this.files.length; i++) {
            var f = this.files[i];
            var name = (f.title || f.name || '').toLowerCase();
            if (name.indexOf(this._filterText) !== -1) {
                matchFiles[f.id] = true;
                // Walk up to root, marking parent folders visible
                var fid = f.folder_id || '';
                while (fid) {
                    matchFolders[fid] = true;
                    var folder = self.getFolder(fid);
                    if (!folder) break;
                    fid = folder.parent_id || '';
                }
            }
        }

        // Check which folders match by name
        for (var j = 0; j < this.folders.length; j++) {
            var fd = this.folders[j];
            if (fd.name.toLowerCase().indexOf(this._filterText) !== -1) {
                matchFolders[fd.id] = true;
                var pid = fd.parent_id || '';
                while (pid) {
                    matchFolders[pid] = true;
                    var parent = self.getFolder(pid);
                    if (!parent) break;
                    pid = parent.parent_id || '';
                }
            }
        }

        return { folders: matchFolders, files: matchFiles };
    };

    // ─── Rendering ───

    FolderTree.prototype.render = function() {
        if (!this.container) return;
        var filter = this._getVisibleFolderIDs();
        var html = '';

        // Root item
        var rootSelected = this.selectedID === '' && !this.selectedFileID;
        html += '<div class="folder-tree-node ' + (rootSelected ? 'folder-tree-selected' : '') + '" data-folder-id="" data-node-type="folder">';
        html += '  <div class="folder-tree-label" style="padding-left:8px">';
        html += '    <svg class="w-3.5 h-3.5 text-gray-400 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>';
        html += '    <span class="text-xs truncate">' + escapeHTML(this.rootLabel) + '</span>';
        html += '  </div>';
        html += '</div>';

        // Build children of root
        html += this._renderChildren('', 0, filter);
        this.container.innerHTML = html;
        this._bindEvents();
    };

    FolderTree.prototype._renderChildren = function(parentID, depth, filter) {
        var html = '';
        var children = [];
        for (var i = 0; i < this.folders.length; i++) {
            var f = this.folders[i];
            if ((f.parent_id || '') === parentID) {
                if (filter && !filter.folders[f.id]) continue;
                children.push(f);
            }
        }
        // Sort by order then name
        children.sort(function(a, b) {
            if (a.order !== b.order) return a.order - b.order;
            return a.name.toLowerCase().localeCompare(b.name.toLowerCase());
        });

        for (var j = 0; j < children.length; j++) {
            var folder = children[j];
            var isExpanded = !!this.expanded[folder.id] || (filter !== null);
            var isSelected = this.selectedID === folder.id && !this.selectedFileID;
            var hasChildren = this._hasChildren(folder.id, filter);
            var hasFileChildren = this._hasFileChildren(folder.id, filter);
            var showToggle = hasChildren || hasFileChildren;
            var indent = (depth + 1) * 16 + 8;

            html += '<div class="folder-tree-node ' + (isSelected ? 'folder-tree-selected' : '') + '"';
            html += '  data-folder-id="' + escapeAttr(folder.id) + '" data-node-type="folder"';
            if (this.allowDrag) {
                html += ' draggable="true"';
            }
            html += '>';
            html += '  <div class="folder-tree-label" style="padding-left:' + indent + 'px">';

            // Expand/collapse arrow
            if (showToggle) {
                html += '<button class="folder-tree-toggle" data-toggle-id="' + escapeAttr(folder.id) + '">';
                html += '  <svg class="w-3 h-3 transition-transform ' + (isExpanded ? 'rotate-90' : '') + '" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>';
                html += '</button>';
            } else {
                html += '<span class="w-3 h-3 inline-block"></span>';
            }

            // Folder icon
            html += '    <svg class="w-3.5 h-3.5 flex-shrink-0 ' + (isExpanded ? 'text-yellow-400' : 'text-gray-400') + '" fill="' + (isExpanded ? 'currentColor' : 'none') + '" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>';
            html += '    <span class="text-xs truncate">' + escapeHTML(folder.name) + '</span>';
            html += '  </div>';
            html += '</div>';

            // Render children if expanded
            if (isExpanded) {
                if (hasChildren) {
                    html += this._renderChildren(folder.id, depth + 1, filter);
                }
                // Render file items in this folder
                html += this._renderFiles(folder.id, depth + 1, filter);
            }
        }

        // Render root-level files when parentID is ''
        if (parentID === '') {
            html += this._renderFiles('', depth, filter);
        }

        return html;
    };

    FolderTree.prototype._renderFiles = function(folderID, depth, filter) {
        if (this.files.length === 0) return '';
        var html = '';
        var indent = (depth + 1) * 16 + 8;
        var fileItems = [];

        for (var i = 0; i < this.files.length; i++) {
            var file = this.files[i];
            var fileFolderID = file.folder_id || '';
            if (fileFolderID !== folderID) continue;
            if (filter && !filter.files[file.id]) continue;
            fileItems.push(file);
        }

        // Sort files by title/name
        fileItems.sort(function(a, b) {
            var na = (a.title || a.name || '').toLowerCase();
            var nb = (b.title || b.name || '').toLowerCase();
            return na.localeCompare(nb);
        });

        for (var j = 0; j < fileItems.length; j++) {
            var f = fileItems[j];
            var isFileSelected = this.selectedFileID === f.id;
            var fileName = f.title || f.name || 'Untitled';

            html += '<div class="folder-tree-node folder-tree-file ' + (isFileSelected ? 'folder-tree-selected' : '') + '"';
            html += '  data-file-id="' + escapeAttr(f.id) + '" data-node-type="file"';
            if (this.allowDrag) {
                html += ' draggable="true"';
            }
            html += '  style="cursor:pointer">';
            html += '  <div class="folder-tree-label" style="padding-left:' + indent + 'px">';
            html += '    <span class="w-3 h-3 inline-block"></span>'; // spacer to align with toggle
            // Document icon
            html += '    <svg class="w-3.5 h-3.5 flex-shrink-0 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>';
            html += '    <span class="text-xs truncate">' + escapeHTML(fileName) + '</span>';
            html += '  </div>';
            html += '</div>';
        }

        return html;
    };

    FolderTree.prototype._hasChildren = function(folderID, filter) {
        for (var i = 0; i < this.folders.length; i++) {
            var f = this.folders[i];
            if (f.parent_id === folderID) {
                if (!filter || filter.folders[f.id]) return true;
            }
        }
        return false;
    };

    FolderTree.prototype._hasFileChildren = function(folderID, filter) {
        for (var i = 0; i < this.files.length; i++) {
            var f = this.files[i];
            if ((f.folder_id || '') === folderID) {
                if (!filter || filter.files[f.id]) return true;
            }
        }
        return false;
    };

    // ─── Events ───

    FolderTree.prototype._bindEvents = function() {
        var self = this;
        var nodes = this.container.querySelectorAll('.folder-tree-node');

        for (var i = 0; i < nodes.length; i++) {
            (function(node) {
                var nodeType = node.getAttribute('data-node-type');
                var folderID = node.getAttribute('data-folder-id');
                var fileID = node.getAttribute('data-file-id');

                // Click to select
                var label = node.querySelector('.folder-tree-label');
                if (label) {
                    label.addEventListener('click', function(e) {
                        if (e.target.closest('.folder-tree-toggle')) return;
                        if (nodeType === 'file' && fileID) {
                            self.selectFile(fileID);
                        } else {
                            self.select(folderID || '');
                        }
                    });
                }

                // Toggle expand
                var toggleBtn = node.querySelector('.folder-tree-toggle');
                if (toggleBtn) {
                    toggleBtn.addEventListener('click', function(e) {
                        e.stopPropagation();
                        var id = this.getAttribute('data-toggle-id');
                        self.toggleExpand(id);
                    });
                }

                // Context menu (right-click)
                if (self.onContextMenu) {
                    node.addEventListener('contextmenu', function(e) {
                        e.preventDefault();
                        e.stopPropagation();
                        if (nodeType === 'file' && fileID) {
                            self.onContextMenu('file', fileID, e);
                        } else {
                            self.onContextMenu('folder', folderID || '', e);
                        }
                    });

                    // Long-press for mobile
                    var longPressTimer = null;
                    node.addEventListener('touchstart', function(e) {
                        var touch = e.touches[0];
                        var fakeEvent = { clientX: touch.clientX, clientY: touch.clientY, preventDefault: function() {} };
                        longPressTimer = setTimeout(function() {
                            longPressTimer = null;
                            if (nodeType === 'file' && fileID) {
                                self.onContextMenu('file', fileID, fakeEvent);
                            } else {
                                self.onContextMenu('folder', folderID || '', fakeEvent);
                            }
                        }, LONG_PRESS_MS);
                    }, { passive: true });
                    node.addEventListener('touchend', function() {
                        if (longPressTimer) { clearTimeout(longPressTimer); longPressTimer = null; }
                    });
                    node.addEventListener('touchmove', function() {
                        if (longPressTimer) { clearTimeout(longPressTimer); longPressTimer = null; }
                    });
                }

                // Drop target for bookmarks (only folders)
                if (nodeType === 'folder') {
                    node.addEventListener('dragover', function(e) {
                        e.preventDefault();
                        e.dataTransfer.dropEffect = 'move';
                        node.classList.add('folder-tree-drop-target');
                    });

                    node.addEventListener('dragleave', function() {
                        node.classList.remove('folder-tree-drop-target');
                    });

                    node.addEventListener('drop', function(e) {
                        e.preventDefault();
                        node.classList.remove('folder-tree-drop-target');
                        var itemID = e.dataTransfer.getData('text/tree-item-id');
                        var itemType = e.dataTransfer.getData('text/item-type') || 'file';
                        if (itemID) {
                            if (itemType === 'folder') {
                                // Self-drop prevention
                                if (itemID === folderID) return;
                                // Circular reference prevention
                                if (self._isDescendantOf(folderID, itemID)) return;
                                var targetDepth = self.getDepth(folderID);
                                if (targetDepth >= MAX_DEPTH - 1) return;
                            }
                            self.onDrop(itemID, folderID, itemType);
                        }
                    });

                    // Drag folder itself
                    if (self.allowDrag && folderID) {
                        node.addEventListener('dragstart', function(e) {
                            e.dataTransfer.setData('text/tree-item-id', folderID);
                            e.dataTransfer.setData('text/item-type', 'folder');
                            e.dataTransfer.effectAllowed = 'move';
                            node.classList.add('folder-tree-dragging');
                        });
                        node.addEventListener('dragend', function() {
                            node.classList.remove('folder-tree-dragging');
                        });
                    }
                }

                // Drag file items
                if (nodeType === 'file' && fileID && self.allowDrag) {
                    node.addEventListener('dragstart', function(e) {
                        e.dataTransfer.setData('text/tree-item-id', fileID);
                        e.dataTransfer.setData('text/item-type', 'file');
                        e.dataTransfer.effectAllowed = 'move';
                        node.classList.add('folder-tree-dragging');
                    });
                    node.addEventListener('dragend', function() {
                        node.classList.remove('folder-tree-dragging');
                    });
                }
            })(nodes[i]);
        }

        // Context menu on empty area of container
        if (self.onContextMenu) {
            // Remove old listener if any
            if (this._containerCtxHandler) {
                this.container.removeEventListener('contextmenu', this._containerCtxHandler);
            }
            this._containerCtxHandler = function(e) {
                // Only trigger if clicking the container itself (not a child node)
                if (e.target === self.container || e.target.closest('.folder-tree-node') === null) {
                    e.preventDefault();
                    self.onContextMenu('root', '', e);
                }
            };
            this.container.addEventListener('contextmenu', this._containerCtxHandler);
        }

        // Root drop zone on container
        if (self.allowDrag) {
            if (this._containerDropHandler) {
                this.container.removeEventListener('dragover', this._containerDropHandler);
                this.container.removeEventListener('dragleave', this._containerDragLeave);
                this.container.removeEventListener('drop', this._containerDropDrop);
            }
            this._containerDropHandler = function(e) {
                // Only trigger when dropping on the container itself (not a folder/file node)
                if (e.target === self.container || e.target.closest('.folder-tree-node') === null) {
                    e.preventDefault();
                    e.dataTransfer.dropEffect = 'move';
                    self.container.classList.add('folder-tree-root-drop-target');
                }
            };
            this._containerDragLeave = function(e) {
                if (e.target === self.container || !self.container.contains(e.relatedTarget)) {
                    self.container.classList.remove('folder-tree-root-drop-target');
                }
            };
            this._containerDropDrop = function(e) {
                e.preventDefault();
                self.container.classList.remove('folder-tree-root-drop-target');
                var itemID = e.dataTransfer.getData('text/tree-item-id');
                var itemType = e.dataTransfer.getData('text/item-type') || 'file';
                if (itemID) {
                    self.onDrop(itemID, '', itemType); // empty string = root
                }
            };
            this.container.addEventListener('dragover', this._containerDropHandler);
            this.container.addEventListener('dragleave', this._containerDragLeave);
            this.container.addEventListener('drop', this._containerDropDrop);
        }
    };

    FolderTree.prototype._isDescendantOf = function(folderID, potentialAncestorID) {
        var current = folderID;
        var seen = {};
        while (current) {
            if (seen[current]) return false; // prevent infinite loop
            seen[current] = true;
            var folder = this.getFolder(current);
            if (!folder) return false;
            if (folder.parent_id === potentialAncestorID) return true;
            current = folder.parent_id;
        }
        return false;
    };

    // Create a folder via API
    FolderTree.prototype.createFolder = function(name, parentID) {
        var self = this;
        var depth = parentID ? this.getDepth(parentID) + 1 : 1;
        if (depth > MAX_DEPTH) {
            console.error('Cannot create folder: max depth exceeded');
            return Promise.reject(new Error('Max folder depth exceeded'));
        }
        return fetch(this.apiBase, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                project_id: this.projectID,
                name: name,
                parent_id: parentID || ''
            })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function(folder) {
            if (parentID) {
                self.expanded[parentID] = true;
            }
            return self.load().then(function() { return folder; });
        });
    };

    // Rename a folder
    FolderTree.prototype.renameFolder = function(folderID, newName) {
        var self = this;
        return fetch(this.apiBase + '/' + folderID + '?project_id=' + encodeURIComponent(this.projectID), {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ name: newName })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            return self.load();
        });
    };

    // Delete a folder
    FolderTree.prototype.deleteFolder = function(folderID) {
        var self = this;
        return fetch(this.apiBase + '/' + folderID + '?project_id=' + encodeURIComponent(this.projectID), {
            method: 'DELETE'
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Delete failed');
            if (self.selectedID === folderID) {
                self.selectedID = '';
            }
            return self.load();
        });
    };

    // ─── Helpers ───

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

    // Export
    window.FolderTree = FolderTree;
})();
