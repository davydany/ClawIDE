// ClawIDE Shared Modal Dialog Utility
// Replaces browser prompt() and confirm() with themed <dialog> elements.
(function() {
    'use strict';

    var DIALOG_STYLES = 'bg-gray-900 text-gray-100 rounded-xl shadow-2xl border border-gray-700 p-0 backdrop:bg-black/60';
    var INPUT_STYLES = 'w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500';
    var BTN_PRIMARY = 'px-4 py-2 text-sm bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg transition-colors font-medium';
    var BTN_DANGER = 'px-4 py-2 text-sm bg-red-600 hover:bg-red-500 text-white rounded-lg transition-colors font-medium';
    var BTN_CANCEL = 'px-4 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-800 rounded-lg transition-colors';

    function escapeHTML(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    /**
     * Show a prompt dialog.
     * @param {string} title - Dialog title
     * @param {string} label - Input label
     * @param {string} defaultValue - Default input value
     * @param {Object} [options]
     * @param {string} [options.suffix] - Suffix shown after input (e.g. '.md')
     * @param {string} [options.placeholder] - Input placeholder
     * @returns {Promise<string|null>} Resolved with input value or null if cancelled
     */
    function prompt(title, label, defaultValue, options) {
        options = options || {};
        return new Promise(function(resolve) {
            var dialog = document.createElement('dialog');
            dialog.className = DIALOG_STYLES;
            dialog.style.minWidth = '340px';
            dialog.style.maxWidth = '440px';

            var suffixHTML = options.suffix
                ? '<span class="text-gray-500 text-sm ml-1">' + escapeHTML(options.suffix) + '</span>'
                : '';

            dialog.innerHTML =
                '<div class="px-6 pt-5 pb-4">' +
                '  <h3 class="text-base font-semibold text-white mb-4">' + escapeHTML(title) + '</h3>' +
                '  <label class="block text-xs text-gray-400 mb-1.5">' + escapeHTML(label) + '</label>' +
                '  <div class="flex items-center">' +
                '    <input type="text" class="' + INPUT_STYLES + '" value="' + escapeHTML(defaultValue || '') + '"' +
                '           placeholder="' + escapeHTML(options.placeholder || '') + '">' +
                     suffixHTML +
                '  </div>' +
                '</div>' +
                '<div class="flex justify-end gap-2 px-6 py-3 border-t border-gray-700">' +
                '  <button type="button" class="dialog-cancel ' + BTN_CANCEL + '">Cancel</button>' +
                '  <button type="button" class="dialog-ok ' + BTN_PRIMARY + '">OK</button>' +
                '</div>';

            var input = dialog.querySelector('input');
            var okBtn = dialog.querySelector('.dialog-ok');
            var cancelBtn = dialog.querySelector('.dialog-cancel');

            function finish(value) {
                dialog.close();
                dialog.remove();
                resolve(value);
            }

            okBtn.addEventListener('click', function() {
                finish(input.value);
            });

            cancelBtn.addEventListener('click', function() {
                finish(null);
            });

            dialog.addEventListener('close', function() {
                // ESC key or programmatic close
                dialog.remove();
                resolve(null);
            });

            input.addEventListener('keydown', function(e) {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    finish(input.value);
                }
            });

            document.body.appendChild(dialog);
            dialog.showModal();

            // Auto-focus and select input text
            input.focus();
            input.select();
        });
    }

    /**
     * Show a confirm dialog.
     * @param {string} title - Dialog title
     * @param {string} message - Confirmation message
     * @param {Object} [options]
     * @param {boolean} [options.destructive] - Use red confirm button
     * @param {string} [options.confirmLabel] - Custom confirm button text (default 'Confirm')
     * @param {string} [options.cancelLabel] - Custom cancel button text (default 'Cancel')
     * @returns {Promise<boolean>} Resolved with true if confirmed, false if cancelled
     */
    function confirm(title, message, options) {
        options = options || {};
        return new Promise(function(resolve) {
            var dialog = document.createElement('dialog');
            dialog.className = DIALOG_STYLES;
            dialog.style.minWidth = '340px';
            dialog.style.maxWidth = '440px';

            var btnClass = options.destructive ? BTN_DANGER : BTN_PRIMARY;
            var confirmLabel = options.confirmLabel || 'Confirm';
            var cancelLabel = options.cancelLabel || 'Cancel';

            dialog.innerHTML =
                '<div class="px-6 pt-5 pb-4">' +
                '  <h3 class="text-base font-semibold text-white mb-2">' + escapeHTML(title) + '</h3>' +
                '  <p class="text-sm text-gray-400">' + escapeHTML(message) + '</p>' +
                '</div>' +
                '<div class="flex justify-end gap-2 px-6 py-3 border-t border-gray-700">' +
                '  <button type="button" class="dialog-cancel ' + BTN_CANCEL + '">' + escapeHTML(cancelLabel) + '</button>' +
                '  <button type="button" class="dialog-ok ' + btnClass + '">' + escapeHTML(confirmLabel) + '</button>' +
                '</div>';

            var okBtn = dialog.querySelector('.dialog-ok');
            var cancelBtn = dialog.querySelector('.dialog-cancel');

            function finish(value) {
                dialog.close();
                dialog.remove();
                resolve(value);
            }

            okBtn.addEventListener('click', function() {
                finish(true);
            });

            cancelBtn.addEventListener('click', function() {
                finish(false);
            });

            dialog.addEventListener('close', function() {
                dialog.remove();
                resolve(false);
            });

            document.body.appendChild(dialog);
            dialog.showModal();

            // Focus the confirm button so Enter confirms
            okBtn.focus();
        });
    }

    // Export
    window.ClawIDEDialog = {
        prompt: prompt,
        confirm: confirm
    };
})();
