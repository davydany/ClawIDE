// Update check manager for ClawIDE
function updateCheck() {
    return {
        open: false,
        checking: false,
        installing: false,
        updateAvailable: false,
        currentVersion: '',
        latestVersion: '',
        checkIntervalId: null,
        CHECK_INTERVAL_MS: 5 * 60 * 1000, // 5 minutes

        async init() {
            // Fetch initial status
            await this.fetchStatus();

            // Set up periodic check every 5 minutes
            this.checkIntervalId = setInterval(() => this.fetchStatus(), this.CHECK_INTERVAL_MS);
        },

        async fetchStatus() {
            try {
                const response = await fetch('/api/update/status');
                if (!response.ok) {
                    console.error('Update status check failed:', response.statusText);
                    return;
                }

                const data = await response.json();
                this.currentVersion = data.current_version || '';
                this.latestVersion = data.latest_version || '';
                this.updateAvailable = data.update_available || false;
            } catch (error) {
                console.error('Error fetching update status:', error);
            }
        },

        async checkNow() {
            this.checking = true;
            try {
                const response = await fetch('/api/update/check', { method: 'POST' });
                if (!response.ok) {
                    console.error('Update check failed:', response.statusText);
                    return;
                }

                const data = await response.json();
                this.currentVersion = data.current_version || '';
                this.latestVersion = data.latest_version || '';
                this.updateAvailable = data.update_available || false;
            } catch (error) {
                console.error('Error checking for updates:', error);
            } finally {
                this.checking = false;
            }
        },

        async applyUpdate() {
            if (!this.updateAvailable) {
                alert('No update available');
                return;
            }

            this.installing = true;
            try {
                const response = await fetch('/api/update/apply', { method: 'POST' });
                if (!response.ok) {
                    const data = await response.json();
                    alert('Update failed: ' + (data.message || response.statusText));
                    return;
                }

                // Show success message
                const data = await response.json();
                alert(data.message || 'Update started. ClawIDE will restart automatically.');

                // Close the popover
                this.open = false;
            } catch (error) {
                console.error('Error applying update:', error);
                alert('Update failed: ' + error.message);
            } finally {
                this.installing = false;
            }
        },

        toggle() {
            this.open = !this.open;
        },

        destroy() {
            if (this.checkIntervalId) {
                clearInterval(this.checkIntervalId);
                this.checkIntervalId = null;
            }
        }
    };
}
