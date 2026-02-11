// ClawIDE main application JS

document.addEventListener('DOMContentLoaded', function() {
    // Register htmx event handlers
    document.body.addEventListener('htmx:afterSwap', function(evt) {
        // Re-initialize any components after htmx swaps
    });

    document.body.addEventListener('htmx:responseError', function(evt) {
        console.error('HTMX request failed:', evt.detail);
    });
});
