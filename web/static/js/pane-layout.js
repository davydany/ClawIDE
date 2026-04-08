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

    // --- Drag-and-drop state (desktop only) ---
    var dragState = null; // { paneID, sessionID, projectID, leafEl, pending }

    function getDropPosition(e, rect) {
        var x = (e.clientX - rect.left) / rect.width;
        var y = (e.clientY - rect.top) / rect.height;
        var dL = x, dR = 1 - x, dT = y, dB = 1 - y;
        var min = Math.min(dL, dR, dT, dB);
        if (min === dL) return 'left';
        if (min === dR) return 'right';
        if (min === dT) return 'top';
        return 'bottom';
    }

    function showDropIndicator(leafEl, position) {
        // Remove existing indicator from this leaf
        var existing = leafEl.querySelector('.drop-indicator');
        if (existing) existing.remove();

        var indicator = document.createElement('div');
        indicator.className = 'drop-indicator';

        switch (position) {
            case 'left':
                indicator.style.cssText = 'left:0;top:0;width:50%;height:100%;border-right:2px solid rgba(99,102,241,0.6);';
                break;
            case 'right':
                indicator.style.cssText = 'left:50%;top:0;width:50%;height:100%;border-left:2px solid rgba(99,102,241,0.6);';
                break;
            case 'top':
                indicator.style.cssText = 'left:0;top:0;width:100%;height:50%;border-bottom:2px solid rgba(99,102,241,0.6);';
                break;
            case 'bottom':
                indicator.style.cssText = 'left:0;top:50%;width:100%;height:50%;border-top:2px solid rgba(99,102,241,0.6);';
                break;
        }

        leafEl.appendChild(indicator);
    }

    function removeDropIndicator(leafEl) {
        var existing = leafEl.querySelector('.drop-indicator');
        if (existing) existing.remove();
    }

    function removeAllDropIndicators() {
        var indicators = document.querySelectorAll('.drop-indicator');
        indicators.forEach(function(el) { el.remove(); });
    }

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

            var toolbarRight = document.createElement('div');
            toolbarRight.className = 'flex items-center gap-1';

            // Kebab menu (3-dot)
            var kebabWrap = document.createElement('div');
            kebabWrap.className = 'relative';
            var kebabBtn = document.createElement('button');
            kebabBtn.className = 'text-gray-500 hover:text-gray-300 px-1 transition-colors';
            kebabBtn.innerHTML = '<svg class="w-3.5 h-3.5" viewBox="0 0 16 16" fill="currentColor"><circle cx="8" cy="3" r="1.5"/><circle cx="8" cy="8" r="1.5"/><circle cx="8" cy="13" r="1.5"/></svg>';
            kebabBtn.title = 'More options';
            var kebabMenu = document.createElement('div');
            kebabMenu.className = 'absolute right-0 top-full mt-1 bg-gray-800 border border-gray-700 rounded-lg shadow-xl py-1 z-50 min-w-[160px] hidden';

            var menuItemAgent = document.createElement('button');
            menuItemAgent.className = 'w-full text-left px-3 py-1.5 text-xs text-gray-300 hover:bg-gray-700 hover:text-white';
            menuItemAgent.textContent = 'New Agent Pane';
            menuItemAgent.onclick = function() {
                kebabMenu.classList.add('hidden');
                splitPane(projectID, sessionID, node.pane_id, 'horizontal', 'agent');
            };
            kebabMenu.appendChild(menuItemAgent);

            var menuItemShell = document.createElement('button');
            menuItemShell.className = 'w-full text-left px-3 py-1.5 text-xs text-gray-300 hover:bg-gray-700 hover:text-white';
            menuItemShell.textContent = 'New Shell Pane';
            menuItemShell.onclick = function() {
                kebabMenu.classList.add('hidden');
                splitPane(projectID, sessionID, node.pane_id, 'horizontal', 'shell');
            };
            kebabMenu.appendChild(menuItemShell);

            // Rename button
            var menuItemRename = document.createElement('button');
            menuItemRename.className = 'w-full text-left px-3 py-1.5 text-xs text-gray-300 hover:bg-gray-700 hover:text-white';
            menuItemRename.textContent = 'Rename';
            menuItemRename.onclick = function() {
                kebabMenu.classList.add('hidden');
                nameSpan.dispatchEvent(new MouseEvent('dblclick'));
            };
            kebabMenu.appendChild(menuItemRename);

            // Add separator
            var separator = document.createElement('div');
            separator.className = 'border-t border-gray-700 my-1';
            kebabMenu.appendChild(separator);

            // Paste button
            var menuItemPaste = document.createElement('button');
            menuItemPaste.className = 'w-full text-left px-3 py-1.5 text-xs text-gray-300 hover:bg-gray-700 hover:text-white';
            menuItemPaste.textContent = 'Paste from Clipboard';
            menuItemPaste.onclick = function() {
                kebabMenu.classList.add('hidden');
                if (window.ClawIDETerminal) {
                    window.ClawIDETerminal.paste(node.pane_id);
                }
            };
            kebabMenu.appendChild(menuItemPaste);

            kebabBtn.onclick = function(e) {
                e.stopPropagation();
                kebabMenu.classList.toggle('hidden');
            };

            // Close menu on outside click
            document.addEventListener('click', function() {
                kebabMenu.classList.add('hidden');
            });

            kebabWrap.appendChild(kebabBtn);
            kebabWrap.appendChild(kebabMenu);
            toolbarRight.appendChild(kebabWrap);

            // Close button
            var closeBtn = document.createElement('button');
            closeBtn.className = 'text-gray-500 hover:text-red-400 px-1 transition-colors';
            closeBtn.innerHTML = '&#x2715;';
            closeBtn.title = 'Close pane';
            closeBtn.onclick = function() {
                closePane(projectID, sessionID, node.pane_id);
            };
            toolbarRight.appendChild(closeBtn);
            toolbar.appendChild(toolbarRight);

            leafEl.appendChild(toolbar);

            // --- Drag-and-drop support (desktop only) ---
            if (!isPhoneLayout()) {
                // Make toolbar draggable
                toolbar.setAttribute('draggable', 'true');

                // Prevent buttons from initiating drag
                var toolbarButtons = toolbar.querySelectorAll('button');
                toolbarButtons.forEach(function(btn) {
                    btn.setAttribute('draggable', 'false');
                });

                toolbar.addEventListener('dragstart', function(e) {
                    if (isPhoneLayout()) { e.preventDefault(); return; }
                    // Need at least 2 panes to move
                    if (activeLayouts[sessionID] && activeLayouts[sessionID].length < 2) {
                        e.preventDefault();
                        return;
                    }
                    dragState = {
                        paneID: node.pane_id,
                        sessionID: sessionID,
                        projectID: projectID,
                        leafEl: leafEl,
                        pending: false
                    };
                    e.dataTransfer.effectAllowed = 'move';
                    e.dataTransfer.setData('text/plain', node.pane_id);
                    // Defer adding class so the drag image captures the original look
                    requestAnimationFrame(function() {
                        leafEl.classList.add('dragging');
                    });
                });

                toolbar.addEventListener('dragend', function() {
                    leafEl.classList.remove('dragging');
                    removeAllDropIndicators();
                    dragState = null;
                });

                // Drop zone listeners on the leaf element
                leafEl.style.position = 'relative'; // for absolute-positioned indicator

                leafEl.addEventListener('dragover', function(e) {
                    if (!dragState || dragState.paneID === node.pane_id || dragState.pending) return;
                    e.preventDefault();
                    e.dataTransfer.dropEffect = 'move';
                    var rect = leafEl.getBoundingClientRect();
                    var position = getDropPosition(e, rect);
                    showDropIndicator(leafEl, position);
                });

                leafEl.addEventListener('dragleave', function(e) {
                    // Only remove if actually leaving the leaf (not entering a child)
                    if (!leafEl.contains(e.relatedTarget)) {
                        removeDropIndicator(leafEl);
                    }
                });

                leafEl.addEventListener('drop', function(e) {
                    e.preventDefault();
                    if (!dragState || dragState.paneID === node.pane_id || dragState.pending) return;
                    var rect = leafEl.getBoundingClientRect();
                    var position = getDropPosition(e, rect);
                    removeAllDropIndicators();
                    dragState.pending = true;
                    movePane(projectID, sessionID, dragState.paneID, node.pane_id, position);
                });
            }

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

    function splitPane(projectID, sessionID, paneID, direction, paneType) {
        var body = 'direction=' + encodeURIComponent(direction);
        if (paneType) {
            body += '&pane_type=' + encodeURIComponent(paneType);
        }
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + paneID + '/split', {
            method: 'POST',
            headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
            body: body,
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
        // Close the WebSocket for this pane first to free an HTTP/1.1 connection
        // slot. Without this, the DELETE request stays pending because all 6
        // browser connections per host are occupied by WebSocket + SSE streams.
        window.ClawIDETerminal.destroy(paneID);

        // Small delay lets the browser release the TCP socket after the WS
        // close handshake, ensuring a free slot for the DELETE request.
        setTimeout(function() {
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
        }, 50);
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

    function movePane(projectID, sessionID, sourcePaneID, targetPaneID, position) {
        fetch('/projects/' + projectID + '/sessions/' + sessionID + '/panes/' + sourcePaneID + '/move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_pane_id: targetPaneID, position: position }),
        })
        .then(function(r) { return r.json(); })
        .then(function(data) {
            var container = document.getElementById('session-panes-' + sessionID);
            if (container) {
                rerenderWithTerminalPreservation(container, data.layout, sessionID, projectID);
            }
        })
        .catch(function(err) {
            console.error('Failed to move pane:', err);
        })
        .finally(function() {
            if (dragState) dragState.pending = false;
            dragState = null;
        });
    }

    // Re-render the layout while preserving live terminal instances (no WS reconnect)
    function rerenderWithTerminalPreservation(container, layoutJSON, sessionID, projectID) {
        var oldPaneIDs = activeLayouts[sessionID] || [];

        // Detach all live terminals (disconnect ResizeObserver but keep xterm + WS)
        oldPaneIDs.forEach(function(paneID) {
            var ts = window.ClawIDETerminal.get(paneID);
            if (ts) {
                ts.detach();
            }
        });

        // Clear the container and rebuild DOM from new layout
        container.innerHTML = '';
        container.dataset.layout = JSON.stringify(layoutJSON);

        var newPaneIDs = [];
        buildNode(container, layoutJSON, sessionID, projectID, newPaneIDs);
        activeLayouts[sessionID] = newPaneIDs;

        // Reattach preserved terminals into their new containers
        requestAnimationFrame(function() {
            newPaneIDs.forEach(function(paneID) {
                var paneContainer = document.getElementById('pane-' + paneID);
                if (!paneContainer) return;

                var ts = window.ClawIDETerminal.get(paneID);
                if (ts) {
                    // Reattach existing terminal to new container
                    window.ClawIDETerminal.reattach(paneID, paneContainer);
                    paneContainer.dataset.initialized = 'true';
                } else {
                    // New pane (shouldn't happen in a move, but be safe)
                    initPaneTerminal(sessionID, paneID);
                }
            });

            // Destroy terminals for panes that no longer exist in the layout
            oldPaneIDs.forEach(function(paneID) {
                if (newPaneIDs.indexOf(paneID) === -1) {
                    window.ClawIDETerminal.destroy(paneID);
                }
            });

            // Restore focus
            if (newPaneIDs.length > 0) {
                var focusedID = window.ClawIDETerminal.getFocusedPaneID();
                if (focusedID && newPaneIDs.indexOf(focusedID) !== -1) {
                    window.ClawIDETerminal.focusPane(focusedID);
                } else {
                    window.ClawIDETerminal.setFocusedPaneID(newPaneIDs[0]);
                }
            }
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
        movePane: movePane,
        renamePane: renamePane,
        isPhoneLayout: isPhoneLayout,
        navigateCarousel: navigateCarousel,
        getCarouselState: function() { return carouselState; },
    };
})();
