// ClawIDE Git Integration Module
// Provides git status badges, refresh, commit modal, and edge case handling
// for .clawide/notes/ and .clawide/bookmarks/ directories.
(function() {
    'use strict';

    var NOTES_STATUS_URL = '/api/notes/git-status';
    var NOTES_COMMIT_URL = '/api/notes/commit';
    var BOOKMARKS_STATUS_URL = '/api/bookmarks/git-status';
    var BOOKMARKS_COMMIT_URL = '/api/bookmarks/commit';

    // Cached status per type
    var statusCache = { notes: null, bookmarks: null };

    // --- Status Badge Rendering ---

    var badgeColors = {
        'M': { bg: 'bg-amber-500/20', text: 'text-amber-400', label: 'Modified' },
        'A': { bg: 'bg-green-500/20', text: 'text-green-400', label: 'Added' },
        '?': { bg: 'bg-blue-500/20', text: 'text-blue-400', label: 'Untracked' },
        'D': { bg: 'bg-red-500/20', text: 'text-red-400', label: 'Deleted' },
        'R': { bg: 'bg-purple-500/20', text: 'text-purple-400', label: 'Renamed' },
        'U': { bg: 'bg-red-600/30', text: 'text-red-300', label: 'Conflict' }
    };

    function renderBadge(status) {
        var colors = badgeColors[status] || { bg: 'bg-th-text-faint/20', text: 'text-th-text-muted', label: status };
        return '<span class="inline-flex items-center px-1 py-0.5 rounded text-[9px] font-mono font-bold ' +
            colors.bg + ' ' + colors.text + '" title="' + colors.label + '">' + escapeHTML(status) + '</span>';
    }

    // --- Fetch Git Status ---

    function fetchStatus(type, projectID) {
        if (!projectID) return Promise.resolve(null);

        var url = (type === 'notes' ? NOTES_STATUS_URL : BOOKMARKS_STATUS_URL) +
            '?project_id=' + encodeURIComponent(projectID);

        return fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                statusCache[type] = data;
                return data;
            })
            .catch(function(err) {
                console.error('Failed to fetch git status for ' + type + ':', err);
                return null;
            });
    }

    // --- Get File Status Badge ---
    // Returns the status character for a file path, or null if clean.

    function getFileStatus(type, filePath) {
        var status = statusCache[type];
        if (!status || !status.files) return null;

        for (var i = 0; i < status.files.length; i++) {
            if (status.files[i].path === filePath) {
                return status.files[i].status;
            }
        }
        return null;
    }

    // Returns change count for a given type
    function getChangeCount(type) {
        var status = statusCache[type];
        if (!status || !status.files) return 0;
        // Count unique paths (a file may appear twice: staged + unstaged)
        var paths = {};
        for (var i = 0; i < status.files.length; i++) {
            paths[status.files[i].path] = true;
        }
        return Object.keys(paths).length;
    }

    // --- Warning Banner ---

    function renderWarningBanner(type) {
        var status = statusCache[type];
        if (!status) return '';

        if (!status.is_git_repo) {
            return '<div class="flex items-center gap-2 px-3 py-1.5 bg-surface-overlay/50 rounded text-[10px] text-th-text-muted">' +
                '<svg class="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>' +
                'Not a git repository &mdash; git features disabled</div>';
        }

        if (status.is_ignored) {
            return '<div class="flex items-center gap-2 px-3 py-1.5 bg-amber-900/30 border border-amber-700/30 rounded text-[10px] text-amber-300">' +
                '<svg class="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                '.clawide is in .gitignore &mdash; changes won\'t be tracked</div>';
        }

        if (status.has_conflict) {
            return '<div class="flex items-center gap-2 px-3 py-1.5 bg-red-900/30 border border-red-700/30 rounded text-[10px] text-red-300">' +
                '<svg class="w-3 h-3 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                'Merge conflict detected &mdash; resolve conflicts before committing</div>';
        }

        return '';
    }

    // --- Refresh Button HTML ---

    function renderRefreshButton(type) {
        return '<button class="p-1 rounded text-th-text-faint hover:text-th-text-primary transition-colors" title="Refresh git status" data-git-refresh="' + type + '">' +
            '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"/></svg>' +
            '</button>';
    }

    // --- Commit Modal ---

    var commitModalEl = null;

    function showCommitModal(type, projectID) {
        var status = statusCache[type];
        if (!status || !status.files || status.files.length === 0) {
            alert('No changes to commit');
            return;
        }

        // Deduplicate files (may appear as both staged and unstaged)
        var fileMap = {};
        for (var i = 0; i < status.files.length; i++) {
            var f = status.files[i];
            if (!fileMap[f.path]) {
                fileMap[f.path] = f.status;
            }
        }

        var filePaths = Object.keys(fileMap);

        // Create modal
        if (commitModalEl) {
            commitModalEl.remove();
        }

        commitModalEl = document.createElement('div');
        commitModalEl.className = 'fixed inset-0 z-50 flex items-center justify-center bg-black/60';
        commitModalEl.setAttribute('data-git-commit-modal', '');

        var typeLabel = type === 'notes' ? 'Notes' : 'Bookmarks';
        var html = '';
        html += '<div class="bg-surface-raised border border-th-border-strong rounded-lg shadow-xl w-full max-w-md mx-4">';
        html += '  <div class="flex items-center justify-between px-4 py-3 border-b border-th-border-strong">';
        html += '    <h3 class="text-sm font-medium text-th-text-primary">Commit ' + typeLabel + ' Changes</h3>';
        html += '    <button class="p-1 rounded text-th-text-muted hover:text-th-text-primary transition-colors" data-git-commit-close>';
        html += '      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>';
        html += '    </button>';
        html += '  </div>';

        // File list with checkboxes
        html += '  <div class="px-4 py-3 max-h-48 overflow-y-auto">';
        html += '    <div class="flex items-center gap-2 mb-2">';
        html += '      <input type="checkbox" id="git-select-all" checked class="rounded border-th-border-muted bg-surface-overlay text-accent focus:ring-accent-border">';
        html += '      <label for="git-select-all" class="text-xs text-th-text-muted">Select all</label>';
        html += '    </div>';
        for (var j = 0; j < filePaths.length; j++) {
            var path = filePaths[j];
            var st = fileMap[path];
            var shortPath = path.replace(/^\.clawide\/(notes|bookmarks)\//, '');
            html += '    <label class="flex items-center gap-2 py-1 cursor-pointer hover:bg-surface-overlay/50 rounded px-1">';
            html += '      <input type="checkbox" checked class="rounded border-th-border-muted bg-surface-overlay text-accent focus:ring-accent-border" value="' + escapeHTML(path) + '" data-git-file>';
            html += '      ' + renderBadge(st);
            html += '      <span class="text-xs text-th-text-tertiary truncate" title="' + escapeHTML(path) + '">' + escapeHTML(shortPath) + '</span>';
            html += '    </label>';
        }
        html += '  </div>';

        // Commit message
        html += '  <div class="px-4 py-3 border-t border-th-border-strong">';
        html += '    <input type="text" id="git-commit-message" placeholder="Commit message..." class="w-full px-3 py-2 text-xs bg-surface-base border border-th-border-muted rounded text-th-text-primary placeholder-th-text-faint focus:border-accent-border focus:ring-1 focus:ring-accent-border outline-none">';
        html += '  </div>';

        // Actions
        html += '  <div class="flex items-center justify-end gap-2 px-4 py-3 border-t border-th-border-strong">';
        html += '    <button class="px-3 py-1.5 text-xs text-th-text-muted hover:text-th-text-primary transition-colors rounded" data-git-commit-close>Cancel</button>';
        html += '    <button class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded transition-colors" data-git-commit-submit>Commit</button>';
        html += '  </div>';
        html += '</div>';

        commitModalEl.innerHTML = html;
        document.body.appendChild(commitModalEl);

        // Bind events
        var closeButtons = commitModalEl.querySelectorAll('[data-git-commit-close]');
        for (var c = 0; c < closeButtons.length; c++) {
            closeButtons[c].addEventListener('click', closeCommitModal);
        }

        // Click backdrop to close
        commitModalEl.addEventListener('click', function(e) {
            if (e.target === commitModalEl) closeCommitModal();
        });

        // Escape to close
        document.addEventListener('keydown', commitModalEscHandler);

        // Select all toggle
        var selectAll = commitModalEl.querySelector('#git-select-all');
        if (selectAll) {
            selectAll.addEventListener('change', function() {
                var checkboxes = commitModalEl.querySelectorAll('[data-git-file]');
                for (var x = 0; x < checkboxes.length; x++) {
                    checkboxes[x].checked = selectAll.checked;
                }
            });
        }

        // Submit
        var submitBtn = commitModalEl.querySelector('[data-git-commit-submit]');
        if (submitBtn) {
            submitBtn.addEventListener('click', function() {
                doCommit(type, projectID);
            });
        }

        // Enter key in message input
        var msgInput = commitModalEl.querySelector('#git-commit-message');
        if (msgInput) {
            msgInput.focus();
            msgInput.addEventListener('keydown', function(e) {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    doCommit(type, projectID);
                }
            });
        }
    }

    function commitModalEscHandler(e) {
        if (e.key === 'Escape') closeCommitModal();
    }

    function closeCommitModal() {
        if (commitModalEl) {
            commitModalEl.remove();
            commitModalEl = null;
        }
        document.removeEventListener('keydown', commitModalEscHandler);
    }

    function doCommit(type, projectID) {
        if (!commitModalEl) return;

        var msgInput = commitModalEl.querySelector('#git-commit-message');
        var message = msgInput ? msgInput.value.trim() : '';
        if (!message) {
            msgInput.classList.add('border-red-500');
            msgInput.focus();
            return;
        }

        var checkboxes = commitModalEl.querySelectorAll('[data-git-file]:checked');
        var files = [];
        for (var i = 0; i < checkboxes.length; i++) {
            files.push(checkboxes[i].value);
        }
        if (files.length === 0) {
            alert('No files selected');
            return;
        }

        var commitUrl = type === 'notes' ? NOTES_COMMIT_URL : BOOKMARKS_COMMIT_URL;

        // Disable submit button
        var submitBtn = commitModalEl.querySelector('[data-git-commit-submit]');
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.textContent = 'Committing...';
        }

        fetch(commitUrl, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                project_id: projectID,
                files: files,
                message: message
            })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function(data) {
            closeCommitModal();
            // Show success toast if available
            if (window.ClawIDEToast) {
                var msg = 'Committed successfully';
                if (data.commit_hash) {
                    msg += ' (' + data.commit_hash + ')';
                }
                window.ClawIDEToast.show(msg, 'success');
            }
            // Refresh status
            fetchStatus(type, projectID);
        })
        .catch(function(err) {
            console.error('Commit failed:', err);
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.textContent = 'Commit';
            }
            alert('Commit failed: ' + err.message);
        });
    }

    // --- Commit Button HTML ---

    function renderCommitButton(type) {
        var status = statusCache[type];
        if (!status || !status.is_git_repo || status.is_ignored) return '';
        var count = getChangeCount(type);
        if (count === 0) return '';

        return '<button class="flex items-center gap-1 px-2 py-1 text-[10px] bg-accent/20 hover:bg-accent/40 text-accent-text rounded transition-colors" title="Commit changes" data-git-commit-open="' + type + '">' +
            '<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><circle cx="12" cy="12" r="4"/><line x1="1.05" y1="12" x2="7" y2="12"/><line x1="17.01" y1="12" x2="22.96" y2="12"/></svg>' +
            '<span>' + count + ' change' + (count !== 1 ? 's' : '') + '</span>' +
            '</button>';
    }

    // --- Global Click Delegation ---

    document.addEventListener('click', function(e) {
        // Refresh button clicks
        var refreshBtn = e.target.closest('[data-git-refresh]');
        if (refreshBtn) {
            var refreshType = refreshBtn.getAttribute('data-git-refresh');
            var pid = getProjectID();
            if (pid) {
                refreshBtn.classList.add('animate-spin');
                fetchStatus(refreshType, pid).then(function() {
                    refreshBtn.classList.remove('animate-spin');
                    // Trigger UI update
                    dispatchStatusUpdate(refreshType);
                });
            }
            return;
        }

        // Commit button clicks
        var commitBtn = e.target.closest('[data-git-commit-open]');
        if (commitBtn) {
            var commitType = commitBtn.getAttribute('data-git-commit-open');
            var cpid = getProjectID();
            if (cpid) {
                showCommitModal(commitType, cpid);
            }
            return;
        }
    });

    // --- Helpers ---

    function getProjectID() {
        // Try notes container first, then bookmarks
        var el = document.getElementById('notes-container') ||
                 document.getElementById('bookmarks-container');
        return el ? (el.getAttribute('data-project-id') || '') : '';
    }

    function dispatchStatusUpdate(type) {
        document.dispatchEvent(new CustomEvent('clawide-git-status-update', {
            detail: { type: type, status: statusCache[type] }
        }));
    }

    function escapeHTML(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // --- Public API ---

    window.ClawIDEGit = {
        fetchStatus: fetchStatus,
        getFileStatus: getFileStatus,
        getChangeCount: getChangeCount,
        getCachedStatus: function(type) { return statusCache[type]; },
        renderBadge: renderBadge,
        renderWarningBanner: renderWarningBanner,
        renderRefreshButton: renderRefreshButton,
        renderCommitButton: renderCommitButton,
        showCommitModal: showCommitModal
    };
})();
