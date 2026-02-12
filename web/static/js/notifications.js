// ClawIDE Notification Bell - Alpine.js data component
function notificationBell() {
    return {
        open: false,
        unreadCount: 0,
        notifications: [],
        eventSource: null,
        reconnectTimer: null,
        desktopPermission: 'default',

        init() {
            this.connectSSE();
            // Request desktop notification permission when first interaction happens
            if ('Notification' in window) {
                this.desktopPermission = Notification.permission;
            }
        },

        connectSSE() {
            if (this.eventSource) {
                this.eventSource.close();
            }

            this.eventSource = new EventSource('/api/notifications/stream');
            // Expose for beforeunload cleanup
            window._clawIDENotificationES = this.eventSource;

            this.eventSource.addEventListener('unread-count', (e) => {
                this.unreadCount = parseInt(e.data, 10) || 0;
            });

            this.eventSource.addEventListener('notification', (e) => {
                try {
                    var n = JSON.parse(e.data);
                    // Prepend to list
                    this.notifications.unshift(n);
                    // Keep max 50 in memory
                    if (this.notifications.length > 50) {
                        this.notifications = this.notifications.slice(0, 50);
                    }
                    this.unreadCount++;
                    this.showDesktopNotification(n);
                } catch (err) {
                    console.error('Failed to parse notification:', err);
                }
            });

            this.eventSource.onerror = () => {
                this.eventSource.close();
                this.eventSource = null;
                // Auto-reconnect after 3 seconds
                if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
                this.reconnectTimer = setTimeout(() => this.connectSSE(), 3000);
            };
        },

        toggle() {
            this.open = !this.open;
            if (this.open) {
                this.fetchNotifications();
                // Request desktop permission on first open
                if ('Notification' in window && this.desktopPermission === 'default') {
                    Notification.requestPermission().then((p) => {
                        this.desktopPermission = p;
                    });
                }
            }
        },

        fetchNotifications() {
            fetch('/api/notifications')
                .then(r => r.json())
                .then(data => {
                    this.notifications = data || [];
                })
                .catch(() => {});
        },

        markRead(id) {
            fetch('/api/notifications/' + id + '/read', { method: 'PATCH' })
                .then(() => {
                    var n = this.notifications.find(n => n.id === id);
                    if (n && !n.read) {
                        n.read = true;
                        this.unreadCount = Math.max(0, this.unreadCount - 1);
                    }
                })
                .catch(() => {});
        },

        markAllRead() {
            fetch('/api/notifications/read-all', { method: 'POST' })
                .then(() => {
                    this.notifications.forEach(n => n.read = true);
                    this.unreadCount = 0;
                })
                .catch(() => {});
        },

        navigate(n) {
            // Mark as read
            if (!n.read) {
                this.markRead(n.id);
            }
            this.open = false;

            // Build target URL with optional pane query param for deep-linking
            var url = '';
            if (n.feature_id && n.project_id) {
                url = '/projects/' + n.project_id + '/features/' + n.feature_id + '/';
            } else if (n.project_id) {
                url = '/projects/' + n.project_id + '/';
            }

            if (!url) return;

            if (n.pane_id) {
                url += '?pane=' + encodeURIComponent(n.pane_id);
            }

            // If already on the same workspace page, focus the pane directly
            if (n.pane_id && window.location.pathname === url.split('?')[0]) {
                window.ClawIDETerminal.focusPane(n.pane_id);
            } else {
                window.location.href = url;
            }
        },

        showDesktopNotification(n) {
            if (!('Notification' in window)) return;
            if (Notification.permission !== 'granted') return;
            if (document.hasFocus()) return; // Only show when tab is unfocused

            try {
                new Notification(n.title, {
                    body: n.body || '',
                    icon: '/static/vendor/alpine.min.js', // placeholder
                    tag: n.id,
                });
            } catch (e) {
                // Desktop notifications may not be available in all contexts
            }
        },

        timeAgo(dateStr) {
            if (!dateStr) return '';
            var date = new Date(dateStr);
            var now = new Date();
            var seconds = Math.floor((now - date) / 1000);

            if (seconds < 60) return 'just now';
            var minutes = Math.floor(seconds / 60);
            if (minutes < 60) return minutes + 'm ago';
            var hours = Math.floor(minutes / 60);
            if (hours < 24) return hours + 'h ago';
            var days = Math.floor(hours / 24);
            if (days < 7) return days + 'd ago';
            return date.toLocaleDateString();
        },

        destroy() {
            if (this.eventSource) {
                this.eventSource.close();
                this.eventSource = null;
            }
            if (this.reconnectTimer) {
                clearTimeout(this.reconnectTimer);
                this.reconnectTimer = null;
            }
        }
    };
}

// Close SSE connection before page navigation to free HTTP/1.1 connection slots.
window.addEventListener('beforeunload', function() {
    // Find the active EventSource via the Alpine component's shared reference.
    if (window._clawIDENotificationES) {
        window._clawIDENotificationES.close();
    }
});
