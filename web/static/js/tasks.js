/*
 * ClawIDE Tasks — Kanban board UI backed by the /api/tasks/* endpoints.
 *
 * The board is rendered from JSON returned by GET /api/tasks/board. Drag-drop sends a POST
 * /api/tasks/{id}/move request and optimistically updates the DOM. The task detail modal calls
 * /api/tasks/{id}/ask-ai to run a CLI AI provider and appends the response as a comment.
 *
 * This file deliberately avoids any framework — plain DOM + fetch + HTML5 drag API — to stay
 * consistent with the rest of the web/ codebase (no React, no jQuery).
 */
(function() {
    'use strict';

    const API = '/api/tasks';
    const AI_API = '/api/ai/providers';

    // ---------------- State ----------------
    let projectID = '';
    let scope = 'project';        // 'project' | 'global'
    let board = null;             // latest fetched board JSON
    let providers = [];           // populated from GET /api/ai/providers
    let currentTaskID = null;     // ID of task shown in modal
    let askAIController = null;   // AbortController for in-flight Ask AI request

    // ---------------- Scope + HTTP helpers ----------------

    function qs() {
        if (scope === 'project' && projectID) {
            return '?project_id=' + encodeURIComponent(projectID);
        }
        return '';
    }

    async function apiFetch(path, opts) {
        const res = await fetch(path, Object.assign({
            headers: { 'Content-Type': 'application/json' }
        }, opts || {}));
        if (!res.ok) {
            const text = await res.text().catch(function() { return res.statusText; });
            throw new Error(res.status + ': ' + (text || res.statusText));
        }
        if (res.status === 204) return null;
        return res.json();
    }

    // ---------------- Board loading + rendering ----------------

    async function loadBoard() {
        const root = document.getElementById('tasks-board');
        if (!root) return;
        root.innerHTML = '<div class="flex items-center justify-center h-full text-th-text-faint text-sm">Loading tasks...</div>';
        try {
            board = await apiFetch(API + '/board' + qs());
            renderBoard();
        } catch (err) {
            root.innerHTML = '<div class="flex items-center justify-center h-full text-red-400 text-sm p-4 text-center">Failed to load board: ' + escapeHTML(err.message) + '</div>';
        }
    }

    async function loadProviders() {
        try {
            providers = await apiFetch(AI_API);
        } catch (err) {
            providers = [];
            console.warn('tasks: failed to load AI providers', err);
        }
    }

    async function loadMetrics() {
        var el = document.getElementById('tasks-metrics');
        if (!el || !projectID) return;
        try {
            var data = await apiFetch(API + '/metrics?project_id=' + encodeURIComponent(projectID) + '&days=7');
            renderMetrics(el, data || []);
        } catch (_) {
            el.innerHTML = '';
        }
    }

    function renderMetrics(el, days) {
        // Sum totals for the last 7 days and show today's numbers.
        var todayStr = new Date().toISOString().slice(0, 10);
        var todayData = null;
        var totalCreated = 0;
        var totalClosed = 0;
        for (var i = 0; i < days.length; i++) {
            totalCreated += days[i].created || 0;
            totalClosed += days[i].closed || 0;
            if (days[i].date === todayStr) todayData = days[i];
        }
        var tc = todayData ? todayData.created : 0;
        var td = todayData ? todayData.closed : 0;
        el.innerHTML =
            '<span class="text-[10px] text-th-text-faint">Today:</span> ' +
            '<span class="text-[10px] text-emerald-400 font-medium">+' + tc + '</span> ' +
            '<span class="text-[10px] text-blue-400 font-medium">-' + td + '</span>' +
            '<span class="text-[10px] text-th-text-ghost mx-1">|</span>' +
            '<span class="text-[10px] text-th-text-faint">7d:</span> ' +
            '<span class="text-[10px] text-emerald-400/70">+' + totalCreated + '</span> ' +
            '<span class="text-[10px] text-blue-400/70">-' + totalClosed + '</span>';
    }

    function renderBoard() {
        var root = document.getElementById('tasks-board');
        if (!root || !board) return;
        root.innerHTML = '';

        var track = document.createElement('div');
        track.className = 'flex items-start gap-3 h-full p-4 min-w-max';
        track.id = 'tasks-track';

        // Column drag-over listener on the track (detects insertion point between columns).
        track.addEventListener('dragover', onColumnDragOver);
        track.addEventListener('dragleave', onColumnDragLeave);
        track.addEventListener('drop', onColumnDrop);

        (board.columns || []).forEach(function(col) { track.appendChild(renderColumn(col)); });
        root.appendChild(track);
    }

    function renderColumn(col) {
        var wrap = document.createElement('div');
        wrap.className = 'flex flex-col w-72 max-h-full rounded-lg bg-surface-base/60 border border-th-border transition-all duration-200';
        wrap.dataset.columnSlug = col.id;
        // NOT draggable on the wrapper — only the header is draggable, so task card drags work.

        var header = document.createElement('div');
        header.className = 'flex items-center gap-1.5 px-3 py-2 border-b border-th-border cursor-grab';
        header.draggable = true;
        header.dataset.columnSlug = col.id;
        // Column drag events live on the header. We use setDragImage to show the full column as
        // the ghost, so it looks like you're dragging the whole thing even though only the header
        // is the drag source.
        header.addEventListener('dragstart', function(e) {
            onColumnDragStart.call(wrap, e);
        });
        header.addEventListener('dragend', function(e) {
            onColumnDragEnd.call(wrap, e);
        });

        // Drag handle (grip icon) — visual affordance for dragging.
        var grip = document.createElement('span');
        grip.className = 'text-th-text-ghost hover:text-th-text-muted transition-colors flex-shrink-0';
        grip.title = 'Drag to reorder';
        grip.innerHTML = '<svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="currentColor"><circle cx="9" cy="5" r="1.5"/><circle cx="15" cy="5" r="1.5"/><circle cx="9" cy="12" r="1.5"/><circle cx="15" cy="12" r="1.5"/><circle cx="9" cy="19" r="1.5"/><circle cx="15" cy="19" r="1.5"/></svg>';
        header.appendChild(grip);

        var titleSpan = document.createElement('span');
        titleSpan.className = 'text-xs font-semibold uppercase text-th-text-primary flex-1 truncate';
        titleSpan.textContent = col.title;
        header.appendChild(titleSpan);

        var countSpan = document.createElement('span');
        countSpan.className = 'text-[10px] text-th-text-faint';
        countSpan.textContent = countTasksInColumn(col);
        header.appendChild(countSpan);

        // Add task button
        var addBtn = document.createElement('button');
        addBtn.className = 'p-1 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors';
        addBtn.title = 'Add task';
        addBtn.innerHTML = '<svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/></svg>';
        addBtn.addEventListener('click', function(e) { e.stopPropagation(); promptNewTask(col.id); });
        header.appendChild(addBtn);

        wrap.appendChild(header);

        const body = document.createElement('div');
        body.className = 'flex-1 overflow-y-auto p-2 space-y-3';
        body.dataset.columnSlug = col.id;
        // Drop target for dragged tasks (not columns — those are handled on the track).
        body.addEventListener('dragover', onTaskDragOver);
        body.addEventListener('drop', onTaskDrop);
        body.addEventListener('dragleave', onTaskDragLeave);

        (col.groups || []).forEach(function(group) {
            if (group.title) {
                const gh = document.createElement('div');
                gh.className = 'text-[11px] uppercase tracking-wide text-th-text-faint font-semibold mt-1 mb-0.5 px-1';
                gh.textContent = group.title;
                body.appendChild(gh);
            }
            (group.tasks || []).forEach(function(task) {
                body.appendChild(renderTaskCard(task, col.id, group.title || ''));
            });
        });

        if (!(col.groups || []).some(function(g) { return (g.tasks || []).length > 0; })) {
            const empty = document.createElement('div');
            empty.className = 'text-center text-th-text-faint text-xs italic py-6';
            empty.textContent = 'No tasks';
            body.appendChild(empty);
        }

        wrap.appendChild(body);
        return wrap;
    }

    function countTasksInColumn(col) {
        let n = 0;
        (col.groups || []).forEach(function(g) { n += (g.tasks || []).length; });
        return n;
    }

    function renderTaskCard(task, columnSlug, groupTitle) {
        const card = document.createElement('div');
        card.className = 'bg-surface-raised border border-th-border-strong rounded px-3 py-2 cursor-pointer hover:border-emerald-400/50 transition-colors';
        card.draggable = true;
        card.dataset.taskId = task.id;
        card.dataset.columnSlug = columnSlug;
        card.dataset.groupTitle = groupTitle;

        const title = document.createElement('div');
        title.className = 'text-sm font-medium text-th-text-primary line-clamp-2';
        title.textContent = task.title || '(untitled)';
        card.appendChild(title);

        if (task.description) {
            const desc = document.createElement('div');
            desc.className = 'mt-1 text-xs text-th-text-muted line-clamp-2';
            desc.textContent = task.description;
            card.appendChild(desc);
        }

        const meta = document.createElement('div');
        meta.className = 'mt-1.5 flex items-center gap-2 text-[10px] text-th-text-faint';
        const commentCount = (task.comments || []).length;
        if (commentCount > 0) {
            meta.innerHTML = '<span class="inline-flex items-center gap-0.5">' +
                '<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"/></svg>' +
                commentCount +
                '</span>';
        }
        card.appendChild(meta);

        card.addEventListener('click', function() { openTaskModal(task.id); });
        card.addEventListener('dragstart', onDragStart);
        card.addEventListener('dragend', onDragEnd);
        return card;
    }

    // ---------------- Drag and drop (tasks + columns) ----------------

    var dragType = null;          // 'task' | 'column'
    var dragTaskID = null;
    var dragSourceColumn = null;
    var dragColumnSlug = null;    // slug of column being dragged
    var columnDropIndex = -1;     // where to insert the dragged column

    // --- Task card drag ---

    function onDragStart(e) {
        dragType = 'task';
        dragTaskID = this.dataset.taskId;
        dragSourceColumn = this.dataset.columnSlug;
        this.classList.add('opacity-40');
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/plain', dragTaskID);
        e.stopPropagation(); // prevent column drag from firing
    }

    function onDragEnd() {
        this.classList.remove('opacity-40');
        dragType = null;
        dragTaskID = null;
        dragSourceColumn = null;
        clearTaskDropHighlights();
    }

    function clearTaskDropHighlights() {
        document.querySelectorAll('#tasks-board .task-drop-highlight').forEach(function(el) {
            el.classList.remove('task-drop-highlight', 'bg-emerald-500/5');
        });
    }

    function onTaskDragOver(e) {
        if (dragType !== 'task') return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';
        this.classList.add('task-drop-highlight', 'bg-emerald-500/5');
    }

    function onTaskDragLeave() {
        this.classList.remove('task-drop-highlight', 'bg-emerald-500/5');
    }

    async function onTaskDrop(e) {
        e.preventDefault();
        this.classList.remove('task-drop-highlight', 'bg-emerald-500/5');
        if (dragType !== 'task' || !dragTaskID) return;
        var destColumnSlug = this.dataset.columnSlug;
        var cards = Array.from(this.querySelectorAll('[data-task-id]'));
        var insertIndex = cards.length;
        for (var i = 0; i < cards.length; i++) {
            var rect = cards[i].getBoundingClientRect();
            if (e.clientY < rect.top + rect.height / 2) {
                insertIndex = i;
                break;
            }
        }
        try {
            await apiFetch(API + '/' + encodeURIComponent(dragTaskID) + '/move' + qs(), {
                method: 'POST',
                body: JSON.stringify({
                    to_column: destColumnSlug,
                    to_group: '',
                    to_index: insertIndex
                })
            });
            await loadBoard();
        } catch (err) {
            ClawIDEDialog.confirm('Move Failed', err.message, { confirmLabel: 'OK' });
            await loadBoard();
        }
    }

    // --- Column drag ---

    function onColumnDragStart(e) {
        // `this` is the column wrapper (bound via .call in the header listener).
        dragType = 'column';
        dragColumnSlug = this.dataset.columnSlug;
        e.dataTransfer.effectAllowed = 'move';
        e.dataTransfer.setData('text/x-column', dragColumnSlug);
        // Use the full column element as the drag ghost so it looks natural.
        e.dataTransfer.setDragImage(this, 40, 20);
        // Fade the source after the ghost is captured.
        var self = this;
        requestAnimationFrame(function() {
            self.style.opacity = '0.35';
            self.style.transform = 'scale(0.95)';
        });
        // Stop propagation so the track's dragover doesn't also fire dragstart on parent elements.
        e.stopPropagation();
    }

    function onColumnDragEnd() {
        this.style.opacity = '';
        this.style.transform = '';
        dragType = null;
        dragColumnSlug = null;
        columnDropIndex = -1;
        clearColumnDropIndicators();
    }

    function clearColumnDropIndicators() {
        document.querySelectorAll('.col-drop-indicator').forEach(function(el) { el.remove(); });
        document.querySelectorAll('#tasks-track > [data-column-slug]').forEach(function(el) {
            el.style.marginLeft = '';
            el.style.marginRight = '';
        });
    }

    function onColumnDragOver(e) {
        if (dragType !== 'column') return;
        e.preventDefault();
        e.dataTransfer.dropEffect = 'move';

        var track = document.getElementById('tasks-track');
        if (!track) return;
        var columns = Array.from(track.querySelectorAll(':scope > [data-column-slug]'));
        if (columns.length < 2) return;

        // Find the insertion index based on cursor X position.
        var newDropIndex = columns.length;
        for (var i = 0; i < columns.length; i++) {
            var rect = columns[i].getBoundingClientRect();
            var midX = rect.left + rect.width / 2;
            if (e.clientX < midX) {
                newDropIndex = i;
                break;
            }
        }

        if (newDropIndex === columnDropIndex) return; // no change
        columnDropIndex = newDropIndex;

        // Show insertion indicator by adding a visible left/right margin on the target column.
        clearColumnDropIndicators();
        // Create a thin vertical indicator line.
        var indicator = document.createElement('div');
        indicator.className = 'col-drop-indicator flex-shrink-0 w-1 rounded-full bg-emerald-400 self-stretch transition-all duration-150';
        indicator.style.minHeight = '60px';

        if (newDropIndex < columns.length) {
            track.insertBefore(indicator, columns[newDropIndex]);
        } else {
            track.appendChild(indicator);
        }
    }

    function onColumnDragLeave(e) {
        // Only clear if leaving the track entirely (not entering a child).
        var track = document.getElementById('tasks-track');
        if (track && !track.contains(e.relatedTarget)) {
            clearColumnDropIndicators();
            columnDropIndex = -1;
        }
    }

    async function onColumnDrop(e) {
        if (dragType !== 'column' || !dragColumnSlug) return;
        e.preventDefault();
        clearColumnDropIndicators();

        // Compute the final index. If dragging rightward, account for the removal shifting indices.
        var track = document.getElementById('tasks-track');
        var columns = Array.from(track.querySelectorAll(':scope > [data-column-slug]'));
        var fromIdx = -1;
        for (var i = 0; i < columns.length; i++) {
            if (columns[i].dataset.columnSlug === dragColumnSlug) { fromIdx = i; break; }
        }
        var toIdx = columnDropIndex;
        if (toIdx < 0) toIdx = columns.length - 1;
        // Adjust for the "remove then insert" semantic: if moving right, the target index shifts
        // down by one after the source is removed.
        if (fromIdx < toIdx) toIdx--;
        if (fromIdx === toIdx) return; // no-op

        try {
            await moveColumn(dragColumnSlug, toIdx);
        } catch (_) {
            // moveColumn already shows an error dialog
        }
    }

    // ---------------- Task creation / column ops ----------------

    async function promptNewTask(columnSlug) {
        var result = await ClawIDEDialog.form('New Task', [
            { key: 'title', label: 'Title', type: 'text', placeholder: 'Task title', required: true },
            { key: 'description', label: 'Description', type: 'textarea', placeholder: 'Optional description (markdown)' }
        ]);
        if (!result) return;
        try {
            await apiFetch(API + qs(), {
                method: 'POST',
                body: JSON.stringify({
                    column: columnSlug,
                    group: '',
                    title: result.title.trim(),
                    description: result.description.trim()
                })
            });
            await loadBoard();
            await loadMetrics();
        } catch (err) {
            await ClawIDEDialog.confirm('Error', 'Failed to create task: ' + err.message, { confirmLabel: 'OK' });
        }
    }

    async function promptNewColumn() {
        var title = await ClawIDEDialog.prompt('New Column', 'Column title', '');
        if (!title || !title.trim()) return;
        try {
            await apiFetch(API + '/columns' + qs(), {
                method: 'POST',
                body: JSON.stringify({ title: title.trim() })
            });
            await loadBoard();
        } catch (err) {
            await ClawIDEDialog.confirm('Error', 'Failed to create column: ' + err.message, { confirmLabel: 'OK' });
        }
    }

    // ---------------- Task detail modal ----------------

    function findTaskByID(id) {
        if (!board || !board.columns) return null;
        for (let ci = 0; ci < board.columns.length; ci++) {
            const col = board.columns[ci];
            for (let gi = 0; gi < (col.groups || []).length; gi++) {
                const g = col.groups[gi];
                for (let ti = 0; ti < (g.tasks || []).length; ti++) {
                    if (g.tasks[ti].id === id) {
                        return { task: g.tasks[ti], column: col };
                    }
                }
            }
        }
        return null;
    }

    function openTaskModal(taskID) {
        currentTaskID = taskID;
        const found = findTaskByID(taskID);
        if (!found) return;
        const task = found.task;
        const root = document.getElementById('tasks-modal-root');
        if (!root) return;

        const modal = document.createElement('div');
        modal.className = 'fixed inset-0 z-50 flex items-start justify-center bg-black/60 p-4 overflow-y-auto';
        modal.addEventListener('click', function(e) {
            if (e.target === modal) closeTaskModal();
        });

        const panel = document.createElement('div');
        panel.className = 'w-full max-w-2xl mt-16 bg-surface-base rounded-lg border border-th-border shadow-xl overflow-hidden';
        panel.addEventListener('click', function(e) { e.stopPropagation(); });

        panel.innerHTML = `
            <div class="flex items-center gap-2 px-4 py-3 border-b border-th-border">
                <input id="tasks-modal-title" type="text"
                       class="flex-1 bg-transparent text-base font-medium text-th-text-primary focus:outline-none"
                       value="${escapeAttr(task.title || '')}">
                <button id="tasks-modal-delete" class="p-1.5 text-red-400 hover:bg-red-900/30 rounded transition-colors" title="Delete task">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><polyline points="3 6 5 6 21 6"/><path d="M19 6v14a2 2 0 01-2 2H7a2 2 0 01-2-2V6m3 0V4a2 2 0 012-2h4a2 2 0 012 2v2"/></svg>
                </button>
                <button id="tasks-modal-close" class="p-1.5 text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded transition-colors" title="Close">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>
                </button>
            </div>
            <div class="p-4 space-y-4 max-h-[70vh] overflow-y-auto">
                <div>
                    <label class="block text-xs uppercase text-th-text-faint mb-1">Description</label>
                    <textarea id="tasks-modal-desc"
                              class="w-full min-h-24 bg-surface-raised border border-th-border-strong rounded px-3 py-2 text-sm text-th-text-primary font-mono focus:outline-none focus:border-accent-border"
                              placeholder="Markdown description...">${escapeHTML(task.description || '')}</textarea>
                    <div class="mt-1 flex justify-end">
                        <button id="tasks-modal-save" class="px-3 py-1 text-xs bg-accent hover:bg-accent-hover text-th-text-primary rounded">Save</button>
                    </div>
                </div>
                <div>
                    <label class="block text-xs uppercase text-th-text-faint mb-1">Comments</label>
                    <div id="tasks-modal-comments" class="space-y-2"></div>
                    <div class="mt-2 flex gap-2">
                        <input id="tasks-modal-new-comment" type="text" placeholder="Add a comment..."
                               class="flex-1 bg-surface-raised border border-th-border-strong rounded px-3 py-1.5 text-sm text-th-text-primary focus:outline-none focus:border-accent-border">
                        <button id="tasks-modal-add-comment" class="px-3 py-1.5 text-xs bg-surface-raised hover:bg-surface-overlay text-th-text-primary rounded border border-th-border-strong">Add</button>
                    </div>
                </div>
                <div class="border-t border-th-border pt-4">
                    <label class="block text-xs uppercase text-th-text-faint mb-2">Ask AI</label>
                    <div class="flex items-center gap-2 mb-2">
                        <select id="tasks-modal-ai-provider" class="bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-xs text-th-text-primary focus:outline-none"></select>
                        <select id="tasks-modal-ai-model" class="bg-surface-raised border border-th-border-strong rounded px-2 py-1 text-xs text-th-text-primary focus:outline-none"></select>
                    </div>
                    <textarea id="tasks-modal-ai-prompt"
                              class="w-full min-h-16 bg-surface-raised border border-th-border-strong rounded px-3 py-2 text-sm text-th-text-primary focus:outline-none focus:border-accent-border"
                              placeholder="Ask the AI to research something, draft code, etc."></textarea>
                    <div class="mt-2 flex items-center gap-2">
                        <button id="tasks-modal-ai-submit" class="px-3 py-1.5 text-xs bg-emerald-600 hover:bg-emerald-500 text-th-text-primary rounded">Ask</button>
                        <button id="tasks-modal-ai-cancel" style="display:none" class="px-3 py-1.5 text-xs bg-red-600 hover:bg-red-500 text-th-text-primary rounded">Cancel</button>
                        <span id="tasks-modal-ai-status" class="text-xs text-th-text-muted"></span>
                    </div>
                </div>
            </div>
        `;

        modal.appendChild(panel);
        root.innerHTML = '';
        root.appendChild(modal);

        // Wire up controls.
        panel.querySelector('#tasks-modal-close').addEventListener('click', closeTaskModal);
        panel.querySelector('#tasks-modal-delete').addEventListener('click', async function() {
            var ok = await ClawIDEDialog.confirm('Delete Task', 'Are you sure you want to delete this task?', { destructive: true, confirmLabel: 'Delete' });
            if (!ok) return;
            try {
                await apiFetch(API + '/' + encodeURIComponent(taskID) + qs(), { method: 'DELETE' });
                closeTaskModal();
                await loadBoard();
            } catch (err) {
                await ClawIDEDialog.confirm('Error', 'Failed to delete task: ' + err.message, { confirmLabel: 'OK' });
            }
        });
        panel.querySelector('#tasks-modal-save').addEventListener('click', async function() {
            const newTitle = panel.querySelector('#tasks-modal-title').value;
            const newDesc = panel.querySelector('#tasks-modal-desc').value;
            try {
                await apiFetch(API + '/' + encodeURIComponent(taskID) + qs(), {
                    method: 'PUT',
                    body: JSON.stringify({ title: newTitle, description: newDesc })
                });
                await loadBoard();
            } catch (err) {
                await ClawIDEDialog.confirm('Error', 'Failed to save task: ' + err.message, { confirmLabel: 'OK' });
            }
        });
        panel.querySelector('#tasks-modal-add-comment').addEventListener('click', async function() {
            const input = panel.querySelector('#tasks-modal-new-comment');
            const body = input.value.trim();
            if (!body) return;
            try {
                await apiFetch(API + '/' + encodeURIComponent(taskID) + '/comments' + qs(), {
                    method: 'POST',
                    body: JSON.stringify({ body: body })
                });
                input.value = '';
                // Reload board to refresh comments, then refresh the modal's comment list.
                await loadBoard();
                refreshModalComments(taskID);
            } catch (err) {
                await ClawIDEDialog.confirm('Error', 'Failed to add comment: ' + err.message, { confirmLabel: 'OK' });
            }
        });
        wireAskAI(panel, taskID);
        refreshModalComments(taskID);
        populateProviderDropdowns(panel);
    }

    function closeTaskModal() {
        currentTaskID = null;
        if (askAIController) {
            askAIController.abort();
            askAIController = null;
        }
        const root = document.getElementById('tasks-modal-root');
        if (root) root.innerHTML = '';
    }

    function refreshModalComments(taskID) {
        const found = findTaskByID(taskID);
        const container = document.getElementById('tasks-modal-comments');
        if (!container) return;
        container.innerHTML = '';
        if (!found || !found.task.comments || found.task.comments.length === 0) {
            container.innerHTML = '<div class="text-xs text-th-text-faint italic">No comments yet.</div>';
            return;
        }
        found.task.comments.forEach(function(c) {
            const row = document.createElement('div');
            const isAI = (c.author || '').indexOf('AI') === 0;
            row.className = 'rounded border px-3 py-2 text-sm ' +
                (isAI ? 'bg-emerald-900/20 border-emerald-800/40 text-th-text-primary' : 'bg-surface-raised border-th-border-strong text-th-text-primary');
            const meta = document.createElement('div');
            meta.className = 'text-[10px] uppercase text-th-text-faint mb-1';
            meta.textContent = (c.author || 'unknown') + ' · ' + formatTimestamp(c.timestamp);
            const body = document.createElement('div');
            body.className = 'whitespace-pre-wrap';
            body.textContent = c.body || '';
            row.appendChild(meta);
            row.appendChild(body);
            container.appendChild(row);
        });
    }

    function populateProviderDropdowns(panel) {
        const providerSelect = panel.querySelector('#tasks-modal-ai-provider');
        const modelSelect = panel.querySelector('#tasks-modal-ai-model');
        providerSelect.innerHTML = '';
        const installed = providers.filter(function(p) { return p.installed; });
        if (installed.length === 0) {
            const opt = document.createElement('option');
            opt.value = '';
            opt.textContent = 'No AI CLI installed';
            providerSelect.appendChild(opt);
            providerSelect.disabled = true;
            modelSelect.disabled = true;
            const btn = panel.querySelector('#tasks-modal-ai-submit');
            if (btn) btn.disabled = true;
            return;
        }
        installed.forEach(function(p) {
            const opt = document.createElement('option');
            opt.value = p.id;
            opt.textContent = p.display_name;
            providerSelect.appendChild(opt);
        });
        function refreshModels() {
            modelSelect.innerHTML = '';
            const p = providers.find(function(x) { return x.id === providerSelect.value; });
            if (!p) return;
            (p.models || []).forEach(function(m) {
                const opt = document.createElement('option');
                opt.value = m.id;
                opt.textContent = m.display_name;
                modelSelect.appendChild(opt);
            });
            if (p.default_model) modelSelect.value = p.default_model;
        }
        providerSelect.addEventListener('change', refreshModels);
        refreshModels();
    }

    function wireAskAI(panel, taskID) {
        const submit = panel.querySelector('#tasks-modal-ai-submit');
        const cancel = panel.querySelector('#tasks-modal-ai-cancel');
        const status = panel.querySelector('#tasks-modal-ai-status');
        const promptEl = panel.querySelector('#tasks-modal-ai-prompt');
        const providerSel = panel.querySelector('#tasks-modal-ai-provider');
        const modelSel = panel.querySelector('#tasks-modal-ai-model');

        submit.addEventListener('click', async function() {
            const prompt = promptEl.value.trim();
            if (!prompt) return;
            const provider = providerSel.value;
            const model = modelSel.value;
            if (!provider || !model) {
                ClawIDEDialog.confirm('Missing Selection', 'Choose a provider and model first.', { confirmLabel: 'OK' });
                return;
            }
            submit.style.display = 'none';
            cancel.style.display = '';
            status.textContent = 'Running ' + provider + '/' + model + '...';
            askAIController = new AbortController();
            try {
                const res = await fetch(API + '/' + encodeURIComponent(taskID) + '/ask-ai' + qs(), {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ provider: provider, model: model, prompt: prompt }),
                    signal: askAIController.signal
                });
                if (!res.ok) {
                    const errText = await res.text();
                    throw new Error(res.status + ': ' + errText);
                }
                const data = await res.json();
                status.textContent = 'Done in ' + (data.duration_ms || 0) + 'ms';
                promptEl.value = '';
                await loadBoard();
                refreshModalComments(taskID);
            } catch (err) {
                if (err.name === 'AbortError') {
                    status.textContent = 'Canceled';
                } else {
                    status.textContent = 'Error: ' + err.message;
                }
            } finally {
                submit.style.display = '';
                cancel.style.display = 'none';
                askAIController = null;
            }
        });

        cancel.addEventListener('click', function() {
            if (askAIController) askAIController.abort();
        });
    }

    // ---------------- Scope toggle ----------------

    function setScope(newScope) {
        scope = newScope;
        const proj = document.getElementById('tasks-scope-project');
        const glob = document.getElementById('tasks-scope-global');
        if (proj && glob) {
            if (newScope === 'project') {
                proj.classList.add('bg-surface-overlay', 'text-th-text-primary');
                proj.classList.remove('text-th-text-muted');
                glob.classList.remove('bg-surface-overlay', 'text-th-text-primary');
                glob.classList.add('text-th-text-muted');
            } else {
                glob.classList.add('bg-surface-overlay', 'text-th-text-primary');
                glob.classList.remove('text-th-text-muted');
                proj.classList.remove('bg-surface-overlay', 'text-th-text-primary');
                proj.classList.add('text-th-text-muted');
            }
        }
        loadBoard();
    }

    // ---------------- Utilities ----------------

    function escapeHTML(s) {
        return String(s || '').replace(/[&<>"']/g, function(ch) {
            return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch];
        });
    }
    function escapeAttr(s) { return escapeHTML(s); }

    function formatTimestamp(ts) {
        if (!ts) return '';
        try {
            const d = new Date(ts);
            if (isNaN(d.getTime())) return ts;
            return d.toLocaleString();
        } catch (_) { return ts; }
    }

    // ---------------- Column reorder ----------------

    async function moveColumn(slug, toIndex) {
        try {
            await apiFetch(API + '/columns/' + encodeURIComponent(slug) + '/move' + qs(), {
                method: 'POST',
                body: JSON.stringify({ to_index: toIndex })
            });
            await loadBoard();
        } catch (err) {
            ClawIDEDialog.confirm('Error', 'Failed to move column: ' + err.message, { confirmLabel: 'OK' });
        }
    }

    // ---------------- Storage settings ----------------

    async function loadStorageSettings() {
        if (!projectID) return;
        try {
            var data = await apiFetch(API + '/settings?project_id=' + encodeURIComponent(projectID));
            updateStorageUI(data.task_storage);
        } catch (_) {}
    }

    function updateStorageUI(mode) {
        var inProjBtn = document.getElementById('tasks-storage-in-project');
        var globalBtn = document.getElementById('tasks-storage-global');
        if (!inProjBtn || !globalBtn) return;
        if (mode === 'global') {
            globalBtn.classList.add('bg-surface-overlay', 'text-th-text-primary');
            globalBtn.classList.remove('text-th-text-muted');
            inProjBtn.classList.remove('bg-surface-overlay', 'text-th-text-primary');
            inProjBtn.classList.add('text-th-text-muted');
        } else {
            inProjBtn.classList.add('bg-surface-overlay', 'text-th-text-primary');
            inProjBtn.classList.remove('text-th-text-muted');
            globalBtn.classList.remove('bg-surface-overlay', 'text-th-text-primary');
            globalBtn.classList.add('text-th-text-muted');
        }
    }

    async function setStorageMode(mode) {
        if (!projectID) return;
        try {
            var data = await apiFetch(API + '/settings?project_id=' + encodeURIComponent(projectID), {
                method: 'PUT',
                body: JSON.stringify({ task_storage: mode })
            });
            updateStorageUI(data.task_storage);
            // Reload the board since the file path changed (may scaffold a new board).
            await loadBoard();
        } catch (err) {
            ClawIDEDialog.confirm('Error', 'Failed to change storage: ' + err.message, { confirmLabel: 'OK' });
        }
    }

    // ---------------- Public API ----------------

    window.ClawIDETasks = {
        init: async function(pid) {
            projectID = pid || '';
            await loadProviders();
            await loadStorageSettings();
            await loadBoard();
            await loadMetrics();
        },
        reload: async function() { await loadBoard(); await loadMetrics(); },
        setScope: setScope,
        setStorageMode: setStorageMode,
        promptNewColumn: promptNewColumn
    };
})();
