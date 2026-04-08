// ClawIDE MCP Servers Manager
// Sidebar server list + full-screen modal for CRUD management of MCP servers.
(function() {
    'use strict';

    var projectID = '';
    var servers = [];
    var selectedServer = null;
    var scopeFilter = 'all'; // 'all' | 'project' | 'global'
    var searchQuery = '';
    var isCreating = false;
    var modalEl = null;
    var logPollTimer = null;

    function getAPIBase() {
        return '/projects/' + projectID + '/api/mcp-servers';
    }

    // ── Sidebar ──────────────────────────────────────────────────

    function initSidebar() {
        var match = window.location.pathname.match(/\/projects\/([^/]+)/);
        if (match) projectID = match[1];
        if (!projectID) return;
        loadServers();
    }

    function loadServers(cb) {
        var url = getAPIBase();
        if (scopeFilter !== 'all') {
            url += '?scope=' + scopeFilter;
        }
        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                servers = data || [];
                renderSidebar();
                if (modalEl) renderModalList();
                if (cb) cb();
            })
            .catch(function(err) {
                console.error('Failed to load MCP servers:', err);
            });
    }

    function renderSidebar() {
        var container = document.getElementById('mcp-servers-sidebar');
        if (!container) return;

        if (servers.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs px-3 py-2">No MCP servers configured</div>';
            return;
        }

        var html = '';
        var filtered = filterServers(servers);
        var shown = filtered.slice(0, 10);
        for (var i = 0; i < shown.length; i++) {
            var srv = shown[i];
            var badge = srv.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300">G</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300">P</span>';
            var statusDot = getStatusDot(srv.status_info ? srv.status_info.status : 'stopped');
            html += '<div class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs text-th-text-tertiary hover:bg-surface-raised cursor-pointer truncate" '
                + 'onclick="ClawIDEMCPServers.openManager(\'' + escapeAttr(srv.scope) + '\', \'' + escapeAttr(srv.name) + '\')" '
                + 'title="' + escapeAttr(srv.command + ' ' + (srv.args || []).join(' ')) + '">'
                + statusDot + ' ' + badge + ' '
                + '<span class="truncate">' + escapeHTML(srv.name) + '</span>'
                + '</div>';
        }
        if (filtered.length > 10) {
            html += '<div class="text-th-text-faint text-[10px] px-3 py-1">+' + (filtered.length - 10) + ' more</div>';
        }
        container.innerHTML = html;
    }

    function getStatusDot(status) {
        if (status === 'running') {
            return '<span class="w-2 h-2 rounded-full bg-emerald-400 flex-shrink-0 inline-block"></span>';
        } else if (status === 'error') {
            return '<span class="w-2 h-2 rounded-full bg-red-400 flex-shrink-0 inline-block"></span>';
        }
        return '<span class="w-2 h-2 rounded-full bg-th-border-muted flex-shrink-0 inline-block"></span>';
    }

    // ── Modal ────────────────────────────────────────────────────

    function openManager(selectScope, selectName) {
        if (modalEl) {
            if (selectScope && selectName) {
                selectServerByRef(selectScope, selectName);
            }
            return;
        }

        createModal();
        loadServers(function() {
            if (selectScope && selectName) {
                selectServerByRef(selectScope, selectName);
            }
        });
    }

    function closeManager() {
        if (logPollTimer) {
            clearInterval(logPollTimer);
            logPollTimer = null;
        }
        if (modalEl) {
            modalEl.remove();
            modalEl = null;
            selectedServer = null;
            isCreating = false;
        }
    }

    function createModal() {
        modalEl = document.createElement('div');
        modalEl.id = 'mcp-servers-modal';
        modalEl.className = 'fixed inset-0 z-[200] flex items-center justify-center';

        modalEl.innerHTML = ''
            + '<div class="absolute inset-0 bg-black/70 backdrop-blur-sm" onclick="ClawIDEMCPServers.closeManager()"></div>'
            + '<div class="relative w-[90vw] max-w-5xl h-[80vh] bg-surface-base border border-th-border-strong rounded-xl shadow-2xl flex flex-col overflow-hidden">'
            + '  <div class="flex items-center justify-between px-5 py-3 border-b border-th-border">'
            + '    <h2 class="text-base font-semibold text-th-text-primary flex items-center gap-2">'
            + '      <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"/></svg>'
            + '      MCP Servers'
            + '    </h2>'
            + '    <button onclick="ClawIDEMCPServers.closeManager()" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '    </button>'
            + '  </div>'
            + '  <div class="flex flex-1 min-h-0">'
            // Left pane
            + '    <div class="w-72 border-r border-th-border flex flex-col flex-shrink-0 bg-surface-base/60">'
            + '      <div class="flex border-b border-th-border">'
            + '        <button id="mcp-tab-all" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEMCPServers.setFilter(\'all\')">All</button>'
            + '        <button id="mcp-tab-project" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEMCPServers.setFilter(\'project\')">Project</button>'
            + '        <button id="mcp-tab-global" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEMCPServers.setFilter(\'global\')">Global</button>'
            + '      </div>'
            + '      <div class="p-2">'
            + '        <input id="mcp-modal-search" type="text" placeholder="Search servers..."'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded px-2.5 py-1.5 text-xs text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500">'
            + '      </div>'
            + '      <div id="mcp-modal-list" class="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5"></div>'
            + '      <div class="p-2 border-t border-th-border">'
            + '        <button onclick="ClawIDEMCPServers.newServer()" class="w-full flex items-center justify-center gap-1.5 px-3 py-2 text-xs text-emerald-400 hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '          New MCP Server'
            + '        </button>'
            + '      </div>'
            + '    </div>'
            // Right pane
            + '    <div id="mcp-editor-pane" class="flex-1 flex flex-col min-w-0 overflow-hidden">'
            + '      <div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select a server or create a new one</div>'
            + '    </div>'
            + '  </div>'
            + '</div>';

        document.body.appendChild(modalEl);

        var searchInput = document.getElementById('mcp-modal-search');
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                searchQuery = this.value.trim().toLowerCase();
                renderModalList();
            });
        }

        modalEl.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                closeManager();
                e.stopPropagation();
            }
            if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                e.preventDefault();
                saveCurrentServer();
            }
        });

        updateFilterTabs();
        renderModalList();
    }

    function updateFilterTabs() {
        var tabs = ['all', 'project', 'global'];
        for (var i = 0; i < tabs.length; i++) {
            var tab = document.getElementById('mcp-tab-' + tabs[i]);
            if (!tab) continue;
            if (tabs[i] === scopeFilter) {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-emerald-400 border-b-2 border-emerald-400 transition-colors';
            } else {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-th-text-faint hover:text-th-text-tertiary transition-colors';
            }
        }
    }

    function setFilter(f) {
        scopeFilter = f;
        updateFilterTabs();
        loadServers();
    }

    function renderModalList() {
        var container = document.getElementById('mcp-modal-list');
        if (!container) return;

        var filtered = filterServers(servers);

        if (filtered.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs text-center py-4">No servers found</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < filtered.length; i++) {
            var srv = filtered[i];
            var isSelected = selectedServer && selectedServer.scope === srv.scope && selectedServer.name === srv.name;
            var badge = srv.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300 flex-shrink-0">Global</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300 flex-shrink-0">Project</span>';
            var statusDot = getStatusDot(srv.status_info ? srv.status_info.status : 'stopped');
            var cmd = srv.command || '';
            if (cmd.length > 40) cmd = cmd.substring(0, 40) + '...';

            html += '<div class="px-2.5 py-2 rounded-lg cursor-pointer transition-colors '
                + (isSelected ? 'bg-emerald-600/20 border border-emerald-500/30' : 'hover:bg-surface-raised border border-transparent')
                + '" onclick="ClawIDEMCPServers.selectServer(\'' + escapeAttr(srv.scope) + '\', \'' + escapeAttr(srv.name) + '\')">'
                + '<div class="flex items-center gap-1.5">'
                + '  ' + statusDot
                + '  <span class="text-sm text-th-text-primary font-medium truncate">' + escapeHTML(srv.name) + '</span>'
                + '  ' + badge
                + '</div>';
            if (cmd) {
                html += '<div class="text-[11px] text-th-text-faint mt-0.5 truncate pl-3.5">' + escapeHTML(cmd) + '</div>';
            }
            html += '</div>';
        }
        container.innerHTML = html;
    }

    function selectServerByRef(scope, name) {
        for (var i = 0; i < servers.length; i++) {
            if (servers[i].scope === scope && servers[i].name === name) {
                selectServer(scope, name);
                return;
            }
        }
    }

    function selectServer(scope, name) {
        isCreating = false;
        if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null; }

        fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(name))
            .then(function(r) {
                if (!r.ok) throw new Error('Failed to load server');
                return r.json();
            })
            .then(function(srv) {
                selectedServer = srv;
                renderEditor(srv);
                renderModalList();
            })
            .catch(function(err) {
                console.error('Failed to load MCP server:', err);
                showToast('Failed to load server', 'error');
            });
    }

    function newServer() {
        isCreating = true;
        if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null; }
        selectedServer = {
            name: '',
            command: '',
            args: [],
            env: {},
            autoStart: false,
            scope: 'project'
        };
        renderEditor(selectedServer);
        renderModalList();
        setTimeout(function() {
            var nameInput = document.getElementById('mcp-field-name');
            if (nameInput) nameInput.focus();
        }, 50);
    }

    // ── Editor Pane ──────────────────────────────────────────────

    function renderEditor(srv) {
        var pane = document.getElementById('mcp-editor-pane');
        if (!pane) return;

        var argsStr = (srv.args || []).join(', ');
        var statusInfo = srv.status_info || { status: 'stopped' };

        pane.innerHTML = ''
            + '<div class="flex-1 overflow-y-auto">'
            // Header
            + '<div class="sticky top-0 bg-surface-base z-10 px-5 py-3 border-b border-th-border flex items-center justify-between">'
            + '  <div class="flex items-center gap-2">'
            + '    <h3 class="text-sm font-semibold text-th-text-primary">' + (isCreating ? 'New MCP Server' : escapeHTML(srv.name)) + '</h3>'
            + '    ' + (srv.scope === 'global'
                ? '<span class="text-[10px] px-1.5 py-0.5 rounded bg-purple-900/50 text-purple-300">Global</span>'
                : '<span class="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/50 text-blue-300">Project</span>')
            + '  </div>'
            + '  <div class="flex items-center gap-2">'
            + '    <button onclick="ClawIDEMCPServers.saveCurrentServer()" class="px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 text-th-text-primary rounded-lg transition-colors flex items-center gap-1">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>'
            + '      Save'
            + '    </button>'
            + (isCreating ? '' : '<button onclick="ClawIDEMCPServers.moveCurrentServer()" class="px-3 py-1.5 text-xs text-th-text-muted hover:text-th-text-primary hover:bg-surface-overlay rounded-lg transition-colors flex items-center gap-1">'
                + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/></svg>'
                + (srv.scope === 'global' ? 'Move to Project' : 'Move to Global')
                + '</button>')
            + (isCreating ? '' : '<button onclick="ClawIDEMCPServers.deleteCurrentServer()" class="px-3 py-1.5 text-xs text-red-400 hover:text-th-text-primary hover:bg-red-900/50 rounded-lg transition-colors flex items-center gap-1">'
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
            + '        <input id="mcp-field-name" type="text" value="' + escapeAttr(srv.name) + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500"'
            + '               placeholder="my-mcp-server">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Scope</label>'
            + '        <select id="mcp-field-scope"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-emerald-500">'
            + '          <option value="project"' + (srv.scope === 'project' ? ' selected' : '') + '>Project</option>'
            + '          <option value="global"' + (srv.scope === 'global' ? ' selected' : '') + '>Global</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '    <div class="grid grid-cols-2 gap-3 mt-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Command *</label>'
            + '        <input id="mcp-field-command" type="text" value="' + escapeAttr(srv.command || '') + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500"'
            + '               placeholder="npx, node, python, etc.">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Arguments</label>'
            + '        <input id="mcp-field-args" type="text" value="' + escapeAttr(argsStr) + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500"'
            + '               placeholder="-y, @package/name, start">'
            + '        <p class="text-[10px] text-th-text-ghost mt-0.5">Comma-separated list of arguments</p>'
            + '      </div>'
            + '    </div>'
            + '    <div class="mt-3">'
            + '      <label class="flex items-center gap-2 cursor-pointer">'
            + '        <input id="mcp-field-autostart" type="checkbox"' + (srv.autoStart ? ' checked' : '')
            + '               class="w-4 h-4 rounded bg-surface-raised border-th-border-muted text-emerald-600 focus:ring-emerald-500 focus:ring-offset-surface-base">'
            + '        <span class="text-xs text-th-text-tertiary">Auto Start</span>'
            + '        <span class="text-[10px] text-th-text-ghost" title="If enabled, Claude Code will automatically start this server">?</span>'
            + '      </label>'
            + '    </div>'
            + '  </div>'

            // Environment Variables
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"/></svg>'
            + '      Environment Variables'
            + '    </h4>'
            + '    <div id="mcp-env-editor" class="space-y-2">'
            + renderEnvRows(srv.env || {})
            + '    </div>'
            + '    <button onclick="ClawIDEMCPServers._addEnvRow()" class="mt-2 flex items-center gap-1 px-2 py-1 text-[11px] text-emerald-400 hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors">'
            + '      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '      Add Variable'
            + '    </button>'
            + '  </div>'

            // Status & Actions (existing servers only)
            + (isCreating ? '' : renderStatusSection(srv, statusInfo))

            // Logs (existing servers only)
            + (isCreating ? '' : renderLogsSection())

            + '</div>'
            + '</div>';

        // Start log polling if viewing an existing server
        if (!isCreating) {
            refreshLogs();
            if (logPollTimer) clearInterval(logPollTimer);
            logPollTimer = setInterval(function() {
                if (selectedServer && !isCreating) {
                    refreshLogs();
                    refreshStatus();
                }
            }, 3000);
        }
    }

    function renderEnvRows(env) {
        var keys = Object.keys(env || {});
        if (keys.length === 0) {
            return '<div class="text-[11px] text-th-text-ghost">No environment variables set</div>';
        }
        var html = '';
        for (var i = 0; i < keys.length; i++) {
            html += renderEnvRow(keys[i], env[keys[i]], i);
        }
        return html;
    }

    function renderEnvRow(key, value, idx) {
        return '<div class="flex items-center gap-2 mcp-env-row" data-idx="' + idx + '">'
            + '  <input type="text" value="' + escapeAttr(key) + '" placeholder="KEY"'
            + '         class="mcp-env-key flex-1 bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 text-xs text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500 font-mono">'
            + '  <div class="flex-1 relative">'
            + '    <input type="password" value="' + escapeAttr(value) + '" placeholder="value"'
            + '           class="mcp-env-val w-full bg-surface-raised border border-th-border-strong rounded px-2 py-1.5 pr-7 text-xs text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-emerald-500 font-mono">'
            + '    <button onclick="ClawIDEMCPServers._toggleEnvVisibility(this)" class="absolute right-1.5 top-1/2 -translate-y-1/2 text-th-text-faint hover:text-th-text-tertiary" title="Toggle visibility">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"/></svg>'
            + '    </button>'
            + '  </div>'
            + '  <button onclick="this.closest(\'.mcp-env-row\').remove()" class="p-1 text-th-text-faint hover:text-red-400 transition-colors flex-shrink-0">'
            + '    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '  </button>'
            + '</div>';
    }

    function renderStatusSection(srv, statusInfo) {
        var status = statusInfo.status || 'stopped';
        var statusColor = status === 'running' ? 'text-emerald-400' : (status === 'error' ? 'text-red-400' : 'text-th-text-muted');
        var statusBg = status === 'running' ? 'bg-emerald-900/30' : (status === 'error' ? 'bg-red-900/30' : 'bg-surface-raised');
        var uptime = '';
        if (status === 'running' && statusInfo.uptime_seconds) {
            uptime = ' (' + formatUptime(statusInfo.uptime_seconds) + ')';
        }

        return ''
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"/></svg>'
            + '      Status & Actions'
            + '    </h4>'
            + '    <div class="flex items-center gap-3">'
            + '      <span id="mcp-status-badge" class="px-2.5 py-1 rounded-full text-xs font-medium ' + statusColor + ' ' + statusBg + '">'
            + '        ' + status.charAt(0).toUpperCase() + status.slice(1) + uptime
            + '      </span>'
            + (statusInfo.error ? '<span class="text-xs text-red-400 truncate">' + escapeHTML(statusInfo.error) + '</span>' : '')
            + '      <div class="flex items-center gap-1.5 ml-auto">'
            + '        <button onclick="ClawIDEMCPServers._startServer()" class="px-2.5 py-1.5 text-xs bg-emerald-700 hover:bg-emerald-600 text-th-text-primary rounded-lg transition-colors disabled:opacity-50"'
            + (status === 'running' ? ' disabled' : '') + '>Start</button>'
            + '        <button onclick="ClawIDEMCPServers._stopServer()" class="px-2.5 py-1.5 text-xs bg-surface-overlay hover:bg-th-border-muted text-th-text-primary rounded-lg transition-colors disabled:opacity-50"'
            + (status !== 'running' ? ' disabled' : '') + '>Stop</button>'
            + '        <button onclick="ClawIDEMCPServers._restartServer()" class="px-2.5 py-1.5 text-xs bg-surface-overlay hover:bg-th-border-muted text-th-text-primary rounded-lg transition-colors">Restart</button>'
            + '      </div>'
            + '    </div>'
            + '  </div>';
    }

    function renderLogsSection() {
        return ''
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center justify-between">'
            + '      <span class="flex items-center gap-1.5">'
            + '        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"/></svg>'
            + '        Logs'
            + '      </span>'
            + '      <button onclick="ClawIDEMCPServers._refreshLogs()" class="text-[10px] text-th-text-faint hover:text-th-text-tertiary transition-colors">Refresh</button>'
            + '    </h4>'
            + '    <div id="mcp-logs-container" class="bg-surface-deepest border border-th-border rounded-lg p-3 max-h-48 overflow-y-auto font-mono text-[11px] text-th-text-muted">'
            + '      <div class="text-th-text-ghost">No logs yet. Start the server to see output.</div>'
            + '    </div>'
            + '  </div>';
    }

    function refreshLogs() {
        if (!selectedServer || isCreating) return;
        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/logs')
            .then(function(r) { return r.json(); })
            .then(function(data) {
                var container = document.getElementById('mcp-logs-container');
                if (!container) return;
                var lines = data.lines || [];
                if (lines.length === 0) {
                    container.innerHTML = '<div class="text-th-text-ghost">No logs yet. Start the server to see output.</div>';
                } else {
                    container.innerHTML = lines.map(function(line) {
                        return '<div class="whitespace-pre-wrap break-all">' + escapeHTML(line) + '</div>';
                    }).join('');
                    container.scrollTop = container.scrollHeight;
                }
            })
            .catch(function() {});
    }

    function refreshStatus() {
        if (!selectedServer || isCreating) return;
        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/status')
            .then(function(r) { return r.json(); })
            .then(function(info) {
                var badge = document.getElementById('mcp-status-badge');
                if (!badge) return;
                var status = info.status || 'stopped';
                var statusColor = status === 'running' ? 'text-emerald-400' : (status === 'error' ? 'text-red-400' : 'text-th-text-muted');
                var statusBg = status === 'running' ? 'bg-emerald-900/30' : (status === 'error' ? 'bg-red-900/30' : 'bg-surface-raised');
                var uptime = '';
                if (status === 'running' && info.uptime_seconds) {
                    uptime = ' (' + formatUptime(info.uptime_seconds) + ')';
                }
                badge.className = 'px-2.5 py-1 rounded-full text-xs font-medium ' + statusColor + ' ' + statusBg;
                badge.textContent = status.charAt(0).toUpperCase() + status.slice(1) + uptime;

                // Update sidebar too
                loadServers();
            })
            .catch(function() {});
    }

    // ── CRUD Operations ──────────────────────────────────────────

    function gatherFormData() {
        var argsRaw = val('mcp-field-args');
        var args = argsRaw ? argsRaw.split(',').map(function(s) { return s.trim(); }).filter(Boolean) : [];

        var env = {};
        var rows = document.querySelectorAll('.mcp-env-row');
        for (var i = 0; i < rows.length; i++) {
            var keyEl = rows[i].querySelector('.mcp-env-key');
            var valEl = rows[i].querySelector('.mcp-env-val');
            if (keyEl && valEl && keyEl.value.trim()) {
                env[keyEl.value.trim()] = valEl.value;
            }
        }

        return {
            name: val('mcp-field-name'),
            command: val('mcp-field-command'),
            args: args,
            env: env,
            autoStart: checked('mcp-field-autostart'),
            scope: val('mcp-field-scope')
        };
    }

    function saveCurrentServer() {
        var data = gatherFormData();
        if (!data.name) {
            showToast('Server name is required', 'error');
            return;
        }
        if (!data.command) {
            showToast('Command is required', 'error');
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
            .then(function() {
                showToast('MCP server created', 'success');
                isCreating = false;
                loadServers(function() {
                    selectServerByRef(data.scope, data.name);
                });
            })
            .catch(function(err) {
                showToast('Failed to create: ' + err.message, 'error');
            });
        } else {
            var scope = selectedServer.scope;
            var name = selectedServer.name;
            fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(name), {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function() {
                showToast('MCP server updated', 'success');
                loadServers(function() {
                    selectServerByRef(data.scope || scope, data.name || name);
                });
            })
            .catch(function(err) {
                showToast('Failed to update: ' + err.message, 'error');
            });
        }
    }

    function deleteCurrentServer() {
        if (!selectedServer || isCreating) return;

        if (!confirm('Delete MCP server "' + selectedServer.name + '"? This will remove it from the configuration file.')) {
            return;
        }

        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name), {
            method: 'DELETE'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('MCP server deleted', 'success');
            selectedServer = null;
            if (logPollTimer) { clearInterval(logPollTimer); logPollTimer = null; }
            var pane = document.getElementById('mcp-editor-pane');
            if (pane) {
                pane.innerHTML = '<div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select a server or create a new one</div>';
            }
            loadServers();
        })
        .catch(function(err) {
            showToast('Failed to delete: ' + err.message, 'error');
        });
    }

    function moveCurrentServer() {
        if (!selectedServer || isCreating) return;

        var targetScope = selectedServer.scope === 'global' ? 'project' : 'global';
        if (!confirm('Move MCP server "' + selectedServer.name + '" to ' + targetScope + ' scope?')) {
            return;
        }

        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_scope: targetScope })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('MCP server moved to ' + targetScope, 'success');
            var name = selectedServer.name;
            loadServers(function() {
                selectServerByRef(targetScope, name);
            });
        })
        .catch(function(err) {
            showToast('Failed to move: ' + err.message, 'error');
        });
    }

    // ── Process Lifecycle ─────────────────────────────────────────

    function startServer() {
        if (!selectedServer || isCreating) return;
        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/start', {
            method: 'POST'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Server started', 'success');
            setTimeout(function() {
                refreshStatus();
                refreshLogs();
            }, 500);
        })
        .catch(function(err) {
            showToast('Failed to start: ' + err.message, 'error');
        });
    }

    function stopServer() {
        if (!selectedServer || isCreating) return;
        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/stop', {
            method: 'POST'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Server stopped', 'success');
            setTimeout(function() {
                refreshStatus();
                refreshLogs();
            }, 500);
        })
        .catch(function(err) {
            showToast('Failed to stop: ' + err.message, 'error');
        });
    }

    function restartServer() {
        if (!selectedServer || isCreating) return;
        fetch(getAPIBase() + '/' + selectedServer.scope + '/' + encodeURIComponent(selectedServer.name) + '/restart', {
            method: 'POST'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Server restarted', 'success');
            setTimeout(function() {
                refreshStatus();
                refreshLogs();
            }, 500);
        })
        .catch(function(err) {
            showToast('Failed to restart: ' + err.message, 'error');
        });
    }

    // ── Helpers ───────────────────────────────────────────────────

    function addEnvRow() {
        var container = document.getElementById('mcp-env-editor');
        if (!container) return;
        // Remove empty state message if present
        var emptyMsg = container.querySelector('.text-th-text-ghost');
        if (emptyMsg) emptyMsg.remove();

        var idx = container.querySelectorAll('.mcp-env-row').length;
        var div = document.createElement('div');
        div.innerHTML = renderEnvRow('', '', idx);
        container.appendChild(div.firstElementChild);
    }

    function toggleEnvVisibility(btn) {
        var input = btn.closest('.relative').querySelector('input');
        if (!input) return;
        if (input.type === 'password') {
            input.type = 'text';
        } else {
            input.type = 'password';
        }
    }

    function filterServers(list) {
        if (!searchQuery) return list;
        return list.filter(function(srv) {
            var hay = (srv.name + ' ' + (srv.command || '')).toLowerCase();
            return hay.indexOf(searchQuery) !== -1;
        });
    }

    function formatUptime(seconds) {
        if (seconds < 60) return Math.floor(seconds) + 's';
        if (seconds < 3600) return Math.floor(seconds / 60) + 'm ' + Math.floor(seconds % 60) + 's';
        var h = Math.floor(seconds / 3600);
        var m = Math.floor((seconds % 3600) / 60);
        return h + 'h ' + m + 'm';
    }

    function val(id) {
        var el = document.getElementById(id);
        return el ? el.value : '';
    }

    function checked(id) {
        var el = document.getElementById(id);
        return el ? el.checked : false;
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
            console.log('[MCP Servers] ' + type + ': ' + msg);
        }
    }

    // ── Init ─────────────────────────────────────────────────────

    document.addEventListener('DOMContentLoaded', initSidebar);

    // Public API
    window.ClawIDEMCPServers = {
        openManager: openManager,
        closeManager: closeManager,
        selectServer: selectServer,
        newServer: newServer,
        setFilter: setFilter,
        saveCurrentServer: saveCurrentServer,
        deleteCurrentServer: deleteCurrentServer,
        moveCurrentServer: moveCurrentServer,
        reload: loadServers,
        _addEnvRow: addEnvRow,
        _toggleEnvVisibility: toggleEnvVisibility,
        _startServer: startServer,
        _stopServer: stopServer,
        _restartServer: restartServer,
        _refreshLogs: refreshLogs
    };
})();
