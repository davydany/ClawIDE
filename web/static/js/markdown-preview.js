// ClawIDE Markdown Preview
// Renders markdown with Marked.js, syntax highlighting via Highlight.js,
// and Mermaid diagram support in fenced code blocks.
(function() {
    'use strict';

    var initialized = false;

    function init() {
        if (initialized) return;
        initialized = true;

        // Configure Marked.js
        if (typeof marked !== 'undefined') {
            marked.setOptions({
                gfm: true,
                breaks: true,
                highlight: function(code, lang) {
                    if (typeof hljs !== 'undefined' && lang && hljs.getLanguage(lang)) {
                        try {
                            return hljs.highlight(code, { language: lang }).value;
                        } catch (e) { /* fall through */ }
                    }
                    if (typeof hljs !== 'undefined') {
                        try {
                            return hljs.highlightAuto(code).value;
                        } catch (e) { /* fall through */ }
                    }
                    return code;
                }
            });
        }

        // Initialize Mermaid with theme-aware config
        if (typeof mermaid !== 'undefined') {
            var currentMode = document.documentElement.dataset.mode || 'dark';
            var currentTheme = document.documentElement.dataset.theme || 'default';
            var mermaidTheme;
            if (currentTheme === 'mono') {
                mermaidTheme = 'neutral';
            } else {
                mermaidTheme = (currentMode === 'light') ? 'default' : 'dark';
            }
            mermaid.initialize({
                startOnLoad: false,
                theme: mermaidTheme,
                securityLevel: 'strict',
                fontFamily: 'ui-monospace, monospace'
            });
        }

        // Listen for theme changes to reinitialize Mermaid and swap highlight.js stylesheet
        window.addEventListener('clawide:theme-changed', function(e) {
            var mode = (e.detail && e.detail.mode) || 'dark';
            var themeName = (e.detail && e.detail.theme) || 'default';

            if (typeof mermaid !== 'undefined') {
                var mTheme;
                if (themeName === 'mono') {
                    mTheme = 'neutral';
                } else {
                    mTheme = (mode === 'light') ? 'default' : 'dark';
                }
                mermaid.initialize({
                    startOnLoad: false,
                    theme: mTheme,
                    securityLevel: 'strict',
                    fontFamily: 'ui-monospace, monospace'
                });
            }

            // Swap highlight.js stylesheet for light/dark
            var hljsLink = document.querySelector('link[href*="highlightjs"]');
            if (hljsLink) {
                var newStyle = (mode === 'light') ? 'github.min.css' : 'github-dark.min.css';
                hljsLink.href = hljsLink.href.replace(/github(-dark)?\.min\.css/, newStyle);
            }
        });
    }

    /**
     * Render markdown string to HTML with Mermaid diagram support.
     * @param {string} text - Raw markdown content
     * @returns {string} HTML string
     */
    function render(text) {
        if (!text || !text.trim()) {
            return '<span class="text-th-text-faint italic">Nothing to preview</span>';
        }

        init();

        var html;
        if (typeof marked !== 'undefined') {
            html = marked.parse(text);
        } else {
            // Fallback minimal renderer if Marked.js hasn't loaded
            html = fallbackRender(text);
        }

        return html;
    }

    /**
     * After inserting rendered HTML into the DOM, call this to process
     * any Mermaid code blocks into diagrams.
     * @param {HTMLElement} container - The element containing the rendered HTML
     */
    function renderMermaidDiagrams(container) {
        if (typeof mermaid === 'undefined') return;

        // Find all code blocks with class 'language-mermaid'
        var mermaidBlocks = container.querySelectorAll('pre code.language-mermaid');
        if (mermaidBlocks.length === 0) return;

        for (var i = 0; i < mermaidBlocks.length; i++) {
            var codeEl = mermaidBlocks[i];
            var preEl = codeEl.parentElement;
            var source = codeEl.textContent;

            // Create a container div for the diagram
            var diagramDiv = document.createElement('div');
            diagramDiv.className = 'mermaid-diagram my-2 flex justify-center';
            var diagramId = 'mermaid-' + Date.now() + '-' + i;
            diagramDiv.id = diagramId;

            // Replace the pre/code block with the diagram container
            preEl.parentElement.replaceChild(diagramDiv, preEl);

            // Render the diagram
            (function(div, src, id) {
                try {
                    mermaid.render(id + '-svg', src).then(function(result) {
                        div.innerHTML = result.svg;
                    }).catch(function(err) {
                        div.innerHTML = '<div class="text-red-400 text-xs p-2 bg-red-900/20 rounded border border-red-800">'
                            + '<strong>Mermaid Error:</strong> ' + escapeHTML(err.message || String(err))
                            + '</div>';
                    });
                } catch (err) {
                    div.innerHTML = '<div class="text-red-400 text-xs p-2 bg-red-900/20 rounded border border-red-800">'
                        + '<strong>Mermaid Error:</strong> ' + escapeHTML(err.message || String(err))
                        + '</div>';
                }
            })(diagramDiv, source, diagramId);
        }
    }

    /**
     * Convenience: render markdown and insert into a container,
     * then process Mermaid diagrams.
     * @param {HTMLElement} container - Target element
     * @param {string} text - Raw markdown
     */
    function renderInto(container, text) {
        if (!container) return;
        container.innerHTML = render(text);
        renderMermaidDiagrams(container);
    }

    /**
     * Fallback minimal markdown renderer when Marked.js is unavailable.
     */
    function fallbackRender(text) {
        var html = escapeHTML(text);
        // Headings
        html = html.replace(/^### (.+)$/gm, '<h3 class="text-sm font-semibold text-white mt-3 mb-1">$1</h3>');
        html = html.replace(/^## (.+)$/gm, '<h2 class="text-base font-semibold text-white mt-3 mb-1">$1</h2>');
        html = html.replace(/^# (.+)$/gm, '<h1 class="text-lg font-bold text-white mt-3 mb-1">$1</h1>');
        // Bold
        html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
        // Italic
        html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
        // Inline code
        html = html.replace(/`([^`]+)`/g, '<code class="bg-surface-overlay px-1 rounded text-[11px]">$1</code>');
        // Links
        html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener" class="text-accent-text hover:underline">$1</a>');
        // Line breaks
        html = html.replace(/\n/g, '<br>');
        return html;
    }

    function escapeHTML(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Export
    window.ClawIDEMarkdown = {
        render: render,
        renderInto: renderInto,
        renderMermaidDiagrams: renderMermaidDiagrams,
        init: init
    };
})();
