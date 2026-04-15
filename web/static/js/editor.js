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
    var ICON_PREVIEW = '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>';
    var ICON_PREVIEW_SIDE = '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2"/><line x1="12" y1="3" x2="12" y2="21"/></svg>';

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
            previewMode: 'off',
            previewEl: null,
            previewTimer: null,
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
                if (tab.id === tabId) {
                    // Restore display based on preview mode
                    if (tab.previewMode === 'side') {
                        tab.wrapperEl.style.display = 'flex';
                    } else {
                        tab.wrapperEl.style.display = 'block';
                    }
                } else {
                    tab.wrapperEl.style.display = 'none';
                }
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

        // Clear preview timer
        if (tab.previewTimer) {
            clearTimeout(tab.previewTimer);
            tab.previewTimer = null;
        }

        // Tear down preview-mode extras so scroll listeners don't leak.
        teardownScrollSync(tab);
        destroyTocPanel(tab);

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
        empty.className = 'flex items-center justify-center h-full text-th-text-faint';
        empty.innerHTML = '<div class="text-center">' +
            '<svg class="w-10 h-10 mx-auto mb-2 text-th-text-ghost" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
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

        var saveURL = tab.saveURL || ('/projects/' + (pid || projectID) + '/api/file?path=' + encodeURIComponent(tab.filePath));
        fetch(saveURL, {
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
            empty.className = 'flex items-center justify-center flex-1 text-th-text-faint';
            empty.innerHTML = '<div class="text-center">' +
                '<svg class="w-12 h-12 mx-auto mb-3 text-th-text-ghost" fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
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

        // Markdown preview buttons (hidden by default, shown for .md files)
        var previewGroup = document.createElement('div');
        previewGroup.className = 'editor-preview-group hidden';

        var previewSideBtn = document.createElement('button');
        previewSideBtn.className = 'editor-preview-btn editor-control-btn text-th-text-faint hover:text-th-text-tertiary px-1 transition-colors';
        previewSideBtn.dataset.tooltip = 'Side-by-side Preview';
        previewSideBtn.innerHTML = ICON_PREVIEW_SIDE;
        previewSideBtn.onclick = function(e) {
            e.stopPropagation();
            var tab = getActiveTab(paneId);
            if (!tab) return;
            var newMode = (tab.previewMode === 'side') ? 'off' : 'side';
            setPreviewMode(paneId, tab.id, newMode);
        };
        previewGroup.appendChild(previewSideBtn);

        var previewOnlyBtn = document.createElement('button');
        previewOnlyBtn.className = 'editor-preview-btn editor-control-btn text-th-text-faint hover:text-th-text-tertiary px-1 transition-colors';
        previewOnlyBtn.dataset.tooltip = 'Full Preview';
        previewOnlyBtn.innerHTML = ICON_PREVIEW;
        previewOnlyBtn.onclick = function(e) {
            e.stopPropagation();
            var tab = getActiveTab(paneId);
            if (!tab) return;
            var newMode = (tab.previewMode === 'preview') ? 'off' : 'preview';
            setPreviewMode(paneId, tab.id, newMode);
        };
        previewGroup.appendChild(previewOnlyBtn);

        controls.appendChild(previewGroup);

        // Separator between preview and split controls
        var separator = document.createElement('div');
        separator.className = 'editor-controls-separator hidden';
        controls.appendChild(separator);

        // Store references so updatePreviewButtons can find them
        header.dataset.paneId = paneId;
        header._previewSideBtn = previewSideBtn;
        header._previewOnlyBtn = previewOnlyBtn;
        header._previewGroup = previewGroup;
        header._previewSeparator = separator;

        var splitHBtn = document.createElement('button');
        splitHBtn.className = 'editor-control-btn text-th-text-faint hover:text-th-text-tertiary px-1 transition-colors';
        splitHBtn.dataset.tooltip = 'Split Horizontal';
        splitHBtn.innerHTML = ICON_SPLIT_H;
        splitHBtn.onclick = function(e) {
            e.stopPropagation();
            splitPane(paneId, 'horizontal');
        };
        controls.appendChild(splitHBtn);

        var splitVBtn = document.createElement('button');
        splitVBtn.className = 'editor-control-btn text-th-text-faint hover:text-th-text-tertiary px-1 transition-colors';
        splitVBtn.dataset.tooltip = 'Split Vertical';
        splitVBtn.innerHTML = ICON_SPLIT_V;
        splitVBtn.onclick = function(e) {
            e.stopPropagation();
            splitPane(paneId, 'vertical');
        };
        controls.appendChild(splitVBtn);

        var closeBtn = document.createElement('button');
        closeBtn.className = 'editor-control-btn text-th-text-faint hover:text-red-400 px-1 transition-colors';
        closeBtn.dataset.tooltip = 'Close Pane';
        closeBtn.innerHTML = ICON_CLOSE;
        closeBtn.onclick = function(e) {
            e.stopPropagation();
            closePane(paneId);
        };
        controls.appendChild(closeBtn);

        header.appendChild(controls);

        return header;
    }

    // --- Floating tooltip for editor control buttons ---
    // (Uses body-appended element to escape overflow:hidden ancestors)
    var floatingTooltip = null;
    function showTooltip(btn) {
        var text = btn.dataset.tooltip;
        if (!text) return;
        if (!floatingTooltip) {
            floatingTooltip = document.createElement('div');
            floatingTooltip.className = 'editor-floating-tooltip';
            document.body.appendChild(floatingTooltip);
        }
        floatingTooltip.textContent = text;
        floatingTooltip.style.opacity = '0';
        floatingTooltip.style.display = 'block';
        var rect = btn.getBoundingClientRect();
        var ttRect = floatingTooltip.getBoundingClientRect();
        var left = rect.left + rect.width / 2 - ttRect.width / 2;
        var top = rect.bottom + 6;
        // Keep within viewport
        if (left < 4) left = 4;
        if (left + ttRect.width > window.innerWidth - 4) left = window.innerWidth - 4 - ttRect.width;
        floatingTooltip.style.left = left + 'px';
        floatingTooltip.style.top = top + 'px';
        floatingTooltip.style.opacity = '1';
    }
    function hideTooltip() {
        if (floatingTooltip) {
            floatingTooltip.style.opacity = '0';
            floatingTooltip.style.display = 'none';
        }
    }
    // Attach tooltip listeners via event delegation on the editor container
    document.addEventListener('mouseover', function(e) {
        var btn = e.target.closest('.editor-control-btn');
        if (btn && btn.dataset.tooltip) showTooltip(btn);
    });
    document.addEventListener('mouseout', function(e) {
        var btn = e.target.closest('.editor-control-btn');
        if (btn) hideTooltip();
    });

    // --- Markdown preview helpers ---
    function isMarkdownFile(filePath) {
        return filePath && /\.md$/i.test(filePath);
    }

    function updatePreviewButtons(paneId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.container) return;

        var header = pane.container.querySelector('.editor-pane-header');
        if (!header || !header._previewSideBtn) return;

        var tab = getActiveTab(paneId);
        var isMd = tab && isMarkdownFile(tab.filePath);

        // Show/hide the preview group and separator together
        if (header._previewGroup) {
            header._previewGroup.classList.toggle('hidden', !isMd);
        }
        if (header._previewSeparator) {
            header._previewSeparator.classList.toggle('hidden', !isMd);
        }

        if (isMd && tab) {
            header._previewSideBtn.classList.toggle('active', tab.previewMode === 'side');
            header._previewOnlyBtn.classList.toggle('active', tab.previewMode === 'preview');
        } else {
            header._previewSideBtn.classList.remove('active');
            header._previewOnlyBtn.classList.remove('active');
        }
    }

    function ensurePreviewContainer(tab) {
        if (tab.previewEl) return tab.previewEl;
        if (!tab.wrapperEl) return null;

        var container = document.createElement('div');
        container.className = 'md-preview-container note-markdown-preview text-sm text-th-text-tertiary';
        tab.wrapperEl.appendChild(container);
        tab.previewEl = container;
        return container;
    }

    function ensureCmWrap(tab) {
        if (!tab.wrapperEl || !tab.editorView) return null;
        var existing = tab.wrapperEl.querySelector('.cm-editor-wrap');
        if (existing) return existing;

        // Wrap the CodeMirror DOM in a .cm-editor-wrap div
        var wrap = document.createElement('div');
        wrap.className = 'cm-editor-wrap';
        // Move the CM editor dom into the wrap
        if (tab.editorView.dom.parentNode === tab.wrapperEl) {
            tab.wrapperEl.insertBefore(wrap, tab.editorView.dom);
            wrap.appendChild(tab.editorView.dom);
        }
        return wrap;
    }

    function setPreviewMode(paneId, tabId, mode) {
        var tab = findTabById(paneId, tabId);
        if (!tab || !tab.wrapperEl) return;

        tab.previewMode = mode;
        var wrapper = tab.wrapperEl;

        // Remove all mode classes
        wrapper.classList.remove('preview-side', 'preview-only');

        // Always tear down mode-specific extras before reconfiguring.
        teardownScrollSync(tab);
        destroyTocPanel(tab);

        if (mode === 'off') {
            // Remove preview container if it exists
            if (tab.previewEl && tab.previewEl.parentNode) {
                tab.previewEl.parentNode.removeChild(tab.previewEl);
                tab.previewEl = null;
            }
            // Remove resize handle
            var handle = wrapper.querySelector('.editor-preview-resize-handle');
            if (handle) handle.parentNode.removeChild(handle);
            // Ensure cm-wrap is removed (flatten back)
            var cmWrap = wrapper.querySelector('.cm-editor-wrap');
            if (cmWrap && tab.editorView) {
                wrapper.insertBefore(tab.editorView.dom, cmWrap);
                wrapper.removeChild(cmWrap);
            }
            // Show editor
            if (tab.editorView) {
                tab.editorView.dom.style.display = '';
            }
        } else if (mode === 'side') {
            wrapper.classList.add('preview-side');
            ensureCmWrap(tab);
            // Show editor
            if (tab.editorView) {
                tab.editorView.dom.style.display = '';
            }
            // Add resize handle if not present
            if (!wrapper.querySelector('.editor-preview-resize-handle')) {
                var resizeHandle = document.createElement('div');
                resizeHandle.className = 'editor-preview-resize-handle';
                var cmWrap2 = wrapper.querySelector('.cm-editor-wrap');
                if (cmWrap2) {
                    wrapper.insertBefore(resizeHandle, cmWrap2.nextSibling);
                }
                setupPreviewResizeHandle(resizeHandle, wrapper);
            }
            ensurePreviewContainer(tab);
            renderPreview(paneId, tabId);
            setupScrollSync(tab);
        } else if (mode === 'preview') {
            wrapper.classList.add('preview-only');
            ensureCmWrap(tab);
            // Hide editor
            var cmWrap3 = wrapper.querySelector('.cm-editor-wrap');
            if (cmWrap3) {
                cmWrap3.style.display = 'none';
            }
            // Remove resize handle
            var handle2 = wrapper.querySelector('.editor-preview-resize-handle');
            if (handle2) handle2.parentNode.removeChild(handle2);
            ensurePreviewContainer(tab);
            buildTocPanel(tab);
            renderPreview(paneId, tabId);
        }

        updatePreviewButtons(paneId);
        renderTabs(paneId);
    }

    function renderPreview(paneId, tabId) {
        var tab = findTabById(paneId, tabId);
        if (!tab || !tab.previewEl) return;
        if (!tab.editorView) return;

        var content = window.ClawIDECodeMirror.getContent(tab.editorView);
        if (window.ClawIDEMarkdown) {
            var result = window.ClawIDEMarkdown.renderInto(tab.previewEl, content);
            tab.previewHeadings = (result && result.headings) || [];
        }

        // Refresh TOC contents in preview-only mode
        if (tab.previewMode === 'preview' && tab.tocPanel) {
            renderTocContents(tab);
            tab.tocPanel.style.display = tab.previewHeadings.length >= 2 ? '' : 'none';
            updateTocActiveHeading(tab);
        }
        // Side-mode sync reads tab.previewHeadings lazily; no rebuild needed here.
    }

    function debouncedRenderPreview(paneId, tabId) {
        var tab = findTabById(paneId, tabId);
        if (!tab) return;

        if (tab.previewTimer) clearTimeout(tab.previewTimer);
        tab.previewTimer = setTimeout(function() {
            tab.previewTimer = null;
            renderPreview(paneId, tabId);
        }, 150);
    }

    function setupPreviewResizeHandle(handle, wrapper) {
        handle.addEventListener('mousedown', function(e) {
            e.preventDefault();
            var cmWrap = wrapper.querySelector('.cm-editor-wrap');
            if (!cmWrap) return;

            var startX = e.clientX;
            var startWidth = cmWrap.offsetWidth;
            var totalWidth = wrapper.offsetWidth;

            document.body.style.cursor = 'col-resize';
            document.body.style.userSelect = 'none';

            function onMouseMove(e) {
                var delta = e.clientX - startX;
                var newWidth = startWidth + delta;
                var ratio = newWidth / totalWidth;
                ratio = Math.max(0.15, Math.min(0.85, ratio));
                cmWrap.style.flex = '0 0 ' + (ratio * 100) + '%';
            }

            function onMouseUp() {
                document.body.style.cursor = '';
                document.body.style.userSelect = '';
                document.removeEventListener('mousemove', onMouseMove);
                document.removeEventListener('mouseup', onMouseUp);
            }

            document.addEventListener('mousemove', onMouseMove);
            document.addEventListener('mouseup', onMouseUp);
        });
    }

    // ===================================================================
    // Table of Contents (full-preview mode only)
    // ===================================================================

    function buildTocPanel(tab) {
        if (!tab || !tab.wrapperEl || tab.tocPanel) return;

        var panel = document.createElement('aside');
        panel.className = 'md-preview-toc-panel';

        var title = document.createElement('div');
        title.className = 'md-toc-title';
        title.textContent = 'On this page';
        panel.appendChild(title);

        var nav = document.createElement('nav');
        var ul = document.createElement('ul');
        nav.appendChild(ul);
        panel.appendChild(nav);

        // Insert TOC to the left of the preview container
        if (tab.previewEl && tab.previewEl.parentNode === tab.wrapperEl) {
            tab.wrapperEl.insertBefore(panel, tab.previewEl);
        } else {
            tab.wrapperEl.appendChild(panel);
        }

        tab.tocPanel = panel;
        tab.tocListEl = ul;

        // Delegated click handler: resolve data-heading-id → scroll the heading.
        panel.addEventListener('click', function(e) {
            var a = e.target.closest ? e.target.closest('a[data-heading-id]') : null;
            if (!a) return;
            e.preventDefault();
            var id = a.getAttribute('data-heading-id');
            var target = tab.previewEl && tab.previewEl.querySelector('#' + cssEscapeId(id));
            if (target) {
                target.scrollIntoView({ behavior: 'smooth', block: 'start' });
            }
        });

        // Scroll listener on preview to keep active heading in sync.
        if (tab.previewEl) {
            tab.tocScrollHandler = function() {
                if (tab.tocScrollPending) return;
                tab.tocScrollPending = true;
                requestAnimationFrame(function() {
                    tab.tocScrollPending = false;
                    updateTocActiveHeading(tab);
                });
            };
            tab.previewEl.addEventListener('scroll', tab.tocScrollHandler, { passive: true });
        }
    }

    function renderTocContents(tab) {
        if (!tab || !tab.tocListEl) return;
        var headings = tab.previewHeadings || [];
        tab.tocListEl.innerHTML = '';
        for (var i = 0; i < headings.length; i++) {
            var h = headings[i];
            var li = document.createElement('li');
            var indent = Math.max(0, (h.level - 1)) * 12;
            li.style.setProperty('--toc-indent', indent + 'px');
            var a = document.createElement('a');
            a.setAttribute('data-heading-id', h.id);
            a.setAttribute('href', '#' + h.id);
            a.textContent = h.text;
            li.appendChild(a);
            tab.tocListEl.appendChild(li);
        }
    }

    function updateTocActiveHeading(tab) {
        if (!tab || !tab.tocListEl || !tab.previewEl) return;
        var headings = tab.previewHeadings || [];
        if (!headings.length) return;

        var scrollTop = tab.previewEl.scrollTop + 80;
        var activeIdx = 0;
        for (var i = 0; i < headings.length; i++) {
            if (headings[i].element && headings[i].element.offsetTop <= scrollTop) {
                activeIdx = i;
            } else {
                break;
            }
        }

        var links = tab.tocListEl.querySelectorAll('a[data-heading-id]');
        for (var j = 0; j < links.length; j++) {
            if (j === activeIdx) links[j].classList.add('active');
            else links[j].classList.remove('active');
        }
    }

    function destroyTocPanel(tab) {
        if (!tab) return;
        if (tab.tocScrollHandler && tab.previewEl) {
            tab.previewEl.removeEventListener('scroll', tab.tocScrollHandler);
        }
        tab.tocScrollHandler = null;
        tab.tocScrollPending = false;
        if (tab.tocPanel && tab.tocPanel.parentNode) {
            tab.tocPanel.parentNode.removeChild(tab.tocPanel);
        }
        tab.tocPanel = null;
        tab.tocListEl = null;
    }

    // CSS.escape fallback for heading IDs that may contain special characters.
    function cssEscapeId(id) {
        if (window.CSS && typeof window.CSS.escape === 'function') {
            return window.CSS.escape(id);
        }
        return String(id).replace(/[^\w-]/g, '\\$&');
    }

    // ===================================================================
    // Scroll sync (side-by-side mode only)
    // ===================================================================

    function setupScrollSync(tab) {
        if (!tab || !tab.editorView || !tab.previewEl) return;
        if (tab.scrollSync) teardownScrollSync(tab);

        var cm = window.ClawIDECodeMirror;
        if (!cm || !cm.getScrollDOM) return;
        var scrollDOM = cm.getScrollDOM(tab.editorView);
        if (!scrollDOM) return;

        var state = {
            scrollDOM: scrollDOM,
            lastProgrammatic: 0,
            editorPending: false,
            previewPending: false,
            editorHandler: null,
            previewHandler: null,
        };

        state.editorHandler = function() {
            if (performance.now() - state.lastProgrammatic < 100) return;
            if (state.editorPending) return;
            state.editorPending = true;
            requestAnimationFrame(function() {
                state.editorPending = false;
                syncPreviewFromEditor(tab);
            });
        };
        state.previewHandler = function() {
            if (performance.now() - state.lastProgrammatic < 100) return;
            if (state.previewPending) return;
            state.previewPending = true;
            requestAnimationFrame(function() {
                state.previewPending = false;
                syncEditorFromPreview(tab);
            });
        };

        scrollDOM.addEventListener('scroll', state.editorHandler, { passive: true });
        tab.previewEl.addEventListener('scroll', state.previewHandler, { passive: true });

        tab.scrollSync = state;
    }

    function teardownScrollSync(tab) {
        if (!tab || !tab.scrollSync) return;
        var s = tab.scrollSync;
        if (s.scrollDOM && s.editorHandler) {
            s.scrollDOM.removeEventListener('scroll', s.editorHandler);
        }
        if (tab.previewEl && s.previewHandler) {
            tab.previewEl.removeEventListener('scroll', s.previewHandler);
        }
        tab.scrollSync = null;
    }

    // Find the heading index i such that headings[i].sourceLine <= topLine < headings[i+1].sourceLine
    function findHeadingIndexByLine(headings, topLine) {
        if (!headings.length) return -1;
        if (topLine < headings[0].sourceLine) return -1;
        for (var i = 0; i < headings.length - 1; i++) {
            if (headings[i + 1].sourceLine > topLine) return i;
        }
        return headings.length - 1;
    }

    function findHeadingIndexByOffset(headings, scrollTop) {
        if (!headings.length) return -1;
        var firstTop = headings[0].element ? headings[0].element.offsetTop : 0;
        if (scrollTop < firstTop) return -1;
        for (var i = 0; i < headings.length - 1; i++) {
            var nextTop = headings[i + 1].element ? headings[i + 1].element.offsetTop : 0;
            if (nextTop > scrollTop) return i;
        }
        return headings.length - 1;
    }

    function proportionalSyncPreviewFromEditor(tab, state) {
        var src = state.scrollDOM;
        var srcMax = src.scrollHeight - src.clientHeight;
        var dstMax = tab.previewEl.scrollHeight - tab.previewEl.clientHeight;
        if (srcMax <= 0 || dstMax <= 0) return;
        var ratio = src.scrollTop / srcMax;
        state.lastProgrammatic = performance.now();
        tab.previewEl.scrollTop = ratio * dstMax;
    }

    function proportionalSyncEditorFromPreview(tab, state) {
        var src = tab.previewEl;
        var srcMax = src.scrollHeight - src.clientHeight;
        var dstMax = state.scrollDOM.scrollHeight - state.scrollDOM.clientHeight;
        if (srcMax <= 0 || dstMax <= 0) return;
        var ratio = src.scrollTop / srcMax;
        state.lastProgrammatic = performance.now();
        state.scrollDOM.scrollTop = ratio * dstMax;
    }

    function headingsUsable(headings) {
        if (!headings || !headings.length) return false;
        for (var i = 0; i < headings.length; i++) {
            if (headings[i].sourceLine == null || !headings[i].element) return false;
        }
        return true;
    }

    function syncPreviewFromEditor(tab) {
        if (!tab || !tab.scrollSync || !tab.previewEl || !tab.editorView) return;
        var state = tab.scrollSync;
        var headings = tab.previewHeadings || [];
        var cm = window.ClawIDECodeMirror;

        if (!headingsUsable(headings) || !cm || !cm.getTopVisibleLine) {
            proportionalSyncPreviewFromEditor(tab, state);
            return;
        }

        var topLine = cm.getTopVisibleLine(tab.editorView);
        var idx = findHeadingIndexByLine(headings, topLine);
        var previewMax = tab.previewEl.scrollHeight - tab.previewEl.clientHeight;
        if (previewMax <= 0) return;

        var targetY;
        if (idx < 0) {
            // Above the first heading → proportional from top of doc.
            var firstTop = headings[0].element.offsetTop;
            var firstLine = headings[0].sourceLine;
            var frac = firstLine > 1 ? (topLine - 1) / (firstLine - 1) : 0;
            targetY = frac * firstTop;
        } else if (idx >= headings.length - 1) {
            // Below the last heading → proportional from last heading to end of doc.
            var lastEl = headings[idx].element;
            var lastTop = lastEl.offsetTop;
            var docLines = tab.editorView.state.doc.lines;
            var span = Math.max(1, docLines - headings[idx].sourceLine);
            var fracL = Math.min(1, Math.max(0, (topLine - headings[idx].sourceLine) / span));
            targetY = lastTop + fracL * (previewMax - lastTop);
        } else {
            var aEl = headings[idx].element;
            var bEl = headings[idx + 1].element;
            var aLine = headings[idx].sourceLine;
            var bLine = headings[idx + 1].sourceLine;
            var lineSpan = Math.max(1, bLine - aLine);
            var fracM = Math.min(1, Math.max(0, (topLine - aLine) / lineSpan));
            targetY = aEl.offsetTop + fracM * (bEl.offsetTop - aEl.offsetTop);
        }

        state.lastProgrammatic = performance.now();
        tab.previewEl.scrollTop = Math.max(0, Math.min(previewMax, targetY));
    }

    function syncEditorFromPreview(tab) {
        if (!tab || !tab.scrollSync || !tab.previewEl || !tab.editorView) return;
        var state = tab.scrollSync;
        var headings = tab.previewHeadings || [];
        var cm = window.ClawIDECodeMirror;

        if (!headingsUsable(headings) || !cm || !cm.scrollToLine) {
            proportionalSyncEditorFromPreview(tab, state);
            return;
        }

        var scrollTop = tab.previewEl.scrollTop;
        var idx = findHeadingIndexByOffset(headings, scrollTop);
        var docLines = tab.editorView.state.doc.lines;
        var previewMax = tab.previewEl.scrollHeight - tab.previewEl.clientHeight;

        var targetLine;
        if (idx < 0) {
            var firstTop2 = headings[0].element.offsetTop;
            var firstLine2 = headings[0].sourceLine;
            var frac0 = firstTop2 > 0 ? scrollTop / firstTop2 : 0;
            targetLine = 1 + frac0 * (firstLine2 - 1);
        } else if (idx >= headings.length - 1) {
            var lastTop2 = headings[idx].element.offsetTop;
            var span2 = Math.max(1, previewMax - lastTop2);
            var fracEnd = Math.min(1, Math.max(0, (scrollTop - lastTop2) / span2));
            targetLine = headings[idx].sourceLine + fracEnd * (docLines - headings[idx].sourceLine);
        } else {
            var aTop = headings[idx].element.offsetTop;
            var bTop = headings[idx + 1].element.offsetTop;
            var pxSpan = Math.max(1, bTop - aTop);
            var fracMid = Math.min(1, Math.max(0, (scrollTop - aTop) / pxSpan));
            targetLine = headings[idx].sourceLine + fracMid * (headings[idx + 1].sourceLine - headings[idx].sourceLine);
        }

        state.lastProgrammatic = performance.now();
        cm.scrollToLine(tab.editorView, Math.round(targetLine));
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

                // Preview indicator for active .md tabs
                if (tab.previewMode && tab.previewMode !== 'off' && isMarkdownFile(tab.filePath)) {
                    var previewIcon = document.createElement('span');
                    previewIcon.className = 'text-accent-text flex items-center';
                    previewIcon.innerHTML = '<svg class="w-2.5 h-2.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>';
                    tabEl.appendChild(previewIcon);
                }

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

        updatePreviewButtons(paneId);
    }

    // --- Attach CodeMirror to a tab ---
    function attachEditorToTab(paneId, tabId) {
        var pane = editorPanes[paneId];
        if (!pane || !pane.editorContainer) return;

        var tab = findTabById(paneId, tabId);
        if (!tab) return;

        // Remove empty state placeholder if present
        var placeholder = pane.editorContainer.querySelector('.text-th-text-faint');
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
                if (tab.previewMode && tab.previewMode !== 'off') {
                    debouncedRenderPreview(paneId, tab.id);
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
                            if (t.previewMode && t.previewMode !== 'off') {
                                debouncedRenderPreview(pid, t.id);
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

        // Restore preview mode for tabs that had it active
        for (var j = 0; j < pane.tabs.length; j++) {
            (function(t) {
                if (t.previewMode && t.previewMode !== 'off') {
                    var savedMode = t.previewMode;
                    t.previewMode = 'off'; // Reset so setPreviewMode can re-apply
                    t.previewEl = null;
                    requestAnimationFrame(function() {
                        setPreviewMode(paneId, t.id, savedMode);
                    });
                }
            })(pane.tabs[j]);
        }
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

                // Track recently opened files for command palette
                var fileName = filePath.split('/').pop();
                if (window.ClawIDEPalette && window.ClawIDEPalette.addRecentFile) {
                    window.ClawIDEPalette.addRecentFile(fileName, filePath);
                }
            })
            .catch(function(err) {
                console.error('Failed to load file:', err);
            });

        setFocusedPane(targetPaneId);
    }

    function reuseTab(paneId, tab, filePath, content) {
        // Clean up preview state when switching files
        var wasMd = isMarkdownFile(tab.filePath);
        var willBeMd = isMarkdownFile(filePath);

        if (wasMd && !willBeMd) {
            // Switching from .md to non-.md: reset preview
            if (tab.previewMode !== 'off') {
                setPreviewMode(paneId, tab.id, 'off');
            }
            tab.previewMode = 'off';
        }

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

        // Alt+Z: toggle word wrap
        if (e.altKey && e.key === 'z' && !e.metaKey && !e.ctrlKey && !e.shiftKey) {
            e.preventDefault();
            if (window.ClawIDECommands && window.ClawIDECommands.toggleWordWrap) {
                window.ClawIDECommands.toggleWordWrap();
            }
        }

        // Cmd+Shift+V / Ctrl+Shift+V: toggle markdown preview
        if ((e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'v') {
            var pidPreview = getFocusedPaneId();
            if (pidPreview) {
                var previewTab = getActiveTab(pidPreview);
                if (previewTab && isMarkdownFile(previewTab.filePath)) {
                    e.preventDefault();
                    // Cycle: off → side → preview → off
                    var modes = ['off', 'side', 'preview'];
                    var idx = modes.indexOf(previewTab.previewMode || 'off');
                    var nextMode = modes[(idx + 1) % modes.length];
                    setPreviewMode(pidPreview, previewTab.id, nextMode);
                }
            }
        }

        // Cmd+B / Ctrl+B: toggle sidebar collapse
        if ((e.metaKey || e.ctrlKey) && e.key === 'b') {
            e.preventDefault();
            if (window.ClawIDESidebar && window.ClawIDESidebar.toggleCollapse) {
                window.ClawIDESidebar.toggleCollapse();
            }
        }
    });

    // --- Load file using explicit URLs (for feature workspaces) ---
    function loadFileFromURL(fetchURL, filePath, saveBaseURL) {
        if (!rootContainer) {
            rootContainer = document.getElementById('editor-pane-root');
            if (rootContainer && rootContainer.dataset.projectId) {
                projectID = rootContainer.dataset.projectId;
            }
        }

        highlightFileInTree(filePath);

        var targetPaneId = getFocusedPaneId();

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

        var existingTab = findTabByPath(targetPaneId, filePath);
        if (existingTab) {
            switchToTab(targetPaneId, existingTab.id);
            setFocusedPane(targetPaneId);
            return;
        }

        fetch(fetchURL)
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to load file (HTTP ' + resp.status + ')');
                return resp.text();
            })
            .then(function(content) {
                var activeTab = getActiveTab(targetPaneId);

                if (activeTab && !activeTab.modified && activeTab.filePath) {
                    reuseTab(targetPaneId, activeTab, filePath, content);
                } else if (activeTab && !activeTab.filePath) {
                    reuseTab(targetPaneId, activeTab, filePath, content);
                } else {
                    openNewTab(targetPaneId, filePath, content);
                }

                // Store saveBaseURL on the tab so saveTab can use the correct URL
                var tab = getActiveTab(targetPaneId);
                if (tab && saveBaseURL) {
                    tab.saveURL = saveBaseURL;
                }

                // Track recently opened files for command palette
                var fileName = filePath.split('/').pop();
                if (window.ClawIDEPalette && window.ClawIDEPalette.addRecentFile) {
                    window.ClawIDEPalette.addRecentFile(fileName, filePath);
                }
            })
            .catch(function(err) {
                console.error('Failed to load file:', err);
            });

        setFocusedPane(targetPaneId);
    }

    // --- Handle rename of open tabs ---
    function handleRename(oldPath, newPath) {
        var paneIds = Object.keys(editorPanes);
        for (var p = 0; p < paneIds.length; p++) {
            var pane = editorPanes[paneIds[p]];
            if (!pane) continue;
            var needsRender = false;
            for (var t = 0; t < pane.tabs.length; t++) {
                var tab = pane.tabs[t];
                if (!tab.filePath) continue;

                // Exact match (file rename) or prefix match (directory rename)
                if (tab.filePath === oldPath) {
                    tab.filePath = newPath;
                    needsRender = true;
                } else if (tab.filePath.indexOf(oldPath + '/') === 0) {
                    tab.filePath = newPath + tab.filePath.substring(oldPath.length);
                    needsRender = true;
                }

                // Update saveURL if it contains the old path
                if (needsRender && tab.saveURL) {
                    tab.saveURL = tab.saveURL.replace(
                        'path=' + encodeURIComponent(oldPath),
                        'path=' + encodeURIComponent(tab.filePath)
                    );
                }
            }
            if (needsRender) {
                renderTabs(paneIds[p]);
            }
        }
    }

    // --- Handle delete of open tabs ---
    function handleDelete(deletedPath, isDir) {
        var paneIds = Object.keys(editorPanes);
        for (var p = 0; p < paneIds.length; p++) {
            var pane = editorPanes[paneIds[p]];
            if (!pane) continue;

            // Collect tabs to close (iterate in reverse to avoid index shifting)
            var tabsToClose = [];
            for (var t = 0; t < pane.tabs.length; t++) {
                var tab = pane.tabs[t];
                if (!tab.filePath) continue;

                if (tab.filePath === deletedPath) {
                    tabsToClose.push(tab.id);
                } else if (isDir && tab.filePath.indexOf(deletedPath + '/') === 0) {
                    tabsToClose.push(tab.id);
                }
            }

            // Close affected tabs (mark as not modified to skip confirm dialogs)
            for (var c = 0; c < tabsToClose.length; c++) {
                var tabToClose = findTabById(paneIds[p], tabsToClose[c]);
                if (tabToClose) {
                    tabToClose.modified = false; // skip "discard changes?" since the file is gone
                    closeTab(paneIds[p], tabsToClose[c]);
                }
            }
        }
    }

    // --- Expose to global scope ---
    window.ClawIDEEditor = {
        loadFile: loadFile,
        loadFileFromURL: loadFileFromURL,
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
        handleRename: handleRename,
        handleDelete: handleDelete,
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
        getActiveEditorView: function() {
            var pid = getFocusedPaneId();
            if (!pid) return null;
            var tab = getActiveTab(pid);
            return tab ? tab.editorView : null;
        },
        getActiveTab: function() {
            var pid = getFocusedPaneId();
            if (!pid) return null;
            return getActiveTab(pid);
        },
    };
})();
