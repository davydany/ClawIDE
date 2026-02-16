// ClawIDE Sidebar Resize + Collapse
// Drag-to-resize on the sidebar edge handle, clamp width 200-500px,
// collapse/expand toggle with separate mobile/desktop states,
// debounced persist via PUT /api/settings.
(function() {
    'use strict';

    var MIN_WIDTH = 200;
    var MAX_WIDTH = 500;
    var COLLAPSED_WIDTH = 40;
    var DEBOUNCE_MS = 300;
    var MOBILE_BREAKPOINT = '(max-width: 768px)';

    var handle, sidebar, position, startX, startWidth, persistTimer, collapseTimer;
    var sidebarCollapsed = false;
    var expandedWidth = null; // remember width before collapse

    // --- Mobile detection ---
    function isMobile() {
        return window.matchMedia(MOBILE_BREAKPOINT).matches;
    }

    function getStorageKey() {
        return isMobile() ? 'editor.preferences.sidebarCollapsedMobile' : 'editor.preferences.sidebarCollapsed';
    }

    // --- Collapse state management ---
    function loadCollapseState() {
        var key = getStorageKey();
        var stored = localStorage.getItem(key);
        return stored === 'true';
    }

    function saveCollapseState(collapsed) {
        var key = getStorageKey();
        localStorage.setItem(key, collapsed ? 'true' : 'false');

        // Debounced API persist
        clearTimeout(collapseTimer);
        collapseTimer = setTimeout(function() {
            var payload = {};
            if (isMobile()) {
                payload.sidebar_collapsed_mobile = collapsed;
            } else {
                payload.sidebar_collapsed = collapsed;
            }
            fetch('/api/settings', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            }).catch(function(err) {
                console.error('Failed to persist sidebar collapse state:', err);
            });
        }, DEBOUNCE_MS);
    }

    function applyCollapseState(collapsed) {
        if (!sidebar) return;
        sidebarCollapsed = collapsed;

        if (collapsed) {
            expandedWidth = sidebar.offsetWidth || parseInt(sidebar.style.width, 10) || MIN_WIDTH;
            sidebar.classList.add('collapsed');
            sidebar.style.width = COLLAPSED_WIDTH + 'px';
            if (handle) handle.style.display = 'none';
        } else {
            sidebar.classList.remove('collapsed');
            sidebar.style.width = (expandedWidth || MIN_WIDTH) + 'px';
            if (handle) {
                // Restore handle visibility (respects lg:block via CSS)
                handle.style.display = '';
            }
        }
    }

    function toggleSidebarCollapse() {
        applyCollapseState(!sidebarCollapsed);
        saveCollapseState(sidebarCollapsed);
    }

    // --- Resize: drag-to-resize ---
    function init() {
        handle = document.getElementById('sidebar-resize-handle');
        sidebar = document.getElementById('app-sidebar');
        if (!handle || !sidebar) return;

        position = handle.getAttribute('data-position') || 'left';

        handle.addEventListener('mousedown', onMouseDown);
        handle.addEventListener('touchstart', onTouchStart, { passive: false });

        // Load and apply collapse state on init
        var initialCollapsed = loadCollapseState();
        if (initialCollapsed) {
            expandedWidth = parseInt(sidebar.style.width, 10) || MIN_WIDTH;
            applyCollapseState(true);
        }

        // Listen for breakpoint changes to swap state
        var mql = window.matchMedia(MOBILE_BREAKPOINT);
        mql.addEventListener('change', function() {
            var newState = loadCollapseState();
            applyCollapseState(newState);
        });
    }

    function onMouseDown(e) {
        if (sidebarCollapsed) return;
        e.preventDefault();
        startDrag(e.clientX);
        document.addEventListener('mousemove', onMouseMove);
        document.addEventListener('mouseup', onMouseUp);
    }

    function onMouseMove(e) {
        doDrag(e.clientX);
    }

    function onMouseUp() {
        endDrag();
        document.removeEventListener('mousemove', onMouseMove);
        document.removeEventListener('mouseup', onMouseUp);
    }

    function onTouchStart(e) {
        if (sidebarCollapsed) return;
        if (e.touches.length !== 1) return;
        e.preventDefault();
        startDrag(e.touches[0].clientX);
        document.addEventListener('touchmove', onTouchMove, { passive: false });
        document.addEventListener('touchend', onTouchEnd);
    }

    function onTouchMove(e) {
        if (e.touches.length !== 1) return;
        e.preventDefault();
        doDrag(e.touches[0].clientX);
    }

    function onTouchEnd() {
        endDrag();
        document.removeEventListener('touchmove', onTouchMove);
        document.removeEventListener('touchend', onTouchEnd);
    }

    function startDrag(clientX) {
        startX = clientX;
        startWidth = sidebar.offsetWidth;
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    }

    function doDrag(clientX) {
        var delta = clientX - startX;
        var newWidth;

        if (position === 'right') {
            newWidth = startWidth - delta;
        } else {
            newWidth = startWidth + delta;
        }

        newWidth = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, newWidth));
        sidebar.style.width = newWidth + 'px';
    }

    function endDrag() {
        document.body.style.cursor = '';
        document.body.style.userSelect = '';

        var finalWidth = sidebar.offsetWidth;
        expandedWidth = finalWidth;

        // Debounced persist
        clearTimeout(persistTimer);
        persistTimer = setTimeout(function() {
            fetch('/api/settings', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ sidebar_width: finalWidth })
            }).catch(function(err) {
                console.error('Failed to persist sidebar width:', err);
            });
        }, DEBOUNCE_MS);
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // --- Public API ---
    window.ClawIDESidebar = {
        toggleCollapse: toggleSidebarCollapse,
        isCollapsed: function() { return sidebarCollapsed; }
    };
})();
