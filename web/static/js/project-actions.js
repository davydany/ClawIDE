// ClawIDE project actions — rename (display + directory), remove from
// ClawIDE, and trash. Shared by the workspace tab bar and the dashboard
// project cards.
(function() {
    'use strict';

    function dialog() { return window.ClawIDEDialog; }

    async function renameDisplay(id, currentName) {
        var D = dialog();
        if (!D) return;
        var name = await D.prompt('Rename Project', 'Display name', currentName, {
            placeholder: 'My project'
        });
        if (name === null) return;
        name = (name || '').trim();
        if (name === '' || name === currentName) return;
        try {
            var res = await fetch('/projects/' + encodeURIComponent(id) + '/', {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });
            if (!res.ok) throw new Error(await res.text() || 'rename failed');
            window.location.reload();
        } catch (err) {
            alert('Rename failed: ' + err.message);
        }
    }

    async function renameDirectory(id, currentBasename) {
        var D = dialog();
        if (!D) return;
        var name = await D.prompt(
            'Rename Directory on Disk',
            'New directory name (stays in the same parent folder)',
            currentBasename,
            { placeholder: 'my-project' }
        );
        if (name === null) return;
        name = (name || '').trim();
        if (name === '' || name === currentBasename) return;
        if (/[\/\\\x00]|^\.\.?$/.test(name)) {
            alert('Invalid directory name. Use only filename characters — no slashes, no "." or "..".');
            return;
        }
        var ok = await D.confirm(
            'Rename directory on disk?',
            'This renames the folder on your filesystem and closes any open terminal sessions for this project. You will need to reconnect sessions after the rename.',
            { confirmLabel: 'Rename', cancelLabel: 'Cancel' }
        );
        if (!ok) return;
        try {
            var res = await fetch('/projects/' + encodeURIComponent(id) + '/path', {
                method: 'PATCH',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name: name })
            });
            if (!res.ok) throw new Error(await res.text() || 'rename failed');
            window.location.reload();
        } catch (err) {
            alert('Rename directory failed: ' + err.message);
        }
    }

    async function removeFromClawIDE(id, projectName) {
        var D = dialog();
        if (!D) return;
        var ok = await D.confirm(
            'Remove "' + projectName + '" from ClawIDE?',
            'The project will be removed from ClawIDE along with its sessions and features. Files on disk will NOT be touched — you can add the project back later.',
            { destructive: true, confirmLabel: 'Remove', cancelLabel: 'Cancel' }
        );
        if (!ok) return;
        try {
            var res = await fetch('/projects/' + encodeURIComponent(id) + '/', {
                method: 'DELETE',
                headers: { 'HX-Request': 'true' }
            });
            if (!res.ok) throw new Error(await res.text() || 'remove failed');
            window.location.href = '/';
        } catch (err) {
            alert('Remove failed: ' + err.message);
        }
    }

    async function trashProject(id, projectName) {
        var D = dialog();
        if (!D) return;
        var ok = await D.confirm(
            'Delete "' + projectName + '"?',
            'The project folder will be moved to the ClawIDE trash. You can restore it from the trash within 30 days. After that it will be permanently deleted.',
            { destructive: true, confirmLabel: 'Delete', cancelLabel: 'Cancel' }
        );
        if (!ok) return;
        try {
            var res = await fetch('/projects/' + encodeURIComponent(id) + '/trash', {
                method: 'POST',
                headers: { 'HX-Request': 'true' }
            });
            if (!res.ok) throw new Error(await res.text() || 'trash failed');
            window.location.href = '/';
        } catch (err) {
            alert('Delete failed: ' + err.message);
        }
    }

    window.ClawIDEProjectActions = {
        renameDisplay: renameDisplay,
        renameDirectory: renameDirectory,
        removeFromClawIDE: removeFromClawIDE,
        trash: trashProject
    };
})();
