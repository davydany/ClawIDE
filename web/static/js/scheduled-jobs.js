// ClawIDE Scheduled Jobs Manager
// Sidebar job list + full-screen modal for CRUD management of scheduled jobs (loops).
(function() {
    'use strict';

    var projectID = '';
    var jobs = [];
    var sessions = [];
    var selectedJob = null;
    var isCreating = false;
    var modalEl = null;

    function getAPIBase() {
        return '/projects/' + projectID + '/api/scheduled-jobs';
    }

    // ── Sidebar ──────────────────────────────────────────────────

    function initSidebar() {
        var section = document.getElementById('scheduled-jobs-section');
        if (section && section.dataset.projectId) {
            projectID = section.dataset.projectId;
        }
        if (!projectID) {
            var match = window.location.pathname.match(/\/projects\/([^/]+)/);
            if (match) projectID = match[1];
        }
        if (!projectID) return;

        loadJobs();
    }

    function loadJobs(cb) {
        fetch(getAPIBase())
            .then(function(r) { return r.json(); })
            .then(function(data) {
                jobs = data || [];
                renderSidebar();
                if (modalEl) renderModalList();
                if (cb) cb();
            })
            .catch(function(err) {
                console.error('Failed to load scheduled jobs:', err);
            });
    }

    function renderSidebar() {
        var container = document.getElementById('scheduled-jobs-sidebar');
        if (!container) return;

        if (jobs.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs px-3 py-2">No scheduled jobs</div>';
            return;
        }

        var html = '';
        var shown = jobs.slice(0, 10);
        for (var i = 0; i < shown.length; i++) {
            var job = shown[i];
            var dotColor = job.status === 'running' ? 'bg-green-500' : 'bg-neutral-500';
            var agentBadge = job.agent === 'codex'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-orange-900/50 text-orange-300">Codex</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300">Claude</span>';
            html += '<div class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs text-th-text-tertiary hover:bg-surface-raised cursor-pointer truncate" '
                + 'onclick="ClawIDEScheduledJobs.openManager(\'' + escapeAttr(job.id) + '\')" '
                + 'title="' + escapeAttr(job.name + ' — ' + job.prompt) + '">'
                + '<span class="w-2 h-2 rounded-full ' + dotColor + ' flex-shrink-0"></span> '
                + agentBadge + ' '
                + '<span class="truncate">' + escapeHTML(job.name) + '</span>'
                + '</div>';
        }
        if (jobs.length > 10) {
            html += '<div class="text-th-text-faint text-[10px] px-3 py-1">+' + (jobs.length - 10) + ' more</div>';
        }
        container.innerHTML = html;
    }

    // ── Pane helpers ─────────────────────────────────────────────

    function loadSessions(cb) {
        fetch('/projects/' + projectID + '/sessions/')
            .then(function(r) { return r.json(); })
            .then(function(data) {
                sessions = data || [];
                if (cb) cb();
            })
            .catch(function(err) {
                console.error('Failed to load sessions:', err);
                sessions = [];
                if (cb) cb();
            });
    }

    function collectPanes(node, sessionName, result) {
        if (!node) return;
        if (node.type === 'leaf') {
            result.push({
                paneID: node.pane_id,
                tmuxName: node.tmux_name,
                paneName: node.name || node.pane_type || 'pane',
                paneType: node.pane_type || 'shell',
                sessionName: sessionName
            });
        } else if (node.type === 'split') {
            collectPanes(node.first, sessionName, result);
            collectPanes(node.second, sessionName, result);
        }
    }

    function getAllPanes() {
        var panes = [];
        for (var i = 0; i < sessions.length; i++) {
            var sess = sessions[i];
            if (sess.layout) {
                collectPanes(sess.layout, sess.name, panes);
            }
        }
        return panes;
    }

    function buildPaneOptions(selectedPaneID) {
        var panes = getAllPanes();
        var html = '<option value="">Select a pane...</option>';
        for (var i = 0; i < panes.length; i++) {
            var p = panes[i];
            var label = p.sessionName + ' > ' + p.paneName;
            if (p.paneType === 'agent') label += ' (agent)';
            var sel = p.paneID === selectedPaneID ? ' selected' : '';
            html += '<option value="' + escapeAttr(p.paneID) + '"' + sel + '>' + escapeHTML(label) + '</option>';
        }
        return html;
    }

    // ── Modal ────────────────────────────────────────────────────

    function openManager(selectJobID) {
        if (modalEl) {
            if (selectJobID) {
                selectJobByID(selectJobID);
            }
            return;
        }

        createModal();
        loadSessions(function() {
            loadJobs(function() {
                if (selectJobID) {
                    selectJobByID(selectJobID);
                }
            });
        });
    }

    function closeManager() {
        if (modalEl) {
            modalEl.remove();
            modalEl = null;
            selectedJob = null;
            isCreating = false;
        }
    }

    function createModal() {
        modalEl = document.createElement('div');
        modalEl.id = 'scheduled-jobs-modal';
        modalEl.className = 'fixed inset-0 z-[200] flex items-center justify-center';

        modalEl.innerHTML = ''
            // Backdrop
            + '<div class="absolute inset-0 bg-black/70 backdrop-blur-sm" onclick="ClawIDEScheduledJobs.closeManager()"></div>'
            // Modal container
            + '<div class="relative w-[90vw] max-w-5xl h-[80vh] bg-surface-base border border-th-border-strong rounded-xl shadow-2xl flex flex-col overflow-hidden">'
            // Header
            + '  <div class="flex items-center justify-between px-5 py-3 border-b border-th-border">'
            + '    <h2 class="text-base font-semibold text-th-text-primary flex items-center gap-2">'
            + '      <svg class="w-5 h-5 text-accent-text" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
            + '      Scheduled Jobs'
            + '    </h2>'
            + '    <button onclick="ClawIDEScheduledJobs.closeManager()" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '    </button>'
            + '  </div>'
            // Body: two-pane
            + '  <div class="flex flex-1 min-h-0">'
            // Left pane: list
            + '    <div class="w-72 border-r border-th-border flex flex-col flex-shrink-0 bg-surface-base/60">'
            // List
            + '      <div id="scheduled-jobs-modal-list" class="flex-1 overflow-y-auto px-2 py-2 space-y-0.5"></div>'
            // New job button
            + '      <div class="p-2 border-t border-th-border">'
            + '        <button onclick="ClawIDEScheduledJobs.newJob()" class="w-full flex items-center justify-center gap-1.5 px-3 py-2 text-xs text-accent-text hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '          New Scheduled Job'
            + '        </button>'
            + '      </div>'
            + '    </div>'
            // Right pane: editor
            + '    <div id="scheduled-jobs-editor-pane" class="flex-1 flex flex-col min-w-0 overflow-hidden">'
            + '      <div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select a job or create a new one</div>'
            + '    </div>'
            + '  </div>'
            + '</div>';

        document.body.appendChild(modalEl);

        // Keyboard
        modalEl.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                closeManager();
                e.stopPropagation();
            }
            if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                e.preventDefault();
                saveCurrentJob();
            }
        });

        renderModalList();
    }

    function renderModalList() {
        var container = document.getElementById('scheduled-jobs-modal-list');
        if (!container) return;

        if (jobs.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs text-center py-4">No scheduled jobs yet</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < jobs.length; i++) {
            var job = jobs[i];
            var isSelected = selectedJob && selectedJob.id === job.id;
            var dotColor = job.status === 'running' ? 'bg-green-500' : 'bg-neutral-500';
            var statusLabel = job.status === 'running' ? 'Running' : 'Idle';
            var intervalLabel = job.interval ? job.interval : 'dynamic';

            html += '<div class="px-2.5 py-2 rounded-lg cursor-pointer transition-colors '
                + (isSelected ? 'bg-accent/20 border border-accent-border/30' : 'hover:bg-surface-raised border border-transparent')
                + '" onclick="ClawIDEScheduledJobs.selectJob(\'' + escapeAttr(job.id) + '\')">'
                + '<div class="flex items-center gap-1.5">'
                + '  <span class="w-2 h-2 rounded-full ' + dotColor + ' flex-shrink-0"></span>'
                + '  <span class="text-sm text-th-text-primary font-medium truncate">' + escapeHTML(job.name) + '</span>'
                + '</div>'
                + '<div class="text-[11px] text-th-text-faint mt-0.5 truncate">'
                + escapeHTML(job.agent) + ' &middot; ' + escapeHTML(intervalLabel) + ' &middot; ' + escapeHTML(statusLabel)
                + '</div>'
                + '</div>';
        }
        container.innerHTML = html;
    }

    function selectJobByID(id) {
        for (var i = 0; i < jobs.length; i++) {
            if (jobs[i].id === id) {
                selectedJob = jobs[i];
                isCreating = false;
                renderEditor(jobs[i]);
                renderModalList();
                return;
            }
        }
    }

    function selectJob(id) {
        selectJobByID(id);
    }

    function newJob() {
        isCreating = true;
        selectedJob = {
            id: '',
            name: '',
            job_type: 'loop',
            agent: 'claude',
            interval: '',
            prompt: '',
            target_pane_id: '',
            status: 'idle'
        };
        renderEditor(selectedJob);
        renderModalList();
        setTimeout(function() {
            var nameInput = document.getElementById('job-field-name');
            if (nameInput) nameInput.focus();
        }, 50);
    }

    // ── Editor Pane ──────────────────────────────────────────────

    function renderEditor(job) {
        var pane = document.getElementById('scheduled-jobs-editor-pane');
        if (!pane) return;

        var isRunning = job.status === 'running';

        pane.innerHTML = ''
            + '<div class="flex-1 overflow-y-auto">'
            // Editor header
            + '<div class="sticky top-0 bg-surface-base z-10 px-5 py-3 border-b border-th-border flex items-center justify-between">'
            + '  <div class="flex items-center gap-2">'
            + '    <h3 class="text-sm font-semibold text-th-text-primary">' + (isCreating ? 'New Scheduled Job' : escapeHTML(job.name)) + '</h3>'
            + '    ' + (isRunning
                ? '<span class="text-[10px] px-1.5 py-0.5 rounded bg-green-900/50 text-green-300">Running</span>'
                : '<span class="text-[10px] px-1.5 py-0.5 rounded bg-neutral-800/50 text-neutral-400">Idle</span>')
            + '  </div>'
            + '  <div class="flex items-center gap-2">'
            // Start/Stop button
            + (isCreating ? '' : (isRunning
                ? '<button onclick="ClawIDEScheduledJobs.stopJob()" class="px-3 py-1.5 text-xs bg-red-600 hover:bg-red-700 text-white rounded-lg transition-colors flex items-center gap-1">'
                  + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="1" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"/></svg>'
                  + 'Stop'
                  + '</button>'
                : '<button onclick="ClawIDEScheduledJobs.startJob()" class="px-3 py-1.5 text-xs bg-green-600 hover:bg-green-700 text-white rounded-lg transition-colors flex items-center gap-1">'
                  + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polygon points="5 3 19 12 5 21 5 3" stroke-linecap="round" stroke-linejoin="round" stroke-width="2"/></svg>'
                  + 'Start'
                  + '</button>'))
            // Save button
            + '    <button onclick="ClawIDEScheduledJobs.saveCurrentJob()" class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded-lg transition-colors flex items-center gap-1">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>'
            + '      Save'
            + '    </button>'
            // Delete button
            + (isCreating ? '' : '<button onclick="ClawIDEScheduledJobs.deleteCurrentJob()" class="px-3 py-1.5 text-xs text-red-400 hover:text-th-text-primary hover:bg-red-900/50 rounded-lg transition-colors flex items-center gap-1">'
                + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>'
                + 'Delete'
                + '</button>')
            + '  </div>'
            + '</div>'
            // Form fields
            + '<div class="px-5 py-4 space-y-5">'

            // Basic Information
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
            + '      Basic Information'
            + '    </h4>'
            + '    <div class="grid grid-cols-2 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Name *</label>'
            + '        <input id="job-field-name" type="text" value="' + escapeAttr(job.name) + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border"'
            + '               placeholder="e.g. Babysit PRs">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Type</label>'
            + '        <select id="job-field-type" disabled'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border opacity-70">'
            + '          <option value="loop" selected>Loop</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '  </div>'

            // Configuration
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>'
            + '      Configuration'
            + '    </h4>'
            + '    <div class="grid grid-cols-2 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Agent</label>'
            + '        <select id="job-field-agent"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '          <option value="claude"' + (job.agent === 'claude' ? ' selected' : '') + '>Claude Code</option>'
            + '          <option value="codex"' + (job.agent === 'codex' ? ' selected' : '') + '>Codex</option>'
            + '        </select>'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Interval</label>'
            + '        <select id="job-field-interval"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '          <option value=""' + (job.interval === '' ? ' selected' : '') + '>Dynamic (model self-paces)</option>'
            + '          <option value="1m"' + (job.interval === '1m' ? ' selected' : '') + '>Every 1 minute</option>'
            + '          <option value="5m"' + (job.interval === '5m' ? ' selected' : '') + '>Every 5 minutes</option>'
            + '          <option value="15m"' + (job.interval === '15m' ? ' selected' : '') + '>Every 15 minutes</option>'
            + '          <option value="30m"' + (job.interval === '30m' ? ' selected' : '') + '>Every 30 minutes</option>'
            + '          <option value="1h"' + (job.interval === '1h' ? ' selected' : '') + '>Every 1 hour</option>'
            + '          <option value="custom"' + (isCustomInterval(job.interval) ? ' selected' : '') + '>Custom...</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            // Custom interval input (hidden by default)
            + '    <div id="job-custom-interval-row" class="mt-3' + (isCustomInterval(job.interval) ? '' : ' hidden') + '">'
            + '      <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Custom Interval</label>'
            + '      <input id="job-field-custom-interval" type="text" value="' + escapeAttr(isCustomInterval(job.interval) ? job.interval : '') + '"'
            + '             class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border"'
            + '             placeholder="e.g. 10m, 2h, 45s">'
            + '    </div>'
            + '  </div>'

            // Prompt
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"/></svg>'
            + '      Prompt *'
            + '    </h4>'
            + '    <textarea id="job-field-prompt" rows="4"'
            + '              class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border font-mono resize-y"'
            + '              placeholder="e.g. /babysit-prs or Check for failing CI tests and fix them">' + escapeHTML(job.prompt) + '</textarea>'
            + '    <p class="text-[10px] text-th-text-ghost mt-1">The slash command or prompt text to run on each loop iteration.</p>'
            + '  </div>'

            // Target Pane
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>'
            + '      Target Pane'
            + '    </h4>'
            + '    <select id="job-field-target-pane"'
            + '            class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + buildPaneOptions(job.target_pane_id)
            + '    </select>'
            + '    <p class="text-[10px] text-th-text-ghost mt-1">The terminal pane where the /loop command will be sent.</p>'
            + '    <button onclick="ClawIDEScheduledJobs.refreshPanes()" class="mt-1.5 text-[10px] text-accent-text hover:text-th-text-primary transition-colors">'
            + '      Refresh pane list'
            + '    </button>'
            + '  </div>'

            // Last run info (only for existing jobs)
            + (isCreating ? '' : '<div>'
                + '  <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-2 flex items-center gap-1.5">'
                + '    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
                + '    Status'
                + '  </h4>'
                + '  <div class="text-xs text-th-text-tertiary space-y-1">'
                + '    <div>Status: <span class="font-medium">' + escapeHTML(job.status) + '</span></div>'
                + '    <div>Last run: <span class="font-medium">' + (job.last_run_at ? formatDate(job.last_run_at) : 'Never') + '</span></div>'
                + '    <div>Created: <span class="font-medium">' + formatDate(job.created_at) + '</span></div>'
                + '  </div>'
                + '</div>')

            + '</div>' // end space-y-5
            + '</div>'; // end overflow-y-auto

        // Wire up interval toggle
        var intervalSelect = document.getElementById('job-field-interval');
        if (intervalSelect) {
            intervalSelect.addEventListener('change', function() {
                var row = document.getElementById('job-custom-interval-row');
                if (row) {
                    if (this.value === 'custom') {
                        row.classList.remove('hidden');
                    } else {
                        row.classList.add('hidden');
                    }
                }
            });
        }
    }

    // ── CRUD Operations ──────────────────────────────────────────

    function gatherFormData() {
        var interval = val('job-field-interval');
        if (interval === 'custom') {
            interval = val('job-field-custom-interval');
        }
        return {
            name: val('job-field-name'),
            job_type: 'loop',
            agent: val('job-field-agent'),
            interval: interval,
            prompt: val('job-field-prompt'),
            target_pane_id: val('job-field-target-pane')
        };
    }

    function saveCurrentJob() {
        var data = gatherFormData();
        if (!data.name) {
            showToast('Job name is required', 'error');
            return;
        }
        if (!data.prompt) {
            showToast('Prompt is required', 'error');
            return;
        }

        if (isCreating) {
            fetch(getAPIBase(), {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function(created) {
                showToast('Scheduled job created', 'success');
                isCreating = false;
                loadJobs(function() {
                    selectJobByID(created.id);
                });
            })
            .catch(function(err) {
                showToast('Failed to create: ' + err.message, 'error');
            });
        } else {
            fetch(getAPIBase() + '/' + selectedJob.id, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function(updated) {
                showToast('Scheduled job updated', 'success');
                loadJobs(function() {
                    selectJobByID(updated.id);
                });
            })
            .catch(function(err) {
                showToast('Failed to update: ' + err.message, 'error');
            });
        }
    }

    function deleteCurrentJob() {
        if (!selectedJob || isCreating) return;
        if (!confirm('Delete scheduled job "' + selectedJob.name + '"?')) return;

        fetch(getAPIBase() + '/' + selectedJob.id, { method: 'DELETE' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                showToast('Scheduled job deleted', 'success');
                selectedJob = null;
                var pane = document.getElementById('scheduled-jobs-editor-pane');
                if (pane) {
                    pane.innerHTML = '<div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select a job or create a new one</div>';
                }
                loadJobs();
            })
            .catch(function(err) {
                showToast('Failed to delete: ' + err.message, 'error');
            });
    }

    function startJob() {
        if (!selectedJob || isCreating) return;

        fetch(getAPIBase() + '/' + selectedJob.id + '/start', { method: 'POST' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function(updated) {
                showToast('Loop started', 'success');
                loadJobs(function() {
                    selectJobByID(updated.id);
                });
            })
            .catch(function(err) {
                showToast('Failed to start: ' + err.message, 'error');
            });
    }

    function stopJob() {
        if (!selectedJob || isCreating) return;
        if (!confirm('Stop this loop? This sends Ctrl+C to the target pane, which will interrupt whatever is running there.')) return;

        fetch(getAPIBase() + '/' + selectedJob.id + '/stop', { method: 'POST' })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function(updated) {
                showToast('Loop stopped', 'success');
                loadJobs(function() {
                    selectJobByID(updated.id);
                });
            })
            .catch(function(err) {
                showToast('Failed to stop: ' + err.message, 'error');
            });
    }

    function refreshPanes() {
        loadSessions(function() {
            if (selectedJob || isCreating) {
                var currentTarget = val('job-field-target-pane');
                var select = document.getElementById('job-field-target-pane');
                if (select) {
                    select.innerHTML = buildPaneOptions(currentTarget);
                }
                showToast('Pane list refreshed', 'success');
            }
        });
    }

    // ── Helpers ───────────────────────────────────────────────────

    var knownIntervals = ['', '1m', '5m', '15m', '30m', '1h'];
    function isCustomInterval(interval) {
        return interval && knownIntervals.indexOf(interval) === -1;
    }

    function formatDate(dateStr) {
        if (!dateStr) return '';
        var d = new Date(dateStr);
        return d.toLocaleString();
    }

    function val(id) {
        var el = document.getElementById(id);
        return el ? el.value : '';
    }

    function escapeHTML(s) {
        if (!s) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(s));
        return div.innerHTML;
    }

    function escapeAttr(s) {
        if (!s) return '';
        return s.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    function showToast(msg, type) {
        if (typeof ClawIDEToast !== 'undefined') {
            if (type === 'error') {
                ClawIDEToast.error(msg);
            } else {
                ClawIDEToast.success(msg);
            }
        } else {
            console.log('[ScheduledJobs] ' + type + ': ' + msg);
        }
    }

    // ── Init ─────────────────────────────────────────────────────

    document.addEventListener('DOMContentLoaded', initSidebar);

    // Public API
    window.ClawIDEScheduledJobs = {
        openManager: openManager,
        closeManager: closeManager,
        selectJob: selectJob,
        newJob: newJob,
        saveCurrentJob: saveCurrentJob,
        deleteCurrentJob: deleteCurrentJob,
        startJob: startJob,
        stopJob: stopJob,
        refreshPanes: refreshPanes,
        reload: loadJobs
    };
})();
