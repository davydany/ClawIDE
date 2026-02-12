// ClawIDE Terminal Manager
(function() {
    'use strict';

    const terminals = {}; // keyed by paneID
    let focusedPaneID = null;
    const dataInterceptors = [];

    function updateFocusedPane(paneID) {
        // Remove highlight from previous pane
        if (focusedPaneID && focusedPaneID !== paneID) {
            var prevContainer = document.getElementById('pane-' + focusedPaneID);
            if (prevContainer) {
                var prevLeaf = prevContainer.closest('.pane-leaf');
                if (prevLeaf) prevLeaf.classList.remove('focused');
            }
        }
        focusedPaneID = paneID;
        // Add highlight to new pane
        var container = document.getElementById('pane-' + paneID);
        if (container) {
            var leaf = container.closest('.pane-leaf');
            if (leaf) leaf.classList.add('focused');
        }
    }

    function createTerminal(sessionID, paneID, container) {
        if (terminals[paneID]) {
            return terminals[paneID];
        }

        // Suppress DA responses during scrollback replay to prevent
        // escape sequences like ESC[?1;2c from echoing as visible text.
        let replayingScrollback = true;

        const term = new window.XtermTerminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: "'JetBrains Mono', 'Fira Code', monospace",
            theme: {
                background: '#0a0a0a',
                foreground: '#e4e4e7',
                cursor: '#a5b4fc',
                selectionBackground: '#4338ca44',
                black: '#18181b',
                red: '#ef4444',
                green: '#22c55e',
                yellow: '#eab308',
                blue: '#3b82f6',
                magenta: '#a855f7',
                cyan: '#06b6d4',
                white: '#e4e4e7',
                brightBlack: '#52525b',
                brightRed: '#f87171',
                brightGreen: '#4ade80',
                brightYellow: '#facc15',
                brightBlue: '#60a5fa',
                brightMagenta: '#c084fc',
                brightCyan: '#22d3ee',
                brightWhite: '#fafafa',
            },
            allowProposedApi: true,
        });

        const fitAddon = new window.XtermFitAddon();
        term.loadAddon(fitAddon);

        const webLinksAddon = new window.XtermWebLinksAddon();
        term.loadAddon(webLinksAddon);

        term.open(container);
        fitAddon.fit();

        // Clipboard helpers (defined before key handler so they're in scope)
        function copyToClipboard(text) {
            // Try async Clipboard API first (requires secure context)
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(text).then(function() {
                    console.log('Copied via Clipboard API');
                }).catch(function() {
                    fallbackCopy(text);
                });
                return;
            }
            fallbackCopy(text);
        }

        function fallbackCopy(text) {
            // Fallback: temporary textarea + execCommand (works on plain HTTP)
            var textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.cssText = 'position:fixed;left:-9999px;top:-9999px;opacity:0';
            document.body.appendChild(textarea);
            textarea.select();
            try {
                document.execCommand('copy');
                console.log('Copied via execCommand fallback');
            } catch (e) {
                console.error('Copy failed:', e);
            }
            document.body.removeChild(textarea);
        }

        // Enable keyboard copy-paste
        // Cmd+C/V on macOS, Ctrl+Shift+C/V on Linux/Windows
        term.attachCustomKeyEventHandler(function(ev) {
            if (ev.type !== 'keydown') return true;

            var key = ev.key.toLowerCase();

            // Detect copy/paste intent:
            // - Cmd+C/V (metaKey) on macOS
            // - Ctrl+Shift+C/V on Linux/Windows
            var isCopyOrPaste = (key === 'c' || key === 'v') &&
                (ev.metaKey || (ev.ctrlKey && ev.shiftKey));

            if (!isCopyOrPaste) return true;

            if (key === 'c') {
                var selection = term.getSelection();
                if (selection) {
                    ev.preventDefault();
                    copyToClipboard(selection);
                }
                return false;
            }

            if (key === 'v') {
                // If Clipboard API is available, read from it directly
                if (navigator.clipboard && navigator.clipboard.readText) {
                    ev.preventDefault();
                    navigator.clipboard.readText().then(function(text) {
                        if (text) sendData(text);
                    }).catch(function() {
                        console.warn('Clipboard read denied — use right-click paste or toolbar button');
                    });
                    return false;
                }
                // No Clipboard API (non-secure context): let the browser handle
                // Cmd+V natively — xterm will catch the paste event via onData
                return true;
            }

            return true;
        });

        // Track focus for modifier toolbar and visual highlighting
        if (term.textarea) {
            term.textarea.addEventListener('focus', function() {
                updateFocusedPane(paneID);
            });
        }

        // Connect WebSocket with both sessionID and paneID
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${proto}//${window.location.host}/ws/terminal/${sessionID}/${paneID}`;
        let ws = null;
        let reconnectTimer = null;

        function sendData(data) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(new TextEncoder().encode(data));
            }
        }

        function connect() {
            ws = new WebSocket(wsURL);
            ws.binaryType = 'arraybuffer';

            ws.onopen = function() {
                console.log(`Terminal pane ${paneID} connected`);
                replayingScrollback = true;
                // Send initial size
                sendResize();
            };

            ws.onmessage = function(evt) {
                if (evt.data instanceof ArrayBuffer) {
                    term.write(new Uint8Array(evt.data));
                } else {
                    term.write(evt.data);
                }
                // After the first message (scrollback history), stop suppressing.
                // Use setTimeout(0) so any synchronous onData calls during
                // term.write() above are still suppressed.
                if (replayingScrollback) {
                    setTimeout(function() { replayingScrollback = false; }, 0);
                }
            };

            ws.onclose = function() {
                console.log(`Terminal pane ${paneID} disconnected`);
                // Reconnect after delay
                if (!terminals[paneID]?.closed) {
                    reconnectTimer = setTimeout(connect, 2000);
                }
            };

            ws.onerror = function(err) {
                console.error(`Terminal pane ${paneID} error:`, err);
            };
        }

        function sendResize() {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'resize',
                    rows: term.rows,
                    cols: term.cols,
                }));
            }
        }

        // Write to PTY - run data through interceptors first
        term.onData(function(data) {
            if (replayingScrollback) return;
            var processed = data;
            for (var i = 0; i < dataInterceptors.length; i++) {
                processed = dataInterceptors[i](processed);
            }
            sendData(processed);
        });

        // Handle resize
        const resizeObserver = new ResizeObserver(function() {
            fitAddon.fit();
            sendResize();
        });
        resizeObserver.observe(container);

        connect();

        const termState = {
            term: term,
            fitAddon: fitAddon,
            ws: ws,
            paneID: paneID,
            sessionID: sessionID,
            closed: false,
            sendInput: function(data) {
                sendData(data);
            },
            destroy: function() {
                this.closed = true;
                if (reconnectTimer) clearTimeout(reconnectTimer);
                if (ws) ws.close();
                resizeObserver.disconnect();
                term.dispose();
                if (focusedPaneID === paneID) {
                    focusedPaneID = null;
                }
                delete terminals[paneID];
            }
        };

        terminals[paneID] = termState;

        // Auto-set focus to first terminal created (mobile: users tap toolbar before terminal)
        if (focusedPaneID === null) {
            focusedPaneID = paneID;
        }

        return termState;
    }

    // Expose to global scope
    window.ClawIDETerminal = {
        create: createTerminal,
        get: function(paneID) { return terminals[paneID]; },
        destroy: function(paneID) {
            if (terminals[paneID]) {
                terminals[paneID].destroy();
            }
        },
        destroyAll: function() {
            Object.keys(terminals).forEach(function(id) {
                terminals[id].destroy();
            });
        },
        getFocusedPaneID: function() {
            return focusedPaneID;
        },
        setFocusedPaneID: function(paneID) {
            if (terminals[paneID]) {
                updateFocusedPane(paneID);
            }
        },
        focusPane: function(paneID) {
            var ts = terminals[paneID];
            if (ts) {
                updateFocusedPane(paneID);
                ts.term.focus();
                // Scroll the pane into view if needed
                var container = document.getElementById('pane-' + paneID);
                if (container) {
                    container.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
                }
            }
        },
        getAllPaneIDs: function() {
            return Object.keys(terminals);
        },
        getTerminalSelection: function(paneID) {
            var ts = terminals[paneID];
            if (ts && ts.term) {
                return ts.term.getSelection();
            }
            return '';
        },
        sendInput: function(paneID, data) {
            var ts = terminals[paneID];
            if (ts) {
                ts.sendInput(data);
            }
        },
        addDataInterceptor: function(fn) {
            dataInterceptors.push(fn);
        }
    };
})();
