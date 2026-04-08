// ClawIDE Merge Review — Side-by-side diff viewer for feature branches
(function() {
    'use strict';

    // --- State ---
    var projectID = '';
    var featureID = '';
    var baseURL = '';
    var changedFiles = [];
    var selectedFilePath = '';
    var currentMergeView = null;
    var annotations = [];
    var mainBranch = '';
    var featureBranch = '';
    var stats = null;
    var initialized = false;
    var annotationPollTimer = null;

    // --- Init ---
    function init(pid, fid) {
        if (initialized && projectID === pid && featureID === fid) {
            return; // Already initialized for this feature
        }
        projectID = pid;
        featureID = fid;
        baseURL = '/projects/' + projectID + '/features/' + featureID;
        initialized = true;
        changedFiles = [];
        selectedFilePath = '';
        annotations = [];
        mainBranch = '';
        featureBranch = '';
        stats = null;
        destroyCurrentMergeView();
        fetchChangedFiles();
    }

    // --- Fetch changed files ---
    function fetchChangedFiles() {
        var listEl = document.getElementById('review-file-list');
        var statsEl = document.getElementById('review-stats');
        var diffEl = document.getElementById('review-diff-container');

        if (listEl) listEl.innerHTML = '<div class="text-th-text-faint text-xs p-2">Loading...</div>';

        fetch(baseURL + '/api/review/files')
            .then(function(r) { return r.json(); })
            .then(function(data) {
                changedFiles = data.files || [];
                stats = data.stats;
                mainBranch = data.main_branch;
                featureBranch = data.feature_branch;

                renderStats(statsEl);
                renderFileList(listEl);

                if (changedFiles.length === 0) {
                    if (diffEl) {
                        diffEl.innerHTML = '<div class="flex items-center justify-center h-full text-th-text-faint"><div class="text-center"><svg class="w-12 h-12 mx-auto mb-3 text-th-text-ghost" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M5 13l4 4L19 7"/></svg><p class="text-sm">No changes to review</p></div></div>';
                    }
                } else {
                    // Auto-select first file
                    loadDiff(changedFiles[0].path);
                }
            })
            .catch(function(err) {
                if (listEl) listEl.innerHTML = '<div class="text-red-400 text-xs p-2">Failed to load files</div>';
                console.error('Failed to fetch review files:', err);
            });
    }

    // --- Render stats ---
    function renderStats(container) {
        if (!container || !stats) return;
        container.innerHTML = '<span class="text-th-text-muted text-xs">' +
            stats.files_changed + ' file' + (stats.files_changed !== 1 ? 's' : '') + ' changed' +
            (stats.insertions ? ', <span class="text-green-400">+' + stats.insertions + '</span>' : '') +
            (stats.deletions ? ', <span class="text-red-400">-' + stats.deletions + '</span>' : '') +
            '</span>';
    }

    // --- Render file list ---
    function renderFileList(container) {
        if (!container) return;

        if (changedFiles.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs p-2">No changes</div>';
            return;
        }

        var html = '';
        changedFiles.forEach(function(f) {
            var statusClass = '';
            var statusLabel = f.status;
            switch (f.status) {
                case 'M': statusClass = 'bg-yellow-600'; statusLabel = 'M'; break;
                case 'A': statusClass = 'bg-green-600'; statusLabel = 'A'; break;
                case 'D': statusClass = 'bg-red-600'; statusLabel = 'D'; break;
                case 'R': statusClass = 'bg-purple-600'; statusLabel = 'R'; break;
                case 'C': statusClass = 'bg-blue-600'; statusLabel = 'C'; break;
                default: statusClass = 'bg-th-border-muted'; break;
            }

            var isSelected = f.path === selectedFilePath;
            var fname = f.path.split('/').pop();
            var dir = f.path.substring(0, f.path.length - fname.length);

            html += '<div class="review-file-item flex items-center gap-2 px-3 py-1.5 cursor-pointer text-xs transition-colors ' +
                (isSelected ? 'bg-surface-overlay text-th-text-primary' : 'text-th-text-tertiary hover:bg-surface-raised') +
                '" data-path="' + f.path + '" onclick="ClawIDEMergeReview.loadDiff(\'' + f.path.replace(/'/g, "\\'") + '\')">' +
                '<span class="px-1 py-0.5 text-[10px] font-mono font-semibold rounded text-th-text-primary ' + statusClass + '">' + statusLabel + '</span>' +
                '<span class="truncate"><span class="text-th-text-faint">' + dir + '</span>' + fname + '</span>' +
                '</div>';
        });

        container.innerHTML = html;
    }

    // --- Load diff for a file ---
    function loadDiff(filePath) {
        selectedFilePath = filePath;

        // Update file list selection
        var items = document.querySelectorAll('.review-file-item');
        items.forEach(function(el) {
            if (el.getAttribute('data-path') === filePath) {
                el.classList.add('bg-surface-overlay', 'text-th-text-primary');
                el.classList.remove('text-th-text-tertiary', 'hover:bg-surface-raised');
            } else {
                el.classList.remove('bg-surface-overlay', 'text-th-text-primary');
                el.classList.add('text-th-text-tertiary', 'hover:bg-surface-raised');
            }
        });

        var diffEl = document.getElementById('review-diff-container');
        if (!diffEl) return;

        diffEl.innerHTML = '<div class="flex items-center justify-center h-full text-th-text-faint text-sm">Loading diff...</div>';

        // Find the file entry to handle renames
        var fileEntry = null;
        for (var i = 0; i < changedFiles.length; i++) {
            if (changedFiles[i].path === filePath) {
                fileEntry = changedFiles[i];
                break;
            }
        }

        // Determine which paths to fetch from each branch
        var featurePath = filePath;
        var mainPath = filePath;
        if (fileEntry && fileEntry.old_path) {
            mainPath = fileEntry.old_path; // For renames, fetch old path from main
        }

        // Fetch both versions in parallel
        var featurePromise = fetchFileContent(featurePath, 'feature');
        var mainPromise = fetchFileContent(mainPath, 'main');

        Promise.all([featurePromise, mainPromise])
            .then(function(results) {
                var featureContent = results[0];
                var mainContent = results[1];

                // Handle binary files
                if (featureContent === null || mainContent === null) {
                    diffEl.innerHTML = '<div class="flex items-center justify-center h-full text-th-text-faint"><div class="text-center"><svg class="w-10 h-10 mx-auto mb-2 text-th-text-ghost" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg><p class="text-sm">Binary file — cannot display diff</p></div></div>';
                    return;
                }

                destroyCurrentMergeView();
                diffEl.innerHTML = '';

                if (typeof ClawIDECodeMirror !== 'undefined' && ClawIDECodeMirror.createMergeView) {
                    ClawIDECodeMirror.createMergeView(diffEl, featureContent, mainContent, filePath)
                        .then(function(mv) {
                            currentMergeView = mv;
                        });
                } else {
                    diffEl.innerHTML = '<div class="p-4 text-red-400 text-sm">CodeMirror MergeView not available</div>';
                }
            })
            .catch(function(err) {
                diffEl.innerHTML = '<div class="p-4 text-red-400 text-sm">Failed to load diff: ' + err.message + '</div>';
                console.error('Failed to load diff:', err);
            });
    }

    // --- Fetch file content ---
    function fetchFileContent(filePath, ref) {
        return fetch(baseURL + '/api/review/file-content?path=' + encodeURIComponent(filePath) + '&ref=' + ref)
            .then(function(r) {
                if (r.status === 409) {
                    // Binary file
                    return null;
                }
                if (r.headers.get('X-File-Status') === 'not-found') {
                    return ''; // File doesn't exist at this ref
                }
                if (!r.ok) {
                    throw new Error('HTTP ' + r.status);
                }
                return r.text();
            });
    }

    // --- AI Review ---
    function startAIReview() {
        var termEl = document.getElementById('review-ai-terminal');
        var annotListEl = document.getElementById('review-annotations-list');

        if (!termEl) return;

        // Check if AI review command is configured
        var aiBtn = document.getElementById('review-ai-btn');
        if (aiBtn && aiBtn.dataset.command === '') {
            if (typeof ClawIDEToast !== 'undefined') {
                ClawIDEToast.show('AI review command not configured. Set it in Settings.', 'warning');
            }
            return;
        }

        var command = (aiBtn && aiBtn.dataset.command) || '';
        if (!command) return;

        // Replace placeholders
        command = command.replace(/\{MAIN_BRANCH\}/g, mainBranch);
        command = command.replace(/\{FEATURE_BRANCH\}/g, featureBranch);
        command = command.replace(/\{DIFF_RANGE\}/g, mainBranch + '...' + featureBranch);

        termEl.innerHTML = '<div class="p-3 font-mono text-xs text-th-text-muted"><div class="text-green-400 mb-1">$ ' + escapeHtml(command) + '</div><div class="text-th-text-faint">Running AI review...</div></div>';

        // Show the bottom panel
        var bottomPanel = document.getElementById('review-bottom-panel');
        if (bottomPanel) bottomPanel.classList.remove('hidden');

        // Start polling for annotations
        if (annotationPollTimer) clearInterval(annotationPollTimer);
        annotationPollTimer = setInterval(function() {
            fetchAnnotations();
        }, 3000);
    }

    // --- Fetch annotations ---
    function fetchAnnotations() {
        fetch(baseURL + '/api/review/annotations')
            .then(function(r) { return r.json(); })
            .then(function(data) {
                annotations = data.annotations || [];
                if (data.status === 'complete' || data.status === 'error') {
                    if (annotationPollTimer) {
                        clearInterval(annotationPollTimer);
                        annotationPollTimer = null;
                    }
                }
                renderAnnotationsList();
            })
            .catch(function(err) {
                console.error('Failed to fetch annotations:', err);
            });
    }

    // --- Render annotations ---
    function renderAnnotationsList() {
        var container = document.getElementById('review-annotations-list');
        if (!container) return;

        if (annotations.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs p-3">No annotations yet</div>';
            return;
        }

        var html = '';
        annotations.forEach(function(a) {
            var severityClass = 'border-th-border-strong';
            var severityBadge = 'bg-th-border-muted';
            switch (a.severity) {
                case 'error': severityClass = 'border-red-800'; severityBadge = 'bg-red-600'; break;
                case 'warning': severityClass = 'border-yellow-800'; severityBadge = 'bg-yellow-600'; break;
                case 'info': severityClass = 'border-blue-800'; severityBadge = 'bg-blue-600'; break;
                case 'suggestion': severityClass = 'border-green-800'; severityBadge = 'bg-green-600'; break;
            }

            html += '<div class="annotation-card p-2 mb-1 rounded border ' + severityClass + ' cursor-pointer hover:bg-surface-raised/50 transition-colors" onclick="ClawIDEMergeReview.loadDiff(\'' + a.file.replace(/'/g, "\\'") + '\')">' +
                '<div class="flex items-center gap-2 mb-1">' +
                '<span class="px-1.5 py-0.5 text-[10px] font-semibold rounded text-th-text-primary ' + severityBadge + '">' + (a.severity || 'info') + '</span>' +
                '<span class="text-[10px] text-th-text-faint truncate">' + a.file + ':' + a.line + (a.end_line ? '-' + a.end_line : '') + '</span>' +
                '</div>' +
                '<p class="text-xs text-th-text-tertiary">' + escapeHtml(a.comment) + '</p>' +
                '</div>';
        });

        container.innerHTML = html;
    }

    // --- Merge actions ---
    function doMerge() {
        if (!confirm('Merge ' + featureBranch + ' into ' + mainBranch + '?')) return;

        fetch(baseURL + '/api/merge', { method: 'POST' })
            .then(function(r) {
                if (r.ok || r.redirected) {
                    return r.json();
                }
                return r.text().then(function(t) { throw new Error(t); });
            })
            .then(function(data) {
                if (data && data.redirect) {
                    window.location.href = data.redirect;
                } else {
                    window.location.href = '/projects/' + projectID + '/';
                }
            })
            .catch(function(err) {
                if (typeof ClawIDEToast !== 'undefined') {
                    ClawIDEToast.show('Merge failed: ' + err.message, 'error');
                } else {
                    alert('Merge failed: ' + err.message);
                }
            });
    }

    function doQuickMerge() {
        if (!confirm('Quick merge ' + featureBranch + ' into the main branch? This skips the review.')) return;

        fetch(baseURL + '/api/merge', { method: 'POST' })
            .then(function(r) {
                if (r.ok || r.redirected) {
                    return r.json();
                }
                return r.text().then(function(t) { throw new Error(t); });
            })
            .then(function(data) {
                if (data && data.redirect) {
                    window.location.href = data.redirect;
                } else {
                    window.location.href = '/projects/' + projectID + '/';
                }
            })
            .catch(function(err) {
                if (typeof ClawIDEToast !== 'undefined') {
                    ClawIDEToast.show('Merge failed: ' + err.message, 'error');
                } else {
                    alert('Merge failed: ' + err.message);
                }
            });
    }

    // --- Cleanup ---
    function destroy() {
        destroyCurrentMergeView();
        if (annotationPollTimer) {
            clearInterval(annotationPollTimer);
            annotationPollTimer = null;
        }
        initialized = false;
        changedFiles = [];
        selectedFilePath = '';
        annotations = [];
    }

    function destroyCurrentMergeView() {
        if (currentMergeView && typeof ClawIDECodeMirror !== 'undefined') {
            ClawIDECodeMirror.destroyMergeView(currentMergeView);
            currentMergeView = null;
        }
    }

    // --- Util ---
    function escapeHtml(text) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(text));
        return div.innerHTML;
    }

    // --- Expose ---
    window.ClawIDEMergeReview = {
        init: init,
        loadDiff: loadDiff,
        startAIReview: startAIReview,
        doMerge: doMerge,
        doQuickMerge: doQuickMerge,
        destroy: destroy,
    };
})();
