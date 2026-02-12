// ClawIDE Sidebar Resize
// Drag-to-resize on the sidebar edge handle, clamp width 200-500px,
// debounced persist via PUT /api/settings.
(function() {
    'use strict';

    var MIN_WIDTH = 200;
    var MAX_WIDTH = 500;
    var DEBOUNCE_MS = 300;

    var handle, sidebar, position, startX, startWidth, persistTimer;

    function init() {
        handle = document.getElementById('sidebar-resize-handle');
        sidebar = document.getElementById('app-sidebar');
        if (!handle || !sidebar) return;

        position = handle.getAttribute('data-position') || 'left';

        handle.addEventListener('mousedown', onMouseDown);
        handle.addEventListener('touchstart', onTouchStart, { passive: false });
    }

    function onMouseDown(e) {
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
})();
