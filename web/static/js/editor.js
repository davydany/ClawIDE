// ClawIDE File Editor — Multi-pane, multi-tab CodeMirror 6 integration
(function() {
    'use strict';

    // --- State ---
    var paneCounter = 0;
    var tabCounter = 0;
    var editorPanes = {};  // paneId -> { tabs, activeTabId, container, tabBarEl, editorContainer }
    var editorLayout = null; // PaneNode tree (same shape as terminal pane-layout.js)
    var focusedPaneId = null;
    var rootContainer = null;
    var projectID = null;

    // --- ID generators ---
    function nextPaneId() {
        paneCounter++;
        return 'epane-' + paneCounter;
    }

    function nextTabId() {
        tabCounter++;
        return 'etab-' + tabCounter;
    }

    // --- Layout tree helpers ---
    function makeLeaf(paneId) {
        return { type: 'leaf', paneId: paneId };
    }

    function makeSplit(direction, first, second, ratio) {
        return { type: 'split', direction: direction, ratio: ratio || 0.5, first: first, second: second };
    }

    function findParent(node, paneId) {
        if (!node || node.type !== 'split') return null;
        if (node.first && node.first.type === 'leaf' && node.first.paneId === paneId) {
            return { parent: node, key: 'first', sibling: 'second' };
        }
        if (node.second && node.second.type === 'leaf' && node.second.paneId === paneId) {
            return { parent: node, key: 'second', sibling: 'first' };
        }
        return findParent(node.first, paneId) || findParent(node.second, paneId);
    }

    function replaceInTree(root, target, replacement) {
        if (root === target) return replacement;
        if (root.type === 'split') {
            root.first = replaceInTree(root.first, target, replacement);
            root.second = replaceInTree(root.second, target, replacement);
        }
        return root;
    }

    function findLeaf(node, paneId) {
        if (!node) return null;
        if (node.type === 'leaf' && node.paneId === paneId) return node;
        if (node.type === 'split') {
            return findLeaf(node.first, paneId) || findLeaf(node.second, paneId);
        }
        return null;
    }

    function collectLeafIds(node) {
        if (!node) return [];
        if (node.type === 'leaf') return [node.paneId];
        return collectLeafIds(node.first).concat(collectLeafIds(node.second));
    }

    // --- SVG Icons ---
    var ICON_SPLIT_H = '<svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="2" width="6" height="12" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="9" y="2" width="6" height="12" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>';
    var ICON_SPLIT_V = '<svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor"><rect x="2" y="1" width="12" height="6" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="2" y="9" width="12" height="6" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>';
    var ICON_CLOSE = '&#x2715;';

    // --- Tab helpers ---
    function findTabByPath(paneId, filePath) {
        var pane = editorPanes[paneId];
        if (!pane) return null;
        for (var i = 0; i < pane.tabs.length; i++) {
            if (pane.tabs[i].filePath === filePath) return pane.tabs[i];
        }
        return null;
    }

    function findTabById(paneId, tabId) {
        var pane = editorPanes[paneId];
        if (!pane) return null;
        for (var i = 0; i < pane.tabs.length; i++) {
            if (pane.tabs[i].id === tabId) return pane.tabs[i];
        }
        return null;
    }

    function getActiveTab(paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.activeTabId) return null;
        return findTabById(paneId, pane.activeTabId);
    }

    // --- Create a new tab in a pane ---
    function createTab(paneId, filePath, content) {
        var pane = editorPanes[paneId];
        if (!pane) return null;

        var tabId = nextTabId();
        var tab = {
            id: tabId,
            filePath: filePath,
            editorView: null,
            modified: false,
            content: content || '',
            wrapperEl: null,
        };
        pane.tabs.push(tab);
        pane.activeTabId = tabId;

        // Create editor wrapper in the editor container
        if (pane.editorContainer) {
            attachEditorToTab(paneId, tabId);
        }

        renderTabs(paneId);
        return tab;
    }

    // --- Switch to a tab ---
    function switchToTab(paneId, tabId) {
        var pane = editorPanes[paneId];
        if (!pane) return;

        pane.activeTabId = tabId;

        // Toggle visibility of all tab editor wrappers
        for (var i = 0; i < pane.tabs.length; i++) {
            var tab = pane.tabs[i];
            if (tab.wrapperEl) {
                tab.wrapperEl.style.display = (tab.id === tabId) ? 'block' : 'none';
            }
        }

        renderTabs(paneId);

        // Update file tree highlight
        var activeTab = getActiveTab(paneId);
        if (activeTab) {
            highlightFileInTree(activeTab.filePath);
        }
    }

    // --- Close a tab ---
    function closeTab(paneId, tabId) {
        var pane = editorPanes[paneId];
        if (!pane) return;

        var tab = findTabById(paneId, tabId);
        if (!tab) return;

        // Confirm if modified
        if (tab.modified) {
            var filename = tab.filePath ? tab.filePath.split('/').pop() : 'Untitled';
            if (!confirm('Discard unsaved changes to "' + filename + '"?')) return;
        }

        // Destroy CM view
        if (tab.editorView) {
            window.ClawIDECodeMirror.destroyEditor(tab.editorView);
            tab.editorView = null;
        }

        // Remove wrapper from DOM
        if (tab.wrapperEl && tab.wrapperEl.parentNode) {
            tab.wrapperEl.parentNode.removeChild(tab.wrapperEl);
        }

        // Remove from tabs array
        var idx = pane.tabs.indexOf(tab);
        pane.tabs.splice(idx, 1);

        // Switch to adjacent tab or show empty state
        if (pane.tabs.length > 0) {
            // Pick the tab at the same index (or the last one)
            var newIdx = Math.min(idx, pane.tabs.length - 1);
            switchToTab(paneId, pane.tabs[newIdx].id);
        } else {
            pane.activeTabId = null;
            renderTabs(paneId);
            showEmptyPaneState(paneId);
        }
    }

    // --- Show empty state inside a pane when all tabs are closed ---
    function showEmptyPaneState(paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.editorContainer) return;

        pane.editorContainer.innerHTML = '';
        var empty = document.createElement('div');
        empty.className = 'flex items-center justify-center h-full text-gray-500';
        empty.innerHTML = '<div class="text-center">' +
            '<svg class="w-10 h-10 mx-auto mb-2 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
            '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>' +
            '</svg>' +
            '<p class="text-xs">Select a file</p>' +
            '</div>';
        pane.editorContainer.appendChild(empty);
    }

    // --- Save a specific tab ---
    function saveTab(pid, paneId, tabId) {
        var tab = findTabById(paneId, tabId);
        if (!tab || !tab.filePath || !tab.editorView) return;

        var content = window.ClawIDECodeMirror.getContent(tab.editorView);

        fetch('/projects/' + (pid || projectID) + '/api/file?path=' + encodeURIComponent(tab.filePath), {
            method: 'PUT',
            headers: { 'Content-Type': 'text/plain' },
            body: content,
        })
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to save');
                tab.modified = false;
                tab.content = content;
                renderTabs(paneId);
            })
            .catch(function(err) {
                console.error('Failed to save file:', err);
            });
    }

    function saveActiveTab(pid, paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.activeTabId) return;
        saveTab(pid, paneId, pane.activeTabId);
    }

    // --- DOM rendering ---
    function renderRoot() {
        if (!rootContainer) return;
        rootContainer.innerHTML = '';

        if (!editorLayout) {
            var empty = document.createElement('div');
            empty.id = 'editor-empty-state';
            empty.className = 'flex items-center justify-center flex-1 text-gray-500';
            empty.innerHTML = '<div class="text-center">' +
                '<svg class="w-12 h-12 mx-auto mb-3 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/>' +
                '</svg>' +
                '<p class="text-sm">Select a file to edit</p>' +
                '</div>';
            rootContainer.appendChild(empty);
            return;
        }

        buildNode(rootContainer, editorLayout);
    }

    function buildNode(parent, node) {
        if (!node) return;

        if (node.type === 'leaf') {
            var leafEl = document.createElement('div');
            leafEl.className = 'editor-pane-leaf flex flex-col flex-1 min-w-0 min-h-0';
            leafEl.dataset.paneId = node.paneId;

            // Header: tab bar + pane controls
            var header = createPaneHeader(node.paneId);
            leafEl.appendChild(header);

            // Editor body: holds all tab editor wrappers
            var editorBody = document.createElement('div');
            editorBody.className = 'editor-pane-body flex-1 min-h-0 overflow-hidden relative';
            editorBody.dataset.editorContainer = node.paneId;
            leafEl.appendChild(editorBody);

            // Focus on click
            leafEl.addEventListener('mousedown', function() {
                setFocusedPane(node.paneId);
            });

            parent.appendChild(leafEl);

            // Store references
            var pane = editorPanes[node.paneId];
            if (pane) {
                pane.container = leafEl;
                pane.tabBarEl = header.querySelector('.editor-tab-bar');
                pane.editorContainer = editorBody;
            }

            // Re-attach existing tab editors
            requestAnimationFrame(function() {
                reattachTabEditors(node.paneId);
            });
            return;
        }

        if (node.type === 'split') {
            var isHorizontal = node.direction === 'horizontal';
            var splitEl = document.createElement('div');
            splitEl.className = 'flex flex-1 min-w-0 min-h-0 ' + (isHorizontal ? 'flex-row' : 'flex-col');

            var firstEl = document.createElement('div');
            firstEl.className = 'flex min-w-0 min-h-0';
            firstEl.style.flex = '0 0 ' + ((node.ratio || 0.5) * 100) + '%';
            buildNode(firstEl, node.first);
            splitEl.appendChild(firstEl);

            var handle = document.createElement('div');
            handle.className = 'editor-resize-handle';
            handle.dataset.direction = isHorizontal ? 'horizontal' : 'vertical';
            setupResizeHandle(handle, firstEl, splitEl, node, isHorizontal);
            splitEl.appendChild(handle);

            var secondEl = document.createElement('div');
            secondEl.className = 'flex min-w-0 min-h-0 flex-1';
            buildNode(secondEl, node.second);
            splitEl.appendChild(secondEl);

            parent.appendChild(splitEl);
        }
    }

    // --- Pane header: tab bar + controls ---
    function createPaneHeader(paneId) {
        var header = document.createElement('div');
        header.className = 'editor-pane-header';

        // Tab bar (scrollable)
        var tabBar = document.createElement('div');
        tabBar.className = 'editor-tab-bar';
        header.appendChild(tabBar);

        // Controls section
        var controls = document.createElement('div');
        controls.className = 'editor-pane-controls';

        var splitHBtn = document.createElement('button');
        splitHBtn.className = 'text-gray-500 hover:text-gray-300 px-1 transition-colors';
        splitHBtn.title = 'Split horizontal';
        splitHBtn.innerHTML = ICON_SPLIT_H;
        splitHBtn.onclick = function(e) {
            e.stopPropagation();
            splitPane(paneId, 'horizontal');
        };
        controls.appendChild(splitHBtn);

        var splitVBtn = document.createElement('button');
        splitVBtn.className = 'text-gray-500 hover:text-gray-300 px-1 transition-colors';
        splitVBtn.title = 'Split vertical';
        splitVBtn.innerHTML = ICON_SPLIT_V;
        splitVBtn.onclick = function(e) {
            e.stopPropagation();
            splitPane(paneId, 'vertical');
        };
        controls.appendChild(splitVBtn);

        var closeBtn = document.createElement('button');
        closeBtn.className = 'text-gray-500 hover:text-red-400 px-1 transition-colors';
        closeBtn.title = 'Close pane';
        closeBtn.innerHTML = ICON_CLOSE;
        closeBtn.onclick = function(e) {
            e.stopPropagation();
            closePane(paneId);
        };
        controls.appendChild(closeBtn);

        header.appendChild(controls);

        return header;
    }

    // --- Render the tab bar for a pane ---
    function renderTabs(paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.tabBarEl) return;

        var tabBar = pane.tabBarEl;
        tabBar.innerHTML = '';

        for (var i = 0; i < pane.tabs.length; i++) {
            (function(tab) {
                var tabEl = document.createElement('div');
                tabEl.className = 'editor-tab' + (tab.id === pane.activeTabId ? ' active' : '');
                tabEl.dataset.tabId = tab.id;

                // Filename label
                var nameSpan = document.createElement('span');
                nameSpan.className = 'tab-name';
                nameSpan.textContent = tab.filePath ? tab.filePath.split('/').pop() : 'Untitled';
                nameSpan.title = tab.filePath || '';
                tabEl.appendChild(nameSpan);

                // Modified dot
                if (tab.modified) {
                    var dot = document.createElement('span');
                    dot.className = 'modified-dot';
                    tabEl.appendChild(dot);
                }

                // Close button
                var closeBtn = document.createElement('span');
                closeBtn.className = 'tab-close';
                closeBtn.innerHTML = '&#x2715;';
                closeBtn.title = 'Close';
                closeBtn.addEventListener('click', function(e) {
                    e.stopPropagation();
                    closeTab(paneId, tab.id);
                });
                tabEl.appendChild(closeBtn);

                // Click to switch
                tabEl.addEventListener('click', function(e) {
                    e.stopPropagation();
                    switchToTab(paneId, tab.id);
                    setFocusedPane(paneId);
                });

                tabBar.appendChild(tabEl);
            })(pane.tabs[i]);
        }

        // Scroll active tab into view
        requestAnimationFrame(function() {
            var activeEl = tabBar.querySelector('.editor-tab.active');
            if (activeEl) {
                activeEl.scrollIntoView({ inline: 'center', block: 'nearest', behavior: 'smooth' });
            }
        });
    }

    // --- Attach CodeMirror to a tab ---
    function attachEditorToTab(paneId, tabId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.editorContainer) return;

        var tab = findTabById(paneId, tabId);
        if (!tab) return;

        // Remove empty state placeholder if present
        var placeholder = pane.editorContainer.querySelector('.text-gray-500');
        if (placeholder && !placeholder.classList.contains('editor-tab-wrapper')) {
            pane.editorContainer.innerHTML = '';
        }

        // Create wrapper div for this tab's editor
        var wrapper = document.createElement('div');
        wrapper.className = 'editor-tab-wrapper absolute inset-0';
        wrapper.dataset.tabId = tabId;
        wrapper.style.display = (tab.id === pane.activeTabId) ? 'block' : 'none';
        pane.editorContainer.appendChild(wrapper);
        tab.wrapperEl = wrapper;

        // If editor already exists (from re-render), re-attach
        if (tab.editorView) {
            wrapper.appendChild(tab.editorView.dom);
            return;
        }

        // No file loaded — skip CM creation
        if (!tab.filePath) return;

        if (typeof window.ClawIDECodeMirror === 'undefined') {
            console.error('CodeMirror bundle not loaded');
            return;
        }

        tab.editorView = window.ClawIDECodeMirror.createEditor(
            wrapper,
            tab.content || '',
            tab.filePath,
            function() {
                // onDocChange
                if (!tab.modified) {
                    tab.modified = true;
                    renderTabs(paneId);
                }
            },
            function() {
                // onSave (Cmd+S from within CM)
                saveTab(projectID, paneId, tab.id);
            }
        );
    }

    // --- Re-attach all tab editors after a DOM re-render ---
    function reattachTabEditors(paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.editorContainer) return;

        if (pane.tabs.length === 0) {
            showEmptyPaneState(paneId);
            return;
        }

        for (var i = 0; i < pane.tabs.length; i++) {
            var tab = pane.tabs[i];

            // Create wrapper
            var wrapper = document.createElement('div');
            wrapper.className = 'editor-tab-wrapper absolute inset-0';
            wrapper.dataset.tabId = tab.id;
            wrapper.style.display = (tab.id === pane.activeTabId) ? 'block' : 'none';
            pane.editorContainer.appendChild(wrapper);
            tab.wrapperEl = wrapper;

            if (tab.editorView) {
                // Re-attach existing view
                wrapper.appendChild(tab.editorView.dom);
            } else if (tab.filePath) {
                // Create editor
                (function(t, w, pid) {
                    t.editorView = window.ClawIDECodeMirror.createEditor(
                        w,
                        t.content || '',
                        t.filePath,
                        function() {
                            if (!t.modified) {
                                t.modified = true;
                                renderTabs(pid);
                            }
                        },
                        function() {
                            saveTab(projectID, pid, t.id);
                        }
                    );
                })(tab, wrapper, paneId);
            }
        }

        renderTabs(paneId);
    }

    // --- Resize handle ---
    function setupResizeHandle(handle, firstEl, splitEl, node, isHorizontal) {
        handle.addEventListener('mousedown', function(e) {
            e.preventDefault();
            var startPos = isHorizontal ? e.clientX : e.clientY;
            var startSize = isHorizontal ? firstEl.offsetWidth : firstEl.offsetHeight;
            var totalSize = isHorizontal ? splitEl.offsetWidth : splitEl.offsetHeight;

            document.body.style.cursor = isHorizontal ? 'col-resize' : 'row-resize';
            document.body.style.userSelect = 'none';

            function onMouseMove(e) {
                var currentPos = isHorizontal ? e.clientX : e.clientY;
                var delta = currentPos - startPos;
                var newSize = startSize + delta;
                var handleSize = isHorizontal ? handle.offsetWidth : handle.offsetHeight;
                var ratio = newSize / (totalSize - handleSize);
                ratio = Math.max(0.1, Math.min(0.9, ratio));
                firstEl.style.flex = '0 0 ' + (ratio * 100) + '%';
            }

            function onMouseUp(e) {
                document.body.style.cursor = '';
                document.body.style.userSelect = '';
                document.removeEventListener('mousemove', onMouseMove);
                document.removeEventListener('mouseup', onMouseUp);

                var currentPos = isHorizontal ? e.clientX : e.clientY;
                var delta = currentPos - startPos;
                var newSize = startSize + delta;
                var handleSize = isHorizontal ? handle.offsetWidth : handle.offsetHeight;
                var ratio = newSize / (totalSize - handleSize);
                node.ratio = Math.max(0.1, Math.min(0.9, ratio));
            }

            document.addEventListener('mousemove', onMouseMove);
            document.addEventListener('mouseup', onMouseUp);
        });
    }

    // --- Focused pane ---
    function setFocusedPane(paneId) {
        focusedPaneId = paneId;

        document.querySelectorAll('.editor-pane-leaf').forEach(function(el) {
            var header = el.querySelector('.editor-pane-header');
            if (header) {
                if (el.dataset.paneId === paneId) {
                    header.classList.add('focused');
                } else {
                    header.classList.remove('focused');
                }
            }
        });
    }

    function getFocusedPaneId() {
        if (!focusedPaneId || !editorPanes[focusedPaneId]) {
            var ids = Object.keys(editorPanes);
            focusedPaneId = ids.length > 0 ? ids[0] : null;
        }
        return focusedPaneId;
    }

    // --- Highlight active file in tree ---
    function highlightFileInTree(filePath) {
        document.querySelectorAll('.file-tree-item.active').forEach(function(el) {
            el.classList.remove('active');
        });
        if (filePath) {
            document.querySelectorAll('.file-tree-item').forEach(function(el) {
                if (el.getAttribute('data-filepath') === filePath) {
                    el.classList.add('active');
                }
            });
        }
    }

    // --- Public API ---

    function loadFile(pid, filePath) {
        projectID = pid;

        if (!rootContainer) {
            rootContainer = document.getElementById('editor-pane-root');
            if (rootContainer && rootContainer.dataset.projectId) {
                projectID = rootContainer.dataset.projectId;
            }
        }

        highlightFileInTree(filePath);

        var targetPaneId = getFocusedPaneId();

        // If no panes exist, create the first one
        if (!targetPaneId) {
            targetPaneId = nextPaneId();
            editorPanes[targetPaneId] = {
                tabs: [],
                activeTabId: null,
                container: null,
                tabBarEl: null,
                editorContainer: null,
            };
            editorLayout = makeLeaf(targetPaneId);
            renderRoot();
        }

        var pane = editorPanes[targetPaneId];

        // 1. Check if filePath already open in a tab → switch to it
        var existingTab = findTabByPath(targetPaneId, filePath);
        if (existingTab) {
            switchToTab(targetPaneId, existingTab.id);
            setFocusedPane(targetPaneId);
            return;
        }

        // Fetch file content first, then decide how to open it
        fetch('/projects/' + projectID + '/api/file?path=' + encodeURIComponent(filePath))
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to load file (HTTP ' + resp.status + ')');
                return resp.text();
            })
            .then(function(content) {
                var activeTab = getActiveTab(targetPaneId);

                // 2. If active tab is not modified and has a file → reuse it
                if (activeTab && !activeTab.modified && activeTab.filePath) {
                    reuseTab(targetPaneId, activeTab, filePath, content);
                }
                // 3. If active tab has no file (fresh pane) → reuse it
                else if (activeTab && !activeTab.filePath) {
                    reuseTab(targetPaneId, activeTab, filePath, content);
                }
                // 4. Otherwise → create new tab
                else {
                    openNewTab(targetPaneId, filePath, content);
                }
            })
            .catch(function(err) {
                console.error('Failed to load file:', err);
            });

        setFocusedPane(targetPaneId);
    }

    function reuseTab(paneId, tab, filePath, content) {
        tab.filePath = filePath;
        tab.content = content;
        tab.modified = false;

        if (tab.editorView) {
            window.ClawIDECodeMirror.setContent(tab.editorView, content, filePath);
        } else if (tab.wrapperEl) {
            // Need to create editor in existing wrapper
            tab.wrapperEl.innerHTML = '';
            tab.editorView = window.ClawIDECodeMirror.createEditor(
                tab.wrapperEl,
                content,
                filePath,
                function() {
                    if (!tab.modified) {
                        tab.modified = true;
                        renderTabs(paneId);
                    }
                },
                function() {
                    saveTab(projectID, paneId, tab.id);
                }
            );
        }

        renderTabs(paneId);
    }

    function openNewTab(paneId, filePath, content) {
        var pane = editorPanes[paneId];
        if (!pane) return;

        // If this is the first tab and there's an empty state, clear it
        if (pane.tabs.length === 0 && pane.editorContainer) {
            pane.editorContainer.innerHTML = '';
        }

        var tab = createTab(paneId, filePath, content);
        if (!tab) return;

        // If editor wasn't created in createTab (container might not exist yet),
        // it will be handled by reattachTabEditors
    }

    function splitPane(paneId, direction) {
        var leaf = findLeaf(editorLayout, paneId);
        if (!leaf) return;

        var pane = editorPanes[paneId];
        var newPaneId = nextPaneId();
        var activeTab = getActiveTab(paneId);

        // New pane gets a single tab with the active file
        editorPanes[newPaneId] = {
            tabs: [],
            activeTabId: null,
            container: null,
            tabBarEl: null,
            editorContainer: null,
        };

        var newSplit = makeSplit(direction, makeLeaf(paneId), makeLeaf(newPaneId));

        if (editorLayout === leaf) {
            editorLayout = newSplit;
        } else {
            replaceInTree(editorLayout, leaf, newSplit);
        }

        // Detach all editors before re-render
        detachAllEditors();
        renderRoot();

        // Create a tab in the new pane with the same file
        if (activeTab && activeTab.filePath) {
            openNewTab(newPaneId, activeTab.filePath, activeTab.content);
        }

        setFocusedPane(newPaneId);
    }

    function closePane(paneId) {
        var pane = editorPanes[paneId];
        if (!pane) return;

        // Check for unsaved tabs
        var unsaved = pane.tabs.filter(function(t) { return t.modified; });
        if (unsaved.length > 0) {
            var names = unsaved.map(function(t) { return t.filePath ? t.filePath.split('/').pop() : 'Untitled'; }).join(', ');
            if (!confirm('Discard unsaved changes in: ' + names + '?')) return;
        }

        // Destroy all CM editors in this pane
        for (var i = 0; i < pane.tabs.length; i++) {
            if (pane.tabs[i].editorView) {
                window.ClawIDECodeMirror.destroyEditor(pane.tabs[i].editorView);
                pane.tabs[i].editorView = null;
            }
        }
        delete editorPanes[paneId];

        // Update layout tree
        if (editorLayout && editorLayout.type === 'leaf' && editorLayout.paneId === paneId) {
            editorLayout = null;
            focusedPaneId = null;
            renderRoot();
            return;
        }

        var parentInfo = findParent(editorLayout, paneId);
        if (parentInfo) {
            var siblingNode = parentInfo.parent[parentInfo.sibling];
            if (editorLayout === parentInfo.parent) {
                editorLayout = siblingNode;
            } else {
                replaceInTree(editorLayout, parentInfo.parent, siblingNode);
            }
        }

        detachAllEditors();
        renderRoot();

        var remainingIds = collectLeafIds(editorLayout);
        if (remainingIds.length > 0) {
            setFocusedPane(remainingIds[0]);
        } else {
            focusedPaneId = null;
        }
    }

    function detachAllEditors() {
        Object.keys(editorPanes).forEach(function(id) {
            var pane = editorPanes[id];
            if (!pane) return;
            for (var i = 0; i < pane.tabs.length; i++) {
                var tab = pane.tabs[i];
                if (tab.editorView && tab.editorView.dom.parentNode) {
                    tab.editorView.dom.parentNode.removeChild(tab.editorView.dom);
                }
                tab.wrapperEl = null;
            }
        });
    }

    // --- Global keyboard shortcuts ---
    document.addEventListener('keydown', function(e) {
        var editorRoot = document.getElementById('editor-pane-root');
        if (!editorRoot || editorRoot.offsetParent === null) return;

        // Cmd+S / Ctrl+S: save active tab
        if ((e.metaKey || e.ctrlKey) && e.key === 's') {
            var pid = getFocusedPaneId();
            if (pid) {
                e.preventDefault();
                saveActiveTab(projectID, pid);
            }
        }

        // Cmd+W / Ctrl+W: close active tab
        if ((e.metaKey || e.ctrlKey) && e.key === 'w') {
            var pid2 = getFocusedPaneId();
            if (pid2) {
                var pane = editorPanes[pid2];
                if (pane && pane.activeTabId) {
                    e.preventDefault();
                    closeTab(pid2, pane.activeTabId);
                }
            }
        }
    });

    // --- Expose to global scope ---
    window.ClawIDEEditor = {
        loadFile: loadFile,
        saveFile: function(pid) {
            var pid2 = getFocusedPaneId();
            if (pid2) saveActiveTab(pid, pid2);
        },
        saveTab: saveTab,
        closeTab: closeTab,
        savePane: saveActiveTab,
        splitPane: splitPane,
        closePane: closePane,
        getFocusedPaneId: getFocusedPaneId,
        isModified: function() {
            return Object.keys(editorPanes).some(function(id) {
                var pane = editorPanes[id];
                return pane.tabs.some(function(t) { return t.modified; });
            });
        },
        getCurrentFile: function() {
            var pid = getFocusedPaneId();
            if (!pid) return null;
            var tab = getActiveTab(pid);
            return tab ? tab.filePath : null;
        },
    };
})();
