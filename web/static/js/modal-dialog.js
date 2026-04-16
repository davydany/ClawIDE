// ClawIDE Shared Modal Dialog Utility
// Replaces browser prompt() and confirm() with themed <dialog> elements.
(function() {
    'use strict';

    var DIALOG_STYLES = 'bg-surface-base text-th-text-secondary rounded-xl shadow-2xl border border-th-border-strong p-0 backdrop:bg-black/60';
    var INPUT_STYLES = 'w-full bg-surface-raised border border-th-border-strong rounded-lg px-3 py-2 text-sm text-th-text-primary placeholder-th-text-faint focus:outline-none focus:border-accent-border';
    var BTN_PRIMARY = 'px-4 py-2 text-sm bg-accent hover:bg-accent-hover text-th-text-primary rounded-lg transition-colors font-medium';
    var BTN_DANGER = 'px-4 py-2 text-sm bg-red-600 hover:bg-red-500 text-th-text-primary rounded-lg transition-colors font-medium';
    var BTN_CANCEL = 'px-4 py-2 text-sm text-th-text-muted hover:text-th-text-primary hover:bg-surface-raised rounded-lg transition-colors';

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
                ? '<span class="text-th-text-faint text-sm ml-1">' + escapeHTML(options.suffix) + '</span>'
                : '';

            dialog.innerHTML =
                '<div class="px-6 pt-5 pb-4">' +
                '  <h3 class="text-base font-semibold text-th-text-primary mb-4">' + escapeHTML(title) + '</h3>' +
                '  <label class="block text-xs text-th-text-muted mb-1.5">' + escapeHTML(label) + '</label>' +
                '  <div class="flex items-center">' +
                '    <input type="text" class="' + INPUT_STYLES + '" value="' + escapeHTML(defaultValue || '') + '"' +
                '           placeholder="' + escapeHTML(options.placeholder || '') + '">' +
                     suffixHTML +
                '  </div>' +
                '</div>' +
                '<div class="flex justify-end gap-2 px-6 py-3 border-t border-th-border-strong">' +
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
                '  <h3 class="text-base font-semibold text-th-text-primary mb-2">' + escapeHTML(title) + '</h3>' +
                '  <p class="text-sm text-th-text-muted">' + escapeHTML(message) + '</p>' +
                '</div>' +
                '<div class="flex justify-end gap-2 px-6 py-3 border-t border-th-border-strong">' +
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

    /**
     * Show a multi-field form dialog.
     * @param {string} title - Dialog title
     * @param {Array<{key:string, label:string, type?:string, placeholder?:string, required?:boolean, value?:string}>} fields
     * @param {Object} [options]
     * @param {string} [options.submitLabel] - Submit button text (default 'Create')
     * @param {string} [options.cancelLabel] - Cancel button text (default 'Cancel')
     * @returns {Promise<Object|null>} Resolved with {key: value, ...} or null if cancelled
     */
    function form(title, fields, options) {
        options = options || {};
        var submitLabel = options.submitLabel || 'Create';
        var cancelLabel = options.cancelLabel || 'Cancel';

        return new Promise(function(resolve) {
            var dialog = document.createElement('dialog');
            dialog.className = DIALOG_STYLES;
            dialog.style.minWidth = '400px';
            dialog.style.maxWidth = '520px';

            var fieldsHTML = '';
            for (var i = 0; i < fields.length; i++) {
                var f = fields[i];
                var inputType = f.type || 'text';
                var req = f.required ? ' required' : '';
                var val = escapeHTML(f.value || '');
                var ph = escapeHTML(f.placeholder || '');
                fieldsHTML += '<div class="mb-3">';
                fieldsHTML += '  <label class="block text-xs text-th-text-muted mb-1.5">' + escapeHTML(f.label) + '</label>';
                if (inputType === 'textarea') {
                    fieldsHTML += '  <textarea data-field="' + escapeHTML(f.key) + '" class="' + INPUT_STYLES + ' resize-y" rows="3" placeholder="' + ph + '"' + req + '>' + val + '</textarea>';
                } else {
                    fieldsHTML += '  <input type="' + inputType + '" data-field="' + escapeHTML(f.key) + '" class="' + INPUT_STYLES + '" value="' + val + '" placeholder="' + ph + '"' + req + '>';
                }
                fieldsHTML += '</div>';
            }

            dialog.innerHTML =
                '<div class="px-6 pt-5 pb-2">' +
                '  <h3 class="text-base font-semibold text-th-text-primary mb-4">' + escapeHTML(title) + '</h3>' +
                   fieldsHTML +
                '</div>' +
                '<div class="flex justify-end gap-2 px-6 py-3 border-t border-th-border-strong">' +
                '  <button type="button" class="dialog-cancel ' + BTN_CANCEL + '">' + escapeHTML(cancelLabel) + '</button>' +
                '  <button type="button" class="dialog-ok ' + BTN_PRIMARY + '">' + escapeHTML(submitLabel) + '</button>' +
                '</div>';

            var okBtn = dialog.querySelector('.dialog-ok');
            var cancelBtn = dialog.querySelector('.dialog-cancel');

            function collect() {
                var result = {};
                var inputs = dialog.querySelectorAll('[data-field]');
                for (var j = 0; j < inputs.length; j++) {
                    result[inputs[j].dataset.field] = inputs[j].value;
                }
                return result;
            }

            function validate() {
                for (var j = 0; j < fields.length; j++) {
                    if (fields[j].required) {
                        var el = dialog.querySelector('[data-field="' + fields[j].key + '"]');
                        if (el && !el.value.trim()) {
                            el.focus();
                            el.classList.add('border-red-500');
                            el.addEventListener('input', function() { this.classList.remove('border-red-500'); }, { once: true });
                            return false;
                        }
                    }
                }
                return true;
            }

            function finish(value) {
                dialog.close();
                dialog.remove();
                resolve(value);
            }

            okBtn.addEventListener('click', function() {
                if (validate()) finish(collect());
            });

            cancelBtn.addEventListener('click', function() {
                finish(null);
            });

            dialog.addEventListener('close', function() {
                dialog.remove();
                resolve(null);
            });

            // Enter in text inputs submits; Enter in textareas inserts newline (default).
            dialog.addEventListener('keydown', function(e) {
                if (e.key === 'Enter' && e.target.tagName !== 'TEXTAREA') {
                    e.preventDefault();
                    if (validate()) finish(collect());
                }
            });

            document.body.appendChild(dialog);
            dialog.showModal();

            // Focus the first input.
            var first = dialog.querySelector('[data-field]');
            if (first) first.focus();
        });
    }

    // Export
    window.ClawIDEDialog = {
        prompt: prompt,
        confirm: confirm,
        form: form
    };
})();
