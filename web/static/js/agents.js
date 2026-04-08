// ClawIDE Agents Manager
// Sidebar agent list + full-screen modal for CRUD management of Claude Code Agents.
(function() {
    'use strict';

    var projectID = '';
    var agents = [];
    var selectedAgent = null;
    var scopeFilter = 'all'; // 'all' | 'project' | 'global'
    var searchQuery = '';
    var isCreating = false;
    var modalEl = null;

    function getAPIBase() {
        return '/projects/' + projectID + '/api/agents';
    }

    // ── Sidebar ──────────────────────────────────────────────────

    function initSidebar() {
        var match = window.location.pathname.match(/\/projects\/([^/]+)/);
        if (match) projectID = match[1];
        if (!projectID) return;

        loadAgents();
    }

    function loadAgents(cb) {
        var url = getAPIBase();
        if (scopeFilter !== 'all') {
            url += '?scope=' + scopeFilter;
        }
        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                agents = data || [];
                renderSidebar();
                if (modalEl) renderModalList();
                if (cb) cb();
            })
            .catch(function(err) {
                console.error('Failed to load agents:', err);
            });
    }

    function renderSidebar() {
        var container = document.getElementById('agents-sidebar');
        if (!container) return;

        if (agents.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs px-3 py-2">No agents found</div>';
            return;
        }

        var html = '';
        var filtered = filterAgents(agents);
        var shown = filtered.slice(0, 10);
        for (var i = 0; i < shown.length; i++) {
            var ag = shown[i];
            var badge = ag.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300">G</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300">P</span>';
            html += '<div class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs text-th-text-tertiary hover:bg-surface-raised cursor-pointer truncate" '
                + 'onclick="ClawIDEAgents.openManager(\'' + escapeAttr(ag.scope) + '\', \'' + escapeAttr(ag.file_name) + '\')" '
                + 'title="' + escapeAttr(ag.description || ag.name) + '">'
                + badge + ' '
                + '<span class="truncate">' + escapeHTML(ag.name) + '</span>'
                + '</div>';
        }
        if (filtered.length > 10) {
            html += '<div class="text-th-text-faint text-[10px] px-3 py-1">+' + (filtered.length - 10) + ' more</div>';
        }
        container.innerHTML = html;
    }

    // ── Modal ────────────────────────────────────────────────────

    function openManager(selectScope, selectName) {
        if (modalEl) {
            if (selectScope && selectName) {
                selectAgentByRef(selectScope, selectName);
            }
            return;
        }

        createModal();
        loadAgents(function() {
            if (selectScope && selectName) {
                selectAgentByRef(selectScope, selectName);
            }
        });
    }

    function closeManager() {
        if (modalEl) {
            modalEl.remove();
            modalEl = null;
            selectedAgent = null;
            isCreating = false;
        }
    }

    function createModal() {
        modalEl = document.createElement('div');
        modalEl.id = 'agents-modal';
        modalEl.className = 'fixed inset-0 z-[200] flex items-center justify-center';

        modalEl.innerHTML = ''
            // Backdrop
            + '<div class="absolute inset-0 bg-black/70 backdrop-blur-sm" onclick="ClawIDEAgents.closeManager()"></div>'
            // Modal container
            + '<div class="relative w-[90vw] max-w-5xl h-[80vh] bg-surface-base border border-th-border-strong rounded-xl shadow-2xl flex flex-col overflow-hidden">'
            // Header
            + '  <div class="flex items-center justify-between px-5 py-3 border-b border-th-border">'
            + '    <h2 class="text-base font-semibold text-th-text-primary flex items-center gap-2">'
            + '      <svg class="w-5 h-5 text-accent-text" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z"/></svg>'
            + '      Agents Manager'
            + '    </h2>'
            + '    <button onclick="ClawIDEAgents.closeManager()" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '    </button>'
            + '  </div>'
            // Body: two-pane
            + '  <div class="flex flex-1 min-h-0">'
            // Left pane: list
            + '    <div class="w-72 border-r border-th-border flex flex-col flex-shrink-0 bg-surface-base/60">'
            // Scope tabs
            + '      <div class="flex border-b border-th-border">'
            + '        <button id="agents-tab-all" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEAgents.setFilter(\'all\')">All</button>'
            + '        <button id="agents-tab-project" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEAgents.setFilter(\'project\')">Project</button>'
            + '        <button id="agents-tab-global" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDEAgents.setFilter(\'global\')">Global</button>'
            + '      </div>'
            // Search
            + '      <div class="p-2">'
            + '        <input id="agents-modal-search" type="text" placeholder="Search agents..."'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded px-2.5 py-1.5 text-xs text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border">'
            + '      </div>'
            // List
            + '      <div id="agents-modal-list" class="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5"></div>'
            // New agent button
            + '      <div class="p-2 border-t border-th-border">'
            + '        <button onclick="ClawIDEAgents.newAgent()" class="w-full flex items-center justify-center gap-1.5 px-3 py-2 text-xs text-accent-text hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '          New Agent'
            + '        </button>'
            + '      </div>'
            + '    </div>'
            // Right pane: editor
            + '    <div id="agents-editor-pane" class="flex-1 flex flex-col min-w-0 overflow-hidden">'
            + '      <div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select an agent or create a new one</div>'
            + '    </div>'
            + '  </div>'
            + '</div>';

        document.body.appendChild(modalEl);

        // Search handler
        var searchInput = document.getElementById('agents-modal-search');
        if (searchInput) {
            searchInput.addEventListener('input', function() {
                searchQuery = this.value.trim().toLowerCase();
                renderModalList();
            });
        }

        // Keyboard
        modalEl.addEventListener('keydown', function(e) {
            if (e.key === 'Escape') {
                closeManager();
                e.stopPropagation();
            }
            if ((e.metaKey || e.ctrlKey) && e.key === 's') {
                e.preventDefault();
                saveCurrentAgent();
            }
        });

        updateFilterTabs();
        renderModalList();
    }

    function updateFilterTabs() {
        var tabs = ['all', 'project', 'global'];
        for (var i = 0; i < tabs.length; i++) {
            var tab = document.getElementById('agents-tab-' + tabs[i]);
            if (!tab) continue;
            if (tabs[i] === scopeFilter) {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-accent-text border-b-2 border-accent-border transition-colors';
            } else {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-th-text-faint hover:text-th-text-tertiary transition-colors';
            }
        }
    }

    function setFilter(f) {
        scopeFilter = f;
        updateFilterTabs();
        loadAgents();
    }

    function renderModalList() {
        var container = document.getElementById('agents-modal-list');
        if (!container) return;

        var filtered = filterAgents(agents);

        if (filtered.length === 0) {
            container.innerHTML = '<div class="text-th-text-faint text-xs text-center py-4">No agents found</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < filtered.length; i++) {
            var ag = filtered[i];
            var isSelected = selectedAgent && selectedAgent.scope === ag.scope && selectedAgent.file_name === ag.file_name;
            var badge = ag.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300 flex-shrink-0">Global</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300 flex-shrink-0">Project</span>';
            var desc = ag.description || '';
            if (desc.length > 80) desc = desc.substring(0, 80) + '...';

            html += '<div class="px-2.5 py-2 rounded-lg cursor-pointer transition-colors '
                + (isSelected ? 'bg-accent/20 border border-accent-border/30' : 'hover:bg-surface-raised border border-transparent')
                + '" onclick="ClawIDEAgents.selectAgent(\'' + escapeAttr(ag.scope) + '\', \'' + escapeAttr(ag.file_name) + '\')">'
                + '<div class="flex items-center gap-1.5">'
                + '  <span class="text-sm text-th-text-primary font-medium truncate">' + escapeHTML(ag.name) + '</span>'
                + '  ' + badge
                + '</div>';
            if (desc) {
                html += '<div class="text-[11px] text-th-text-faint mt-0.5 truncate">' + escapeHTML(desc) + '</div>';
            }
            html += '</div>';
        }
        container.innerHTML = html;
    }

    function selectAgentByRef(scope, fileName) {
        for (var i = 0; i < agents.length; i++) {
            if (agents[i].scope === scope && agents[i].file_name === fileName) {
                selectAgent(scope, fileName);
                return;
            }
        }
    }

    function selectAgent(scope, fileName) {
        isCreating = false;
        fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(fileName))
            .then(function(r) {
                if (!r.ok) throw new Error('Failed to load agent');
                return r.json();
            })
            .then(function(ag) {
                selectedAgent = ag;
                renderEditor(ag);
                renderModalList();
            })
            .catch(function(err) {
                console.error('Failed to load agent:', err);
                showToast('Failed to load agent', 'error');
            });
    }

    function newAgent() {
        isCreating = true;
        selectedAgent = {
            name: '',
            description: '',
            model: '',
            allowed_tools: '',
            agent_type: '',
            content: '',
            scope: 'project',
            file_name: ''
        };
        renderEditor(selectedAgent);
        renderModalList();
        setTimeout(function() {
            var nameInput = document.getElementById('agent-field-name');
            if (nameInput) nameInput.focus();
        }, 50);
    }

    // ── Editor Pane ──────────────────────────────────────────────

    function renderEditor(ag) {
        var pane = document.getElementById('agents-editor-pane');
        if (!pane) return;

        pane.innerHTML = ''
            + '<div class="flex-1 overflow-y-auto">'
            // Editor header
            + '<div class="sticky top-0 bg-surface-base z-10 px-5 py-3 border-b border-th-border flex items-center justify-between">'
            + '  <div class="flex items-center gap-2">'
            + '    <h3 class="text-sm font-semibold text-th-text-primary">' + (isCreating ? 'New Agent' : escapeHTML(ag.name)) + '</h3>'
            + '    ' + (ag.scope === 'global'
                ? '<span class="text-[10px] px-1.5 py-0.5 rounded bg-purple-900/50 text-purple-300">Global</span>'
                : '<span class="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/50 text-blue-300">Project</span>')
            + '  </div>'
            + '  <div class="flex items-center gap-2">'
            + '    <button onclick="ClawIDEAgents.saveCurrentAgent()" class="px-3 py-1.5 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded-lg transition-colors flex items-center gap-1">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>'
            + '      Save'
            + '    </button>'
            + (isCreating ? '' : '<button onclick="ClawIDEAgents.moveCurrentAgent()" class="px-3 py-1.5 text-xs text-th-text-muted hover:text-th-text-primary hover:bg-surface-overlay rounded-lg transition-colors flex items-center gap-1">'
                + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"/></svg>'
                + (ag.scope === 'global' ? 'Move to Project' : 'Move to Global')
                + '</button>')
            + (isCreating ? '' : '<button onclick="ClawIDEAgents.deleteCurrentAgent()" class="px-3 py-1.5 text-xs text-red-400 hover:text-th-text-primary hover:bg-red-900/50 rounded-lg transition-colors flex items-center gap-1">'
                + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>'
                + 'Delete'
                + '</button>')
            + '  </div>'
            + '</div>'
            // Form fields
            + '<div class="px-5 py-4 space-y-5">'

            // Basic Information section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
            + '      Basic Information'
            + '    </h4>'
            + '    <div class="grid grid-cols-2 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Name *</label>'
            + '        <input id="agent-field-name" type="text" value="' + escapeAttr(ag.name) + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border"'
            + '               placeholder="my-agent">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Scope</label>'
            + '        <select id="agent-field-scope"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '          <option value="project"' + (ag.scope === 'project' ? ' selected' : '') + '>Project</option>'
            + '          <option value="global"' + (ag.scope === 'global' ? ' selected' : '') + '>Global</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '    <div class="mt-3">'
            + '      <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Description</label>'
            + '      <textarea id="agent-field-description" rows="2"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border resize-y"'
            + '                placeholder="What this agent does and when to use it...">' + escapeHTML(ag.description || '') + '</textarea>'
            + '    </div>'
            + '  </div>'

            // Behavior section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>'
            + '      Behavior'
            + '    </h4>'
            + '    <div class="grid grid-cols-3 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Allowed Tools</label>'
            + '        <input id="agent-field-allowed-tools" type="text" value="' + escapeAttr(ag.allowed_tools || '') + '"'
            + '               class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border"'
            + '               placeholder="Read, Grep, Bash, Write, Edit">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Model Override</label>'
            + '        <select id="agent-field-model"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '          <option value=""' + (!ag.model ? ' selected' : '') + '>Default (inherit)</option>'
            + '          <option value="claude-opus-4-6"' + (ag.model === 'claude-opus-4-6' ? ' selected' : '') + '>Opus 4.6</option>'
            + '          <option value="claude-sonnet-4-6"' + (ag.model === 'claude-sonnet-4-6' ? ' selected' : '') + '>Sonnet 4.6</option>'
            + '          <option value="claude-haiku-4-5-20251001"' + (ag.model === 'claude-haiku-4-5-20251001' ? ' selected' : '') + '>Haiku 4.5</option>'
            + '        </select>'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-th-text-faint uppercase tracking-wider mb-1">Agent Type</label>'
            + '        <select id="agent-field-agent-type"'
            + '                class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">'
            + '          <option value=""' + (!ag.agent_type ? ' selected' : '') + '>Default (general-purpose)</option>'
            + '          <option value="Explore"' + (ag.agent_type === 'Explore' ? ' selected' : '') + '>Explore</option>'
            + '          <option value="Plan"' + (ag.agent_type === 'Plan' ? ' selected' : '') + '>Plan</option>'
            + '          <option value="general-purpose"' + (ag.agent_type === 'general-purpose' ? ' selected' : '') + '>General Purpose</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '    <p class="text-[10px] text-th-text-ghost mt-1.5">Allowed Tools: comma-separated list of tools this agent can access (e.g., Read, Write, Bash, Grep, Glob, Edit)</p>'
            + '  </div>'

            // Agent Instructions section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-th-text-muted uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>'
            + '      Agent Instructions (Markdown)'
            + '    </h4>'
            + '    <textarea id="agent-field-content" rows="16"'
            + '              class="w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border font-mono resize-y"'
            + '              placeholder="# My Agent\\n\\nYou are a specialized agent that...\\n\\nWhen given a task, you should...">' + escapeHTML(ag.content || '') + '</textarea>'
            + '    <p class="text-[10px] text-th-text-ghost mt-1">The system prompt and instructions this agent follows when spawned. Markdown format.</p>'
            + '  </div>'

            + '</div>' // end space-y-5
            + '</div>'; // end overflow-y-auto
    }

    // ── CRUD Operations ──────────────────────────────────────────

    function gatherFormData() {
        return {
            name: val('agent-field-name'),
            description: val('agent-field-description'),
            allowed_tools: val('agent-field-allowed-tools'),
            model: val('agent-field-model'),
            agent_type: val('agent-field-agent-type'),
            content: val('agent-field-content'),
            scope: val('agent-field-scope')
        };
    }

    function saveCurrentAgent() {
        var data = gatherFormData();
        if (!data.name) {
            showToast('Agent name is required', 'error');
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
                showToast('Agent created', 'success');
                isCreating = false;
                var newFileName = data.name.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
                loadAgents(function() {
                    selectAgentByRef(data.scope, newFileName);
                });
            })
            .catch(function(err) {
                showToast('Failed to create: ' + err.message, 'error');
            });
        } else {
            var scope = selectedAgent.scope;
            var fileName = selectedAgent.file_name;
            fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(fileName), {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function() {
                showToast('Agent updated', 'success');
                var newFileName = data.name.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
                loadAgents(function() {
                    selectAgentByRef(data.scope || scope, newFileName || fileName);
                });
            })
            .catch(function(err) {
                showToast('Failed to update: ' + err.message, 'error');
            });
        }
    }

    function deleteCurrentAgent() {
        if (!selectedAgent || isCreating) return;

        if (!confirm('Delete agent "' + selectedAgent.name + '"? This will remove the agent file.')) {
            return;
        }

        fetch(getAPIBase() + '/' + selectedAgent.scope + '/' + encodeURIComponent(selectedAgent.file_name), {
            method: 'DELETE'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Agent deleted', 'success');
            selectedAgent = null;
            var pane = document.getElementById('agents-editor-pane');
            if (pane) {
                pane.innerHTML = '<div class="flex-1 flex items-center justify-center text-th-text-faint text-sm">Select an agent or create a new one</div>';
            }
            loadAgents();
        })
        .catch(function(err) {
            showToast('Failed to delete: ' + err.message, 'error');
        });
    }

    function moveCurrentAgent() {
        if (!selectedAgent || isCreating) return;

        var targetScope = selectedAgent.scope === 'global' ? 'project' : 'global';
        if (!confirm('Move agent "' + selectedAgent.name + '" to ' + targetScope + ' scope?')) {
            return;
        }

        fetch(getAPIBase() + '/' + selectedAgent.scope + '/' + encodeURIComponent(selectedAgent.file_name) + '/move', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ target_scope: targetScope })
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Agent moved to ' + targetScope, 'success');
            var fileName = selectedAgent.file_name;
            loadAgents(function() {
                selectAgentByRef(targetScope, fileName);
            });
        })
        .catch(function(err) {
            showToast('Failed to move: ' + err.message, 'error');
        });
    }

    // ── Helpers ───────────────────────────────────────────────────

    function filterAgents(list) {
        if (!searchQuery) return list;
        return list.filter(function(ag) {
            var hay = (ag.name + ' ' + (ag.description || '')).toLowerCase();
            return hay.indexOf(searchQuery) !== -1;
        });
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
            console.log('[Agents] ' + type + ': ' + msg);
        }
    }

    // ── Init ─────────────────────────────────────────────────────

    document.addEventListener('DOMContentLoaded', initSidebar);

    // Public API
    window.ClawIDEAgents = {
        openManager: openManager,
        closeManager: closeManager,
        selectAgent: selectAgent,
        newAgent: newAgent,
        setFilter: setFilter,
        saveCurrentAgent: saveCurrentAgent,
        deleteCurrentAgent: deleteCurrentAgent,
        moveCurrentAgent: moveCurrentAgent,
        reload: loadAgents
    };
})();
