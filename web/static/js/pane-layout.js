// ClawIDE Pane Layout Manager
// Renders PaneNode trees into split terminal panes with resize handles.
// On phone screens (<768px), renders a carousel view showing one pane at a time.
(function() {
    'use strict';

    var activeLayouts = {}; // keyed by sessionID

    // --- Phone detection ---
    var phoneQuery = window.matchMedia('(max-width: 767px)');

    function isPhoneLayout() {
        return phoneQuery.matches;
    }

    // Carousel state per session: { currentIndex, paneNodes, container, projectID }
    var carouselState = {};

    // --- Render entry point ---

    function renderLayout(container, layoutJSON, sessionID, projectID) {
        // Destroy existing terminals for this session
        if (activeLayouts[sessionID]) {
            activeLayouts[sessionID].forEach(function(paneID) {
                window.ClawIDETerminal.destroy(paneID);
            });
        }

        container.innerHTML = '';

        // Store layout JSON on container for re-render on resize boundary crossing
        container.dataset.layout = JSON.stringify(layoutJSON);
        container.dataset.sessionId = sessionID;
        container.dataset.projectId = projectID;

        var paneIDs = [];

        if (isPhoneLayout()) {
            buildCarousel(container, layoutJSON, sessionID, projectID, paneIDs);
        } else {
            buildNode(container, layoutJSON, sessionID, projectID, paneIDs);
        }

        activeLayouts[sessionID] = paneIDs;

        if (isPhoneLayout()) {
            // In carousel, only init the visible pane (lazy init)
            var cs = carouselState[sessionID];
            if (cs && cs.paneNodes.length > 0) {
                var visibleNode = cs.paneNodes[cs.currentIndex];
                requestAnimationFrame(function() {
                    initPaneTerminal(sessionID, visibleNode.pane_id);
                    handleDeepLink(paneIDs, sessionID);
                });
            }
        } else {
            // Desktop: init all terminals
            requestAnimationFrame(function() {
                paneIDs.forEach(function(paneID) {
                    initPaneTerminal(sessionID, paneID);
                });
                handleDeepLink(paneIDs, sessionID);
            });
        }
    }

    // Initialize a terminal in its container if not already created
    function initPaneTerminal(sessionID, paneID) {
        var paneContainer = document.getElementById('pane-' + paneID);
        if (paneContainer && !paneContainer.dataset.initialized) {
            window.ClawIDETerminal.create(sessionID, paneID, paneContainer);
            paneContainer.dataset.initialized = 'true';
        }
    }

    // Handle ?pane= deep-link query param
    function handleDeepLink(paneIDs, sessionID) {
        var urlParams = new URLSearchParams(window.location.search);
        var targetPane = urlParams.get('pane');
        if (targetPane && paneIDs.indexOf(targetPane) !== -1) {
            if (isPhoneLayout()) {
                // Navigate carousel to the target pane
                var cs = carouselState[sessionID];
                if (cs) {
                    for (var i = 0; i < cs.paneNodes.length; i++) {
                        if (cs.paneNodes[i].pane_id === targetPane) {
                            navigateCarousel(sessionID, i);
                            break;
                        }
                    }
                }
            } else {
                setTimeout(function() {
                    window.ClawIDETerminal.focusPane(targetPane);
                }, 200);
            }
            var cleanURL = window.location.pathname + window.location.hash;
            window.history.replaceState(null, '', cleanURL);
        } else if (paneIDs.length > 0) {
            window.ClawIDETerminal.setFocusedPaneID(paneIDs[0]);
        }
    }

    // --- Desktop: buildNode (original split layout) ---

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

            // Pane name (editable on double-click)
            var nameSpan = createPaneName(node, projectID, sessionID);
            toolbarLeft.appendChild(nameSpan);

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

    // --- Pane naming ---

    function createPaneName(node, projectID, sessionID) {
        var nameSpan = document.createElement('span');
        nameSpan.className = 'pane-name';
        nameSpan.textContent = node.name || 'Terminal';
        nameSpan.title = 'Double-click to rename';

        nameSpan.addEventListener('dblclick', function() {
            nameSpan.contentEditable = 'true';
            nameSpan.focus();
            // Select all text
            var range = document.createRange();
            range.selectNodeContents(nameSpan);
            var sel = window.getSelection();
            sel.removeAllRanges();
            sel.addRange(range);
        });

        function commitName() {
            nameSpan.contentEditable = 'false';
            var newName = nameSpan.textContent.trim();
            if (newName === 'Terminal') newName = '';
            node.name = newName;
            nameSpan.textContent = newName || 'Terminal';
            renamePane(projectID, sessionID, node.pane_id, newName);
        }

        nameSpan.addEventListener('blur', commitName);
        nameSpan.addEventListener('keydown', function(e) {
            if (e.key === 'Enter') {
                e.preventDefault();
                nameSpan.blur();
            }
            if (e.key === 'Escape') {
                nameSpan.textContent = node.name || 'Terminal';
                nameSpan.contentEditable = 'false';
            }
        });

        return nameSpan;
    }

    // --- Carousel mode (phone <768px) ---

    function collectLeavesOrdered(node) {
        if (!node) return [];
        if (node.type === 'leaf') return [node];
        var leaves = [];
        if (node.first) leaves = leaves.concat(collectLeavesOrdered(node.first));
        if (node.second) leaves = leaves.concat(collectLeavesOrdered(node.second));
        return leaves;
    }

    function buildCarousel(container, layoutJSON, sessionID, projectID, paneIDs) {
        var leaves = collectLeavesOrdered(layoutJSON);
        if (leaves.length === 0) return;

        // Collect pane IDs
        leaves.forEach(function(leaf) { paneIDs.push(leaf.pane_id); });

        // Determine starting index (try to preserve from previous state)
        var prevState = carouselState[sessionID];
        var startIndex = 0;
        if (prevState && prevState.currentIndex < leaves.length) {
            startIndex = prevState.currentIndex;
        }

        // Build carousel wrapper
        var wrapper = document.createElement('div');
        wrapper.className = 'carousel-wrapper flex flex-col flex-1 min-h-0';

        // --- Header ---
        var header = document.createElement('div');
        header.className = 'carousel-header';

        // Prev button
        var prevBtn = document.createElement('button');
        prevBtn.className = 'carousel-nav-btn';
        prevBtn.innerHTML = '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="15 18 9 12 15 6"/></svg>';
        prevBtn.setAttribute('aria-label', 'Previous pane');
        prevBtn.onclick = function() { navigateCarousel(sessionID, carouselState[sessionID].currentIndex - 1); };
        header.appendChild(prevBtn);

        // Center: pane name + indicator
        var center = document.createElement('div');
        center.className = 'flex-1 flex items-center justify-center gap-2 min-w-0';

        var nameSpan = createPaneName(leaves[startIndex], projectID, sessionID);
        nameSpan.className = 'carousel-pane-name';
        center.appendChild(nameSpan);

        var indicator = document.createElement('span');
        indicator.className = 'text-xs text-gray-500 flex-shrink-0';
        indicator.textContent = (startIndex + 1) + '/' + leaves.length;
        center.appendChild(indicator);

        header.appendChild(center);

        // Next button
        var nextBtn = document.createElement('button');
        nextBtn.className = 'carousel-nav-btn';
        nextBtn.innerHTML = '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 18 15 12 9 6"/></svg>';
        nextBtn.setAttribute('aria-label', 'Next pane');
        nextBtn.onclick = function() { navigateCarousel(sessionID, carouselState[sessionID].currentIndex + 1); };
        header.appendChild(nextBtn);

        // Close button
        var closeBtn = document.createElement('button');
        closeBtn.className = 'carousel-nav-btn text-gray-500 hover:text-red-400';
        closeBtn.innerHTML = '&#x2715;';
        closeBtn.title = 'Close pane';
        closeBtn.onclick = function() {
            var cs = carouselState[sessionID];
            if (cs) {
                closePane(projectID, sessionID, cs.paneNodes[cs.currentIndex].pane_id);
            }
        };
        header.appendChild(closeBtn);

        wrapper.appendChild(header);

        // --- Slides ---
        var slidesContainer = document.createElement('div');
        slidesContainer.className = 'carousel-slides flex-1 min-h-0 relative';

        var slides = [];
        leaves.forEach(function(leaf, idx) {
            var slide = document.createElement('div');
            slide.className = 'absolute inset-0 flex flex-col';
            slide.style.display = idx === startIndex ? 'flex' : 'none';

            var xtermContainer = document.createElement('div');
            xtermContainer.id = 'pane-' + leaf.pane_id;
            xtermContainer.className = 'xterm-container flex-1';
            slide.appendChild(xtermContainer);

            slidesContainer.appendChild(slide);
            slides.push(slide);
        });

        wrapper.appendChild(slidesContainer);
        container.appendChild(wrapper);

        // Update arrow visibility
        if (leaves.length <= 1) {
            prevBtn.style.visibility = 'hidden';
            nextBtn.style.visibility = 'hidden';
        } else {
            prevBtn.style.visibility = startIndex === 0 ? 'hidden' : 'visible';
            nextBtn.style.visibility = startIndex === leaves.length - 1 ? 'hidden' : 'visible';
        }

        // Store carousel state
        carouselState[sessionID] = {
            currentIndex: startIndex,
            paneNodes: leaves,
            slides: slides,
            nameSpan: nameSpan,
            indicator: indicator,
            prevBtn: prevBtn,
            nextBtn: nextBtn,
            container: container,
            projectID: projectID,
        };
    }

    function navigateCarousel(sessionID, newIndex) {
        var cs = carouselState[sessionID];
        if (!cs) return;

        // Clamp index
        newIndex = Math.max(0, Math.min(newIndex, cs.paneNodes.length - 1));
        if (newIndex === cs.currentIndex) return;

        var oldIndex = cs.currentIndex;
        cs.currentIndex = newIndex;

        // Hide old slide, show new slide
        cs.slides[oldIndex].style.display = 'none';
        cs.slides[newIndex].style.display = 'flex';

        // Update header
        var node = cs.paneNodes[newIndex];
        var newNameSpan = createPaneName(node, cs.projectID, sessionID);
        newNameSpan.className = 'carousel-pane-name';
        cs.nameSpan.parentNode.replaceChild(newNameSpan, cs.nameSpan);
        cs.nameSpan = newNameSpan;

        cs.indicator.textContent = (newIndex + 1) + '/' + cs.paneNodes.length;

        // Update arrow visibility
        cs.prevBtn.style.visibility = newIndex === 0 ? 'hidden' : 'visible';
        cs.nextBtn.style.visibility = newIndex === cs.paneNodes.length - 1 ? 'hidden' : 'visible';

        // Lazy-init terminal if not yet created
        initPaneTerminal(sessionID, node.pane_id);

        // Focus terminal and fit
        requestAnimationFrame(function() {
            window.ClawIDETerminal.focusPane(node.pane_id);
            window.ClawIDETerminal.setFocusedPaneID(node.pane_id);
        });
    }

    // --- Desktop helpers ---

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

    // --- Phone/desktop transition on resize ---

    phoneQuery.addEventListener('change', function() {
        // Re-render all visible session layouts when crossing the 768px boundary
        var containers = document.querySelectorAll('[data-layout]');
        containers.forEach(function(container) {
            var layoutJSON = container.dataset.layout;
            var sessionID = container.dataset.sessionId;
            var projectID = container.dataset.projectId;
            if (layoutJSON && sessionID && projectID) {
                try {
                    var layout = JSON.parse(layoutJSON);
                    renderLayout(container, layout, sessionID, projectID);
                } catch (e) {
                    console.error('Failed to re-render layout on resize:', e);
                }
            }
        });
    });

    // --- API functions ---

    function splitPane(projectID, sessionID, paneID, direction) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID + '/split', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'direction=' + encodeURIComponent(direction),
        })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            var container = document.getElementById('session-panes-' + sessionID);
            if (container) {
                renderLayout(container, data.layout, sessionID, projectID);
            }
        })
        .catch(function(err) {
            console.error('Failed to split pane:', err);
        });
    }

    function closePane(projectID, sessionID, paneID) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID, {
            method: 'DELETE',
        })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            if (data.session_closed) {
                window.location.reload();
                return;
            }
            var container = document.getElementById('session-panes-' + sessionID);
            if (container) {
                renderLayout(container, data.layout, sessionID, projectID);
            }
        })
        .catch(function(err) {
            console.error('Failed to close pane:', err);
        });
    }

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

    function renamePane(projectID, sessionID, paneID, name) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID + '/rename', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: 'name=' + encodeURIComponent(name),
        })
        .catch(function(err) {
            console.error('Failed to rename pane:', err);
        });
    }

    // --- Expose to global scope ---
    window.ClawIDEPaneLayout = {
        render: renderLayout,
        splitPane: splitPane,
        closePane: closePane,
        renamePane: renamePane,
        isPhoneLayout: isPhoneLayout,
        navigateCarousel: navigateCarousel,
        getCarouselState: function() { return carouselState; },
    };
})();
