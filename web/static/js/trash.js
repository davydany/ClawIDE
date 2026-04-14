// ClawIDE Trash Bin — modal UI for viewing and restoring soft-deleted features.
(function() {
    'use strict';

    var DIALOG_STYLES = 'bg-gray-900 text-gray-100 rounded-xl shadow-2xl border border-gray-700 p-0 backdrop:bg-black/60';

    function escapeHTML(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    function daysRemaining(trashedAt) {
        var trashed = new Date(trashedAt);
        var expires = new Date(trashed.getTime() + 30 * 24 * 60 * 60 * 1000);
        var now = new Date();
        var diff = Math.ceil((expires - now) / (1000 * 60 * 60 * 24));
        return Math.max(0, diff);
    }

    function daysSince(trashedAt) {
        var trashed = new Date(trashedAt);
        var now = new Date();
        var diff = Math.floor((now - trashed) / (1000 * 60 * 60 * 24));
        if (diff === 0) return 'today';
        if (diff === 1) return '1 day ago';
        return diff + ' days ago';
    }

    function groupByProject(items) {
        var groups = {};
        for (var i = 0; i < items.length; i++) {
            var item = items[i];
            var key = item.project_name || 'Unknown Project';
            if (!groups[key]) groups[key] = [];
            groups[key].push(item);
        }
        return groups;
    }

    function renderItem(item, listEl, dialog) {
        var remaining = daysRemaining(item.trashed_at);
        var row = document.createElement('div');
        row.className = 'flex items-center gap-3 px-4 py-3 border-b border-gray-800 last:border-b-0 hover:bg-gray-800/50 transition-colors';
        row.dataset.trashId = item.id;

        row.innerHTML =
            '<div class="flex-1 min-w-0">' +
            '  <div class="text-sm font-medium text-white truncate">' + escapeHTML(item.feature.name) + '</div>' +
            '  <div class="text-xs text-gray-500 font-mono truncate">' + escapeHTML(item.feature.branch_name) + '</div>' +
            '  <div class="text-xs text-gray-600 mt-0.5">Trashed ' + escapeHTML(daysSince(item.trashed_at)) +
            '  &middot; <span class="' + (remaining <= 5 ? 'text-red-400' : 'text-gray-500') + '">' + remaining + ' day' + (remaining !== 1 ? 's' : '') + ' left</span></div>' +
            '</div>' +
            '<div class="flex items-center gap-1.5 flex-shrink-0">' +
            '  <button class="trash-restore px-2.5 py-1.5 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors">Restore</button>' +
            '  <button class="trash-delete p-1.5 text-gray-500 hover:text-red-400 rounded-lg hover:bg-gray-700 transition-colors" title="Delete permanently">' +
            '    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>' +
            '  </button>' +
            '</div>';

        // Restore handler
        row.querySelector('.trash-restore').addEventListener('click', function() {
            var btn = this;
            btn.disabled = true;
            btn.textContent = 'Restoring...';
            fetch('/api/trash/' + item.id + '/restore', { method: 'POST' })
                .then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                    return r.json();
                })
                .then(function(data) {
                    dialog.close();
                    dialog.remove();
                    window.location.href = '/projects/' + data.project_id + '/features/' + data.feature_id + '/';
                })
                .catch(function(err) {
                    btn.disabled = false;
                    btn.textContent = 'Restore';
                    alert('Restore failed: ' + err.message);
                });
        });

        // Permanent delete handler
        row.querySelector('.trash-delete').addEventListener('click', function() {
            ClawIDEDialog.confirm(
                'Permanently Delete',
                'Delete "' + item.feature.name + '" permanently? The git branch will also be removed. This cannot be undone.',
                { destructive: true, confirmLabel: 'Delete Forever' }
            ).then(function(confirmed) {
                if (!confirmed) return;
                fetch('/api/trash/' + item.id, { method: 'DELETE' })
                    .then(function(r) {
                        if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                        row.remove();
                        updateBadge();
                        checkEmpty(listEl);
                    })
                    .catch(function(err) {
                        alert('Delete failed: ' + err.message);
                    });
            });
        });

        return row;
    }

    function renderProjectItem(item, listEl, dialog) {
        var remaining = daysRemaining(item.trashed_at);
        var row = document.createElement('div');
        row.className = 'flex items-center gap-3 px-4 py-3 border-b border-gray-800 last:border-b-0 hover:bg-gray-800/50 transition-colors';
        row.dataset.trashId = item.id;

        row.innerHTML =
            '<div class="flex-1 min-w-0">' +
            '  <div class="text-sm font-medium text-white truncate">' + escapeHTML(item.project.name) + '</div>' +
            '  <div class="text-xs text-gray-500 font-mono truncate">' + escapeHTML(item.original_path) + '</div>' +
            '  <div class="text-xs text-gray-600 mt-0.5">Trashed ' + escapeHTML(daysSince(item.trashed_at)) +
            '  &middot; <span class="' + (remaining <= 5 ? 'text-red-400' : 'text-gray-500') + '">' + remaining + ' day' + (remaining !== 1 ? 's' : '') + ' left</span></div>' +
            '</div>' +
            '<div class="flex items-center gap-1.5 flex-shrink-0">' +
            '  <button class="trash-restore px-2.5 py-1.5 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors">Restore</button>' +
            '  <button class="trash-delete p-1.5 text-gray-500 hover:text-red-400 rounded-lg hover:bg-gray-700 transition-colors" title="Delete permanently">' +
            '    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>' +
            '  </button>' +
            '</div>';

        row.querySelector('.trash-restore').addEventListener('click', function() {
            var btn = this;
            btn.disabled = true;
            btn.textContent = 'Restoring...';
            fetch('/api/trash/projects/' + item.id + '/restore', { method: 'POST' })
                .then(function(r) {
                    if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                    return r.json();
                })
                .then(function() {
                    dialog.close();
                    dialog.remove();
                    window.location.href = '/';
                })
                .catch(function(err) {
                    btn.disabled = false;
                    btn.textContent = 'Restore';
                    alert('Restore failed: ' + err.message);
                });
        });

        row.querySelector('.trash-delete').addEventListener('click', function() {
            ClawIDEDialog.confirm(
                'Permanently Delete',
                'Delete "' + item.project.name + '" permanently? The project files will be removed from disk. This cannot be undone.',
                { destructive: true, confirmLabel: 'Delete Forever' }
            ).then(function(confirmed) {
                if (!confirmed) return;
                fetch('/api/trash/projects/' + item.id, { method: 'DELETE' })
                    .then(function(r) {
                        if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                        row.remove();
                        updateBadge();
                        checkEmpty(listEl);
                    })
                    .catch(function(err) {
                        alert('Delete failed: ' + err.message);
                    });
            });
        });

        return row;
    }

    function checkEmpty(listEl) {
        // The list contains group headers + item rows. Count only item rows
        // (those with data-trash-id) — if none remain, show the empty state.
        if (listEl.querySelectorAll('[data-trash-id]').length === 0) {
            listEl.innerHTML = '<div class="px-6 py-12 text-center text-gray-500 text-sm">Trash is empty</div>';
        }
    }

    function open() {
        Promise.all([
            fetch('/api/trash').then(function(r) { return r.json(); }).catch(function() { return []; }),
            fetch('/api/trash/projects').then(function(r) { return r.json(); }).catch(function() { return []; })
        ]).then(function(results) {
            var featureItems = results[0] || [];
            var projectItems = results[1] || [];
            var total = featureItems.length + projectItems.length;

            var dialog = document.createElement('dialog');
            dialog.className = DIALOG_STYLES;
            dialog.style.minWidth = '420px';
            dialog.style.maxWidth = '560px';
            dialog.style.maxHeight = '80vh';

            // Header
            var header = document.createElement('div');
            header.className = 'flex items-center justify-between px-6 pt-5 pb-3 border-b border-gray-700';
            header.innerHTML =
                '<h3 class="text-base font-semibold text-white flex items-center gap-2">' +
                '  <svg class="w-5 h-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>' +
                '  Trash' +
                '  <span class="text-xs font-normal text-gray-500">' + total + ' item' + (total !== 1 ? 's' : '') + '</span>' +
                '</h3>' +
                '<button class="dialog-close p-1 text-gray-400 hover:text-white rounded transition-colors">' +
                '  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>' +
                '</button>';
            dialog.appendChild(header);

            // List
            var listEl = document.createElement('div');
            listEl.className = 'overflow-y-auto';
            listEl.style.maxHeight = 'calc(80vh - 80px)';

            if (total === 0) {
                listEl.innerHTML = '<div class="px-6 py-12 text-center text-gray-500 text-sm">Trash is empty</div>';
            } else {
                // Projects section first (bigger ticket items)
                if (projectItems.length > 0) {
                    var projHeader = document.createElement('div');
                    projHeader.className = 'px-4 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider bg-gray-800/50';
                    projHeader.textContent = 'Projects';
                    listEl.appendChild(projHeader);
                    for (var pi = 0; pi < projectItems.length; pi++) {
                        listEl.appendChild(renderProjectItem(projectItems[pi], listEl, dialog));
                    }
                }

                // Features section — grouped by project like before
                if (featureItems.length > 0) {
                    if (projectItems.length > 0) {
                        var featHeader = document.createElement('div');
                        featHeader.className = 'px-4 py-2 text-xs font-semibold text-gray-500 uppercase tracking-wider bg-gray-800/50';
                        featHeader.textContent = 'Features';
                        listEl.appendChild(featHeader);
                    }
                    var groups = groupByProject(featureItems);
                    var projectNames = Object.keys(groups).sort();
                    for (var p = 0; p < projectNames.length; p++) {
                        var projectName = projectNames[p];
                        var groupItems = groups[projectName];

                        // Project subheader (when multiple projects, or when we already have a Projects section above)
                        if (projectNames.length > 1 || projectItems.length > 0) {
                            var groupHeader = document.createElement('div');
                            groupHeader.className = 'px-4 py-1.5 text-[11px] font-medium text-gray-500 tracking-wide bg-gray-800/30';
                            groupHeader.textContent = projectName;
                            listEl.appendChild(groupHeader);
                        }

                        for (var i = 0; i < groupItems.length; i++) {
                            listEl.appendChild(renderItem(groupItems[i], listEl, dialog));
                        }
                    }
                }
            }

            dialog.appendChild(listEl);

            // Close handler
            header.querySelector('.dialog-close').addEventListener('click', function() {
                dialog.close();
                dialog.remove();
            });

            dialog.addEventListener('close', function() {
                dialog.remove();
            });

            document.body.appendChild(dialog);
            dialog.showModal();
        }).catch(function(err) {
            console.error('Failed to load trash:', err);
        });
    }

    function updateBadge() {
        Promise.all([
            fetch('/api/trash').then(function(r) { return r.json(); }).catch(function() { return []; }),
            fetch('/api/trash/projects').then(function(r) { return r.json(); }).catch(function() { return []; })
        ]).then(function(results) {
            var total = (results[0] || []).length + (results[1] || []).length;
            var badges = document.querySelectorAll('#trash-count-badge');
            for (var i = 0; i < badges.length; i++) {
                var badge = badges[i];
                if (total > 0) {
                    badge.textContent = total;
                    badge.classList.remove('hidden');
                } else {
                    badge.classList.add('hidden');
                }
            }
        }).catch(function() {});
    }

    window.ClawIDETrash = {
        open: open,
        updateBadge: updateBadge
    };

    document.addEventListener('DOMContentLoaded', function() {
        ClawIDETrash.updateBadge();
    });
})();
