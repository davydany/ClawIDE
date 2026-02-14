// ClawIDE Toast Notifications
(function() {
    'use strict';

    function showToast(message, duration = 1000) {
        const toast = document.createElement('div');
        toast.className = 'toast';
        toast.setAttribute('role', 'status');
        toast.setAttribute('aria-live', 'polite');
        toast.textContent = message;

        document.body.appendChild(toast);

        // Trigger animation by adding active class
        requestAnimationFrame(() => {
            toast.classList.add('active');
        });

        // Remove after duration
        setTimeout(() => {
            toast.classList.remove('active');
            setTimeout(() => {
                if (toast.parentNode) {
                    document.body.removeChild(toast);
                }
            }, 300); // Wait for fade out animation
        }, duration);
    }

    // Expose globally
    window.ClawIDEToast = {
        show: showToast,
    };
})();
