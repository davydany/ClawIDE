// ClawIDE Skills Manager
// Sidebar skill list + full-screen modal for CRUD management of Claude Code Skills.
(function() {
    'use strict';

    var projectID = '';
    var skills = [];
    var selectedSkill = null;
    var scopeFilter = 'all'; // 'all' | 'project' | 'global'
    var searchQuery = '';
    var isCreating = false;
    var modalEl = null;

    function getAPIBase() {
        return '/projects/' + projectID + '/api/skills';
    }

    // ── Sidebar ──────────────────────────────────────────────────

    function initSidebar() {
        // Extract projectID from the URL: /projects/{id}/...
        var match = window.location.pathname.match(/\/projects\/([^/]+)/);
        if (match) projectID = match[1];
        if (!projectID) return;

        loadSkills();
    }

    function loadSkills(cb) {
        var url = getAPIBase();
        if (scopeFilter !== 'all') {
            url += '?scope=' + scopeFilter;
        }
        fetch(url)
            .then(function(r) { return r.json(); })
            .then(function(data) {
                skills = data || [];
                renderSidebar();
                if (modalEl) renderModalList();
                if (cb) cb();
            })
            .catch(function(err) {
                console.error('Failed to load skills:', err);
            });
    }

    function renderSidebar() {
        var container = document.getElementById('skills-sidebar');
        if (!container) return;

        if (skills.length === 0) {
            container.innerHTML = '<div class="text-gray-500 text-xs px-3 py-2">No skills found</div>';
            return;
        }

        var html = '';
        var filtered = filterSkills(skills);
        // Show up to 10 in sidebar
        var shown = filtered.slice(0, 10);
        for (var i = 0; i < shown.length; i++) {
            var sk = shown[i];
            var badge = sk.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300">G</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300">P</span>';
            html += '<div class="flex items-center gap-1.5 px-3 py-1.5 rounded text-xs text-gray-300 hover:bg-gray-800 cursor-pointer truncate" '
                + 'onclick="ClawIDESkills.openManager(\'' + escapeAttr(sk.scope) + '\', \'' + escapeAttr(sk.dir_name) + '\')" '
                + 'title="' + escapeAttr(sk.description || sk.name) + '">'
                + badge + ' '
                + '<span class="truncate">' + escapeHTML(sk.name) + '</span>'
                + '</div>';
        }
        if (filtered.length > 10) {
            html += '<div class="text-gray-500 text-[10px] px-3 py-1">+' + (filtered.length - 10) + ' more</div>';
        }
        container.innerHTML = html;
    }

    // ── Modal ────────────────────────────────────────────────────

    function openManager(selectScope, selectName) {
        if (modalEl) {
            // Already open, just re-select
            if (selectScope && selectName) {
                selectSkillByRef(selectScope, selectName);
            }
            return;
        }

        createModal();
        loadSkills(function() {
            if (selectScope && selectName) {
                selectSkillByRef(selectScope, selectName);
            }
        });
    }

    function closeManager() {
        if (modalEl) {
            modalEl.remove();
            modalEl = null;
            selectedSkill = null;
            isCreating = false;
        }
    }

    function createModal() {
        modalEl = document.createElement('div');
        modalEl.id = 'skills-modal';
        modalEl.className = 'fixed inset-0 z-[200] flex items-center justify-center';

        modalEl.innerHTML = ''
            // Backdrop
            + '<div class="absolute inset-0 bg-black/70 backdrop-blur-sm" onclick="ClawIDESkills.closeManager()"></div>'
            // Modal container
            + '<div class="relative w-[90vw] max-w-5xl h-[80vh] bg-gray-900 border border-gray-700 rounded-xl shadow-2xl flex flex-col overflow-hidden">'
            // Header
            + '  <div class="flex items-center justify-between px-5 py-3 border-b border-gray-800">'
            + '    <h2 class="text-base font-semibold text-white flex items-center gap-2">'
            + '      <svg class="w-5 h-5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6V4m0 2a2 2 0 100 4m0-4a2 2 0 110 4m-6 8a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4m6 6v10m6-2a2 2 0 100-4m0 4a2 2 0 110-4m0 4v2m0-6V4"/></svg>'
            + '      Skills Manager'
            + '    </h2>'
            + '    <button onclick="ClawIDESkills.closeManager()" class="p-1.5 text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors">'
            + '      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>'
            + '    </button>'
            + '  </div>'
            // Body: two-pane
            + '  <div class="flex flex-1 min-h-0">'
            // Left pane: list
            + '    <div class="w-72 border-r border-gray-800 flex flex-col flex-shrink-0 bg-gray-900/60">'
            // Scope tabs
            + '      <div class="flex border-b border-gray-800">'
            + '        <button id="skills-tab-all" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDESkills.setFilter(\'all\')">All</button>'
            + '        <button id="skills-tab-project" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDESkills.setFilter(\'project\')">Project</button>'
            + '        <button id="skills-tab-global" class="flex-1 px-3 py-2 text-xs font-medium transition-colors" onclick="ClawIDESkills.setFilter(\'global\')">Global</button>'
            + '      </div>'
            // Search
            + '      <div class="p-2">'
            + '        <input id="skills-modal-search" type="text" placeholder="Search skills..."'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500">'
            + '      </div>'
            // List
            + '      <div id="skills-modal-list" class="flex-1 overflow-y-auto px-2 pb-2 space-y-0.5"></div>'
            // New skill button
            + '      <div class="p-2 border-t border-gray-800">'
            + '        <button onclick="ClawIDESkills.newSkill()" class="w-full flex items-center justify-center gap-1.5 px-3 py-2 text-xs text-indigo-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors">'
            + '          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>'
            + '          New Skill'
            + '        </button>'
            + '      </div>'
            + '    </div>'
            // Right pane: editor
            + '    <div id="skills-editor-pane" class="flex-1 flex flex-col min-w-0 overflow-hidden">'
            + '      <div class="flex-1 flex items-center justify-center text-gray-500 text-sm">Select a skill or create a new one</div>'
            + '    </div>'
            + '  </div>'
            + '</div>';

        document.body.appendChild(modalEl);

        // Search handler
        var searchInput = document.getElementById('skills-modal-search');
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
                saveCurrentSkill();
            }
        });

        updateFilterTabs();
        renderModalList();
    }

    function updateFilterTabs() {
        var tabs = ['all', 'project', 'global'];
        for (var i = 0; i < tabs.length; i++) {
            var tab = document.getElementById('skills-tab-' + tabs[i]);
            if (!tab) continue;
            if (tabs[i] === scopeFilter) {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-indigo-400 border-b-2 border-indigo-400 transition-colors';
            } else {
                tab.className = 'flex-1 px-3 py-2 text-xs font-medium text-gray-500 hover:text-gray-300 transition-colors';
            }
        }
    }

    function setFilter(f) {
        scopeFilter = f;
        updateFilterTabs();
        loadSkills();
    }

    function renderModalList() {
        var container = document.getElementById('skills-modal-list');
        if (!container) return;

        var filtered = filterSkills(skills);

        if (filtered.length === 0) {
            container.innerHTML = '<div class="text-gray-500 text-xs text-center py-4">No skills found</div>';
            return;
        }

        var html = '';
        for (var i = 0; i < filtered.length; i++) {
            var sk = filtered[i];
            var isSelected = selectedSkill && selectedSkill.scope === sk.scope && selectedSkill.dir_name === sk.dir_name;
            var badge = sk.scope === 'global'
                ? '<span class="text-[9px] px-1 py-0.5 rounded bg-purple-900/50 text-purple-300 flex-shrink-0">Global</span>'
                : '<span class="text-[9px] px-1 py-0.5 rounded bg-blue-900/50 text-blue-300 flex-shrink-0">Project</span>';
            var desc = sk.description || '';
            if (desc.length > 80) desc = desc.substring(0, 80) + '...';

            html += '<div class="px-2.5 py-2 rounded-lg cursor-pointer transition-colors '
                + (isSelected ? 'bg-indigo-600/20 border border-indigo-500/30' : 'hover:bg-gray-800 border border-transparent')
                + '" onclick="ClawIDESkills.selectSkill(\'' + escapeAttr(sk.scope) + '\', \'' + escapeAttr(sk.dir_name) + '\')">'
                + '<div class="flex items-center gap-1.5">'
                + '  <span class="text-sm text-white font-medium truncate">' + escapeHTML(sk.name) + '</span>'
                + '  ' + badge
                + '</div>';
            if (desc) {
                html += '<div class="text-[11px] text-gray-500 mt-0.5 truncate">' + escapeHTML(desc) + '</div>';
            }
            html += '</div>';
        }
        container.innerHTML = html;
    }

    function selectSkillByRef(scope, dirName) {
        for (var i = 0; i < skills.length; i++) {
            if (skills[i].scope === scope && skills[i].dir_name === dirName) {
                selectSkill(scope, dirName);
                return;
            }
        }
    }

    function selectSkill(scope, dirName) {
        isCreating = false;
        // Fetch full skill content
        fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(dirName))
            .then(function(r) {
                if (!r.ok) throw new Error('Failed to load skill');
                return r.json();
            })
            .then(function(sk) {
                selectedSkill = sk;
                renderEditor(sk);
                renderModalList();
            })
            .catch(function(err) {
                console.error('Failed to load skill:', err);
                showToast('Failed to load skill', 'error');
            });
    }

    function newSkill() {
        isCreating = true;
        selectedSkill = {
            name: '',
            description: '',
            version: '',
            argument_hint: '',
            disable_model_invocation: false,
            user_invocable: true,
            allowed_tools: '',
            model: '',
            effort: '',
            context: '',
            agent: '',
            homepage: '',
            content: '',
            scope: 'project',
            dir_name: ''
        };
        renderEditor(selectedSkill);
        renderModalList();
        // Focus name field
        setTimeout(function() {
            var nameInput = document.getElementById('skill-field-name');
            if (nameInput) nameInput.focus();
        }, 50);
    }

    // ── Editor Pane ──────────────────────────────────────────────

    function renderEditor(sk) {
        var pane = document.getElementById('skills-editor-pane');
        if (!pane) return;

        var userInvocable = sk.user_invocable === undefined || sk.user_invocable === null ? true : sk.user_invocable;

        pane.innerHTML = ''
            + '<div class="flex-1 overflow-y-auto">'
            // Editor header
            + '<div class="sticky top-0 bg-gray-900 z-10 px-5 py-3 border-b border-gray-800 flex items-center justify-between">'
            + '  <div class="flex items-center gap-2">'
            + '    <h3 class="text-sm font-semibold text-white">' + (isCreating ? 'New Skill' : escapeHTML(sk.name)) + '</h3>'
            + '    ' + (sk.scope === 'global'
                ? '<span class="text-[10px] px-1.5 py-0.5 rounded bg-purple-900/50 text-purple-300">Global</span>'
                : '<span class="text-[10px] px-1.5 py-0.5 rounded bg-blue-900/50 text-blue-300">Project</span>')
            + '  </div>'
            + '  <div class="flex items-center gap-2">'
            + '    <button onclick="ClawIDESkills.saveCurrentSkill()" class="px-3 py-1.5 text-xs bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors flex items-center gap-1">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"/></svg>'
            + '      Save'
            + '    </button>'
            + (isCreating ? '' : '<button onclick="ClawIDESkills.deleteCurrentSkill()" class="px-3 py-1.5 text-xs text-red-400 hover:text-white hover:bg-red-900/50 rounded-lg transition-colors flex items-center gap-1">'
                + '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>'
                + 'Delete'
                + '</button>')
            + '  </div>'
            + '</div>'
            // Form fields
            + '<div class="px-5 py-4 space-y-5">'
            // Basic fields section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>'
            + '      Basic Information'
            + '    </h4>'
            + '    <div class="grid grid-cols-2 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Name *</label>'
            + '        <input id="skill-field-name" type="text" value="' + escapeAttr(sk.name) + '"'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"'
            + '               placeholder="my-skill">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Scope</label>'
            + '        <select id="skill-field-scope"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-indigo-500">'
            + '          <option value="project"' + (sk.scope === 'project' ? ' selected' : '') + '>Project</option>'
            + '          <option value="global"' + (sk.scope === 'global' ? ' selected' : '') + '>Global</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '    <div class="mt-3">'
            + '      <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Description</label>'
            + '      <textarea id="skill-field-description" rows="2"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 resize-y"'
            + '                placeholder="When to use this skill...">' + escapeHTML(sk.description || '') + '</textarea>'
            + '    </div>'
            + '    <div class="grid grid-cols-3 gap-3 mt-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Version</label>'
            + '        <input id="skill-field-version" type="text" value="' + escapeAttr(sk.version || '') + '"'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"'
            + '               placeholder="1.0">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Argument Hint</label>'
            + '        <input id="skill-field-argument-hint" type="text" value="' + escapeAttr(sk.argument_hint || '') + '"'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"'
            + '               placeholder="[url] [format]">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Homepage</label>'
            + '        <input id="skill-field-homepage" type="text" value="' + escapeAttr(sk.homepage || '') + '"'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"'
            + '               placeholder="https://github.com/...">'
            + '      </div>'
            + '    </div>'
            + '  </div>'

            // Behavior section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"/><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>'
            + '      Behavior'
            + '    </h4>'
            // Toggles row
            + '    <div class="flex items-center gap-6 mb-3">'
            + '      <label class="flex items-center gap-2 cursor-pointer">'
            + '        <input id="skill-field-user-invocable" type="checkbox"' + (userInvocable ? ' checked' : '')
            + '               class="w-4 h-4 rounded bg-gray-800 border-gray-600 text-indigo-600 focus:ring-indigo-500 focus:ring-offset-gray-900">'
            + '        <span class="text-xs text-gray-300">User Invocable</span>'
            + '        <span class="text-[10px] text-gray-600" title="If disabled, only Claude can invoke this skill (not shown in / menu)">?</span>'
            + '      </label>'
            + '      <label class="flex items-center gap-2 cursor-pointer">'
            + '        <input id="skill-field-disable-model" type="checkbox"' + (sk.disable_model_invocation ? ' checked' : '')
            + '               class="w-4 h-4 rounded bg-gray-800 border-gray-600 text-indigo-600 focus:ring-indigo-500 focus:ring-offset-gray-900">'
            + '        <span class="text-xs text-gray-300">Disable Model Invocation</span>'
            + '        <span class="text-[10px] text-gray-600" title="If enabled, only the user can invoke this skill (Claude cannot auto-invoke)">?</span>'
            + '      </label>'
            + '    </div>'
            // Selects row
            + '    <div class="grid grid-cols-3 gap-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Allowed Tools</label>'
            + '        <input id="skill-field-allowed-tools" type="text" value="' + escapeAttr(sk.allowed_tools || '') + '"'
            + '               class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"'
            + '               placeholder="Read, Grep, Bash">'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Model Override</label>'
            + '        <select id="skill-field-model"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-indigo-500">'
            + '          <option value=""' + (!sk.model ? ' selected' : '') + '>Default (inherit)</option>'
            + '          <option value="claude-opus-4-6"' + (sk.model === 'claude-opus-4-6' ? ' selected' : '') + '>Opus 4.6</option>'
            + '          <option value="claude-sonnet-4-6"' + (sk.model === 'claude-sonnet-4-6' ? ' selected' : '') + '>Sonnet 4.6</option>'
            + '          <option value="claude-haiku-4-5-20251001"' + (sk.model === 'claude-haiku-4-5-20251001' ? ' selected' : '') + '>Haiku 4.5</option>'
            + '        </select>'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Effort</label>'
            + '        <select id="skill-field-effort"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-indigo-500">'
            + '          <option value=""' + (!sk.effort ? ' selected' : '') + '>Default</option>'
            + '          <option value="low"' + (sk.effort === 'low' ? ' selected' : '') + '>Low</option>'
            + '          <option value="medium"' + (sk.effort === 'medium' ? ' selected' : '') + '>Medium</option>'
            + '          <option value="high"' + (sk.effort === 'high' ? ' selected' : '') + '>High</option>'
            + '          <option value="max"' + (sk.effort === 'max' ? ' selected' : '') + '>Max (Opus only)</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '    <div class="grid grid-cols-2 gap-3 mt-3">'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Context</label>'
            + '        <select id="skill-field-context"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-indigo-500">'
            + '          <option value=""' + (!sk.context ? ' selected' : '') + '>Default (inline)</option>'
            + '          <option value="fork"' + (sk.context === 'fork' ? ' selected' : '') + '>Fork (isolated subagent)</option>'
            + '        </select>'
            + '      </div>'
            + '      <div>'
            + '        <label class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Agent Type</label>'
            + '        <select id="skill-field-agent"'
            + '                class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:outline-none focus:border-indigo-500">'
            + '          <option value=""' + (!sk.agent ? ' selected' : '') + '>Default (general-purpose)</option>'
            + '          <option value="Explore"' + (sk.agent === 'Explore' ? ' selected' : '') + '>Explore</option>'
            + '          <option value="Plan"' + (sk.agent === 'Plan' ? ' selected' : '') + '>Plan</option>'
            + '          <option value="general-purpose"' + (sk.agent === 'general-purpose' ? ' selected' : '') + '>General Purpose</option>'
            + '        </select>'
            + '      </div>'
            + '    </div>'
            + '  </div>'

            // Content section
            + '  <div>'
            + '    <h4 class="text-xs font-semibold text-gray-400 uppercase tracking-wider mb-3 flex items-center gap-1.5">'
            + '      <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>'
            + '      Skill Content (Markdown)'
            + '    </h4>'
            + '    <textarea id="skill-field-content" rows="12"'
            + '              class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500 font-mono resize-y"'
            + '              placeholder="# My Skill\\n\\nInstructions for Claude...">' + escapeHTML(sk.content || '') + '</textarea>'
            + '    <p class="text-[10px] text-gray-600 mt-1">Markdown instructions that Claude follows when this skill is invoked. Use $ARGUMENTS for passed arguments.</p>'
            + '  </div>'

            + '</div>' // end space-y-5
            + '</div>'; // end overflow-y-auto
    }

    // ── CRUD Operations ──────────────────────────────────────────

    function gatherFormData() {
        var data = {
            name: val('skill-field-name'),
            description: val('skill-field-description'),
            version: val('skill-field-version'),
            argument_hint: val('skill-field-argument-hint'),
            homepage: val('skill-field-homepage'),
            allowed_tools: val('skill-field-allowed-tools'),
            model: val('skill-field-model'),
            effort: val('skill-field-effort'),
            context: val('skill-field-context'),
            agent: val('skill-field-agent'),
            content: val('skill-field-content'),
            scope: val('skill-field-scope'),
            disable_model_invocation: checked('skill-field-disable-model'),
            user_invocable: checked('skill-field-user-invocable')
        };
        return data;
    }

    function saveCurrentSkill() {
        var data = gatherFormData();
        if (!data.name) {
            showToast('Skill name is required', 'error');
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
                showToast('Skill created', 'success');
                isCreating = false;
                var newDirName = data.name.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
                loadSkills(function() {
                    selectSkillByRef(data.scope, newDirName);
                });
            })
            .catch(function(err) {
                showToast('Failed to create: ' + err.message, 'error');
            });
        } else {
            var scope = selectedSkill.scope;
            var dirName = selectedSkill.dir_name;
            fetch(getAPIBase() + '/' + scope + '/' + encodeURIComponent(dirName), {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            })
            .then(function(r) {
                if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
                return r.json();
            })
            .then(function() {
                showToast('Skill updated', 'success');
                var newDirName = data.name.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/-+/g, '-').replace(/^-|-$/g, '');
                loadSkills(function() {
                    selectSkillByRef(data.scope || scope, newDirName || dirName);
                });
            })
            .catch(function(err) {
                showToast('Failed to update: ' + err.message, 'error');
            });
        }
    }

    function deleteCurrentSkill() {
        if (!selectedSkill || isCreating) return;

        if (!confirm('Delete skill "' + selectedSkill.name + '"? This will remove the entire skill directory.')) {
            return;
        }

        fetch(getAPIBase() + '/' + selectedSkill.scope + '/' + encodeURIComponent(selectedSkill.dir_name), {
            method: 'DELETE'
        })
        .then(function(r) {
            if (!r.ok) return r.text().then(function(t) { throw new Error(t); });
            return r.json();
        })
        .then(function() {
            showToast('Skill deleted', 'success');
            selectedSkill = null;
            var pane = document.getElementById('skills-editor-pane');
            if (pane) {
                pane.innerHTML = '<div class="flex-1 flex items-center justify-center text-gray-500 text-sm">Select a skill or create a new one</div>';
            }
            loadSkills();
        })
        .catch(function(err) {
            showToast('Failed to delete: ' + err.message, 'error');
        });
    }

    // ── Helpers ───────────────────────────────────────────────────

    function filterSkills(list) {
        if (!searchQuery) return list;
        return list.filter(function(sk) {
            var hay = (sk.name + ' ' + (sk.description || '')).toLowerCase();
            return hay.indexOf(searchQuery) !== -1;
        });
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
            console.log('[Skills] ' + type + ': ' + msg);
        }
    }

    // ── Init ─────────────────────────────────────────────────────

    document.addEventListener('DOMContentLoaded', initSidebar);

    // Public API
    window.ClawIDESkills = {
        openManager: openManager,
        closeManager: closeManager,
        selectSkill: selectSkill,
        newSkill: newSkill,
        setFilter: setFilter,
        saveCurrentSkill: saveCurrentSkill,
        deleteCurrentSkill: deleteCurrentSkill,
        reload: loadSkills
    };
})();
