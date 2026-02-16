// ClawIDE Scratchpad
// Single persistent textarea with auto-save on blur.
(function() {
    'use strict';

    var API_BASE = '/api/scratchpad';
    var saveTimer = null;
    var textarea;

    function init() {
        textarea = document.getElementById('scratchpad-content');
        if (!textarea) return;

        textarea.addEventListener('blur', function() {
            debouncedSave();
        });

        loadContent();
    }

    function loadContent() {
        fetch(API_BASE)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                if (textarea && data.content !== undefined) {
                    textarea.value = data.content;
                }
            })
            .catch(function(err) {
                console.error('Failed to load scratchpad:', err);
            });
    }

    function saveContent() {
        if (!textarea) return;

        fetch(API_BASE, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content: textarea.value })
        })
        .catch(function(err) {
            console.error('Failed to save scratchpad:', err);
        });
    }

    function debouncedSave() {
        clearTimeout(saveTimer);
        saveTimer = setTimeout(saveContent, 500);
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose for external use
    window.ClawIDEScratchpad = {
        reload: loadContent
    };
})();
