// ClawIDE Pane Layout Manager
// Renders PaneNode trees into split terminal panes with resize handles.
(function() {
    'use strict';

    var activeLayouts = {}; // keyed by sessionID

    // Render the pane layout tree into the given container
    function renderLayout(container, layoutJSON, sessionID, projectID) {
        // Destroy existing terminals for this session
        if (activeLayouts[sessionID]) {
            activeLayouts[sessionID].forEach(function(paneID) {
                window.ClawIDETerminal.destroy(paneID);
            });
        }

        container.innerHTML = '';
        var paneIDs = [];

        buildNode(container, layoutJSON, sessionID, projectID, paneIDs);

        activeLayouts[sessionID] = paneIDs;

        // Initialize terminals for each leaf pane after DOM is ready
        requestAnimationFrame(function() {
            paneIDs.forEach(function(paneID) {
                var paneContainer = document.getElementById('pane-' + paneID);
                if (paneContainer) {
                    window.ClawIDETerminal.create(sessionID, paneID, paneContainer);
                }
            });
        });
    }

    function buildNode(parent, node, sessionID, projectID, paneIDs) {
        if (!node) return;

        if (node.type === 'leaf') {
            var leafEl = document.createElement('div');
            leafEl.className = 'pane-leaf flex flex-col flex-1 min-w-0 min-h-0';

            // Mini toolbar
            var toolbar = document.createElement('div');
            toolbar.className = 'pane-toolbar';

            var toolbarLeft = document.createElement('div');
            toolbarLeft.className = 'flex items-center gap-1 flex-1';

            // Split horizontal button
            var splitH = createToolbarButton('Split H', 'horizontal', function() {
                splitPane(projectID, sessionID, node.pane_id, 'horizontal');
            });
            toolbarLeft.appendChild(splitH);

            // Split vertical button
            var splitV = createToolbarButton('Split V', 'vertical', function() {
                splitPane(projectID, sessionID, node.pane_id, 'vertical');
            });
            toolbarLeft.appendChild(splitV);

            toolbar.appendChild(toolbarLeft);

            // Close button
            var closeBtn = document.createElement('button');
            closeBtn.className = 'text-gray-500 hover:text-red-400 px-1 transition-colors';
            closeBtn.innerHTML = '&#x2715;';
            closeBtn.title = 'Close pane';
            closeBtn.onclick = function() {
                closePane(projectID, sessionID, node.pane_id);
            };
            toolbar.appendChild(closeBtn);

            leafEl.appendChild(toolbar);

            // xterm container
            var xtermContainer = document.createElement('div');
            xtermContainer.id = 'pane-' + node.pane_id;
            xtermContainer.className = 'xterm-container flex-1';
            leafEl.appendChild(xtermContainer);

            parent.appendChild(leafEl);
            paneIDs.push(node.pane_id);
            return;
        }

        if (node.type === 'split') {
            var splitEl = document.createElement('div');
            var isHorizontal = node.direction === 'horizontal';
            splitEl.className = 'flex flex-1 min-w-0 min-h-0 ' + (isHorizontal ? 'flex-row' : 'flex-col');

            // First child
            var firstEl = document.createElement('div');
            firstEl.className = 'flex min-w-0 min-h-0';
            firstEl.style.flex = '0 0 ' + ((node.ratio || 0.5) * 100) + '%';
            buildNode(firstEl, node.first, sessionID, projectID, paneIDs);
            splitEl.appendChild(firstEl);

            // Resize handle
            var handle = document.createElement('div');
            handle.className = 'pane-resize-handle';
            handle.dataset.direction = isHorizontal ? 'horizontal' : 'vertical';
            setupResizeHandle(handle, firstEl, splitEl, node, isHorizontal, projectID, sessionID);
            splitEl.appendChild(handle);

            // Second child
            var secondEl = document.createElement('div');
            secondEl.className = 'flex min-w-0 min-h-0 flex-1';
            buildNode(secondEl, node.second, sessionID, projectID, paneIDs);
            splitEl.appendChild(secondEl);

            parent.appendChild(splitEl);
        }
    }

    function createToolbarButton(label, direction, onclick) {
        var btn = document.createElement('button');
        btn.className = 'text-gray-500 hover:text-gray-300 px-1 transition-colors';
        btn.title = 'Split ' + direction;

        if (direction === 'horizontal') {
            btn.innerHTML = '<svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor"><rect x="1" y="2" width="6" height="12" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="9" y="2" width="6" height="12" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>';
        } else {
            btn.innerHTML = '<svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor"><rect x="2" y="1" width="12" height="6" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/><rect x="2" y="9" width="12" height="6" rx="1" fill="none" stroke="currentColor" stroke-width="1.5"/></svg>';
        }

        btn.onclick = onclick;
        return btn;
    }

    function setupResizeHandle(handle, firstEl, splitEl, node, isHorizontal, projectID, sessionID) {
        var dragging = false;
        var startPos = 0;
        var startSize = 0;
        var totalSize = 0;

        // Find the first leaf pane ID in either child for resize persist
        function findLeafPaneID(n) {
            if (!n) return null;
            if (n.type === 'leaf') return n.pane_id;
            return findLeafPaneID(n.first) || findLeafPaneID(n.second);
        }

        handle.addEventListener('mousedown', function(e) {
            e.preventDefault();
            dragging = true;
            startPos = isHorizontal ? e.clientX : e.clientY;
            startSize = isHorizontal ? firstEl.offsetWidth : firstEl.offsetHeight;
            totalSize = isHorizontal ? splitEl.offsetWidth : splitEl.offsetHeight;

            document.body.style.cursor = isHorizontal ? 'col-resize' : 'row-resize';
            document.body.style.userSelect = 'none';

            function onMouseMove(e) {
                if (!dragging) return;
                var currentPos = isHorizontal ? e.clientX : e.clientY;
                var delta = currentPos - startPos;
                var newSize = startSize + delta;
                var handleSize = isHorizontal ? handle.offsetWidth : handle.offsetHeight;
                var ratio = newSize / (totalSize - handleSize);

                // Clamp ratio
                ratio = Math.max(0.1, Math.min(0.9, ratio));
                firstEl.style.flex = '0 0 ' + (ratio * 100) + '%';

                // Refit any terminals in the affected panes
                Object.keys(window.ClawIDETerminal || {}).forEach(function() {
                    // Terminals will auto-resize via ResizeObserver
                });
            }

            function onMouseUp(e) {
                if (!dragging) return;
                dragging = false;
                document.body.style.cursor = '';
                document.body.style.userSelect = '';

                document.removeEventListener('mousemove', onMouseMove);
                document.removeEventListener('mouseup', onMouseUp);

                // Persist ratio
                var currentPos = isHorizontal ? e.clientX : e.clientY;
                var delta = currentPos - startPos;
                var newSize = startSize + delta;
                var handleSize = isHorizontal ? handle.offsetWidth : handle.offsetHeight;
                var ratio = newSize / (totalSize - handleSize);
                ratio = Math.max(0.1, Math.min(0.9, ratio));

                var paneID = findLeafPaneID(node.first);
                if (paneID) {
                    persistResize(projectID, sessionID, paneID, ratio);
                }
            }

            document.addEventListener('mousemove', onMouseMove);
            document.addEventListener('mouseup', onMouseUp);
        });
    }

    // API: split a pane
    function splitPane(projectID, sessionID, paneID, direction) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID + '/split', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'direction=' + encodeURIComponent(direction),
        })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            // Re-render the layout with the updated tree
            var container = document.getElementById('session-panes-' + sessionID);
            if (container) {
                renderLayout(container, data.layout, sessionID, projectID);
            }
        })
        .catch(function(err) {
            console.error('Failed to split pane:', err);
        });
    }

    // API: close a pane
    function closePane(projectID, sessionID, paneID) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID, {
            method: 'DELETE',
        })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.session_closed) {
                // Session was deleted — reload to update tab bar
                window.location.reload();
                return;
            }
            // Re-render the layout with the updated tree
            var container = document.getElementById('session-panes-' + sessionID);
            if (container) {
                renderLayout(container, data.layout, sessionID, projectID);
            }
        })
        .catch(function(err) {
            console.error('Failed to close pane:', err);
        });
    }

    // API: persist resize ratio
    function persistResize(projectID, sessionID, paneID, ratio) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID + '/resize', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ratio: ratio }),
        })
        .catch(function(err) {
            console.error('Failed to persist resize:', err);
        });
    }

    // Expose to global scope
    window.ClawIDEPaneLayout = {
        render: renderLayout,
        splitPane: splitPane,
        closePane: closePane,
    };
})();
