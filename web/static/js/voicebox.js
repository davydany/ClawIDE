// ClawIDE Voice Box
// Modal-based text input for voice dictation on mobile, with history.
(function() {
    'use strict';

    var API_BASE = '/api/voicebox';
    var history = [];

    // DOM references
    var modal, textarea, historyList, historySection;

    function init() {
        modal = document.getElementById('voicebox-modal');
        textarea = document.getElementById('voicebox-textarea');
        historyList = document.getElementById('voicebox-history-list');
        historySection = document.getElementById('voicebox-history-section');

        if (!modal) return;

        // Close on backdrop click
        modal.addEventListener('click', function(e) {
            if (e.target === modal) close();
        });

        // Keyboard shortcut: Cmd/Ctrl+Enter to send
        if (textarea) {
            textarea.addEventListener('keydown', function(e) {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                    e.preventDefault();
                    handleSend();
                }
                // Escape to close
                if (e.key === 'Escape') {
                    e.preventDefault();
                    close();
                }
            });
        }
    }

    function open() {
        if (!modal) return;
        modal.classList.remove('hidden');
        if (textarea) {
            textarea.value = '';
            textarea.focus();
        }
        loadHistory();
    }

    function close() {
        if (!modal) return;
        modal.classList.add('hidden');
        if (textarea) textarea.value = '';
    }

    function handleSend() {
        if (!textarea) return;
        var content = textarea.value.trim();
        if (!content) return;

        // Send to active terminal
        sendToTerminal(content);

        // Copy to clipboard
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(content).catch(function(err) {
                console.error('Clipboard copy failed:', err);
            });
        }

        // Save to history
        fetch(API_BASE, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ content: content })
        })
        .then(function(r) {
            if (!r.ok) throw new Error('Save failed');
            return r.json();
        })
        .then(function() {
            loadHistory();
        })
        .catch(function(err) {
            console.error('Failed to save voicebox entry:', err);
        });

        close();
    }

    function sendToTerminal(content) {
        var paneID = window.ClawIDETerminal.getFocusedPaneID();
        if (!paneID) {
            var allPanes = window.ClawIDETerminal.getAllPaneIDs();
            if (allPanes.length === 0) return;
            paneID = allPanes[0];
        }
        window.ClawIDETerminal.sendInput(paneID, content);
    }

    function loadHistory() {
        fetch(API_BASE)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                history = data || [];
                renderHistory();
            })
            .catch(function(err) {
                console.error('Failed to load voicebox history:', err);
            });
    }

    function renderHistory() {
        if (!historyList) return;

        if (history.length === 0) {
            historyList.innerHTML = '<div class="text-gray-500 text-xs p-3 text-center">No history yet</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < history.length; i++) {
            var entry = history[i];
            var preview = entry.content.length > 80 ? entry.content.substring(0, 80) + '...' : entry.content;
            var timeAgo = formatTimeAgo(entry.created_at);

            html += '<div class="voicebox-history-item" data-id="' + entry.id + '">';
            html += '  <div class="flex items-center justify-between gap-2">';
            html += '    <span class="text-[10px] text-gray-500 flex-shrink-0">' + timeAgo + '</span>';
            html += '    <div class="flex items-center gap-0.5 flex-shrink-0">';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-green-400 transition-colors" title="Re-send to terminal" data-resend="' + entry.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 10 4 15 9 20"/><path d="M20 4v7a4 4 0 01-4 4H4"/></svg>';
            html += '      </button>';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-blue-400 transition-colors" title="Copy to clipboard" data-copy="' + entry.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>';
            html += '      </button>';
            html += '      <button class="p-0.5 rounded text-gray-500 hover:text-red-400 transition-colors" title="Delete" data-delete="' + entry.id + '">';
            html += '        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>';
            html += '      </button>';
            html += '    </div>';
            html += '  </div>';
            html += '  <div class="text-xs text-gray-300 mt-0.5 font-mono break-all">' + escapeHTML(preview) + '</div>';
            html += '</div>';
        }
        historyList.innerHTML = html;

        // Bind action buttons
        var resendBtns = historyList.querySelectorAll('[data-resend]');
        for (var j = 0; j < resendBtns.length; j++) {
            resendBtns[j].addEventListener('click', function() {
                resendEntry(this.getAttribute('data-resend'));
            });
        }

        var copyBtns = historyList.querySelectorAll('[data-copy]');
        for (var k = 0; k < copyBtns.length; k++) {
            copyBtns[k].addEventListener('click', function() {
                copyEntry(this.getAttribute('data-copy'));
            });
        }

        var deleteBtns = historyList.querySelectorAll('[data-delete]');
        for (var l = 0; l < deleteBtns.length; l++) {
            deleteBtns[l].addEventListener('click', function() {
                deleteEntry(this.getAttribute('data-delete'));
            });
        }
    }

    function resendEntry(id) {
        var entry = findEntry(id);
        if (!entry) return;
        sendToTerminal(entry.content);
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(entry.content).catch(function() {});
        }
    }

    function copyEntry(id) {
        var entry = findEntry(id);
        if (!entry) return;
        if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(entry.content).catch(function(err) {
                console.error('Copy failed:', err);
            });
        }
    }

    function deleteEntry(id) {
        fetch(API_BASE + '/' + id, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) throw new Error('Delete failed');
                loadHistory();
            })
            .catch(function(err) {
                console.error('Failed to delete voicebox entry:', err);
            });
    }

    function handleClearHistory() {
        if (!confirm('Clear all voice box history?')) return;
        fetch(API_BASE, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) throw new Error('Clear failed');
                loadHistory();
            })
            .catch(function(err) {
                console.error('Failed to clear voicebox history:', err);
            });
    }

    function findEntry(id) {
        for (var i = 0; i < history.length; i++) {
            if (history[i].id === id) return history[i];
        }
        return null;
    }

    function formatTimeAgo(dateStr) {
        var date = new Date(dateStr);
        var now = new Date();
        var diff = Math.floor((now - date) / 1000);
        if (diff < 60) return 'just now';
        if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
        if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
        return Math.floor(diff / 86400) + 'd ago';
    }

    function escapeHTML(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Expose for external use
    window.ClawIDEVoiceBox = {
        open: open,
        close: close,
        send: handleSend,
        reload: loadHistory,
        clearHistory: handleClearHistory
    };
})();
