// CCMux File Editor
(function() {
    'use strict';

    let currentFile = null;
    let modified = false;

    function loadFile(projectID, filePath) {
        fetch('/projects/' + projectID + '/api/file?path=' + encodeURIComponent(filePath))
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to load file');
                return resp.text();
            })
            .then(function(content) {
                currentFile = filePath;
                modified = false;
                var editor = document.getElementById('editor-content');
                if (editor) {
                    editor.value = content;
                }
                var nameEl = document.getElementById('editor-filename');
                if (nameEl) {
                    nameEl.textContent = filePath.split('/').pop();
                }
                updateModifiedIndicator();
            })
            .catch(function(err) {
                console.error('Failed to load file:', err);
            });
    }

    function saveFile(projectID) {
        if (!currentFile) return;

        var editor = document.getElementById('editor-content');
        if (!editor) return;

        fetch('/projects/' + projectID + '/api/file?path=' + encodeURIComponent(currentFile), {
            method: 'PUT',
            headers: { 'Content-Type': 'text/plain' },
            body: editor.value,
        })
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to save');
                modified = false;
                updateModifiedIndicator();
            })
            .catch(function(err) {
                console.error('Failed to save file:', err);
            });
    }

    function updateModifiedIndicator() {
        var indicator = document.getElementById('editor-modified');
        if (indicator) {
            indicator.style.display = modified ? 'inline' : 'none';
        }
    }

    window.CCMuxEditor = {
        loadFile: loadFile,
        saveFile: saveFile,
        isModified: function() { return modified; },
        getCurrentFile: function() { return currentFile; },
    };
})();
