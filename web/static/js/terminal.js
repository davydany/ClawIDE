// ClawIDE Terminal Manager
(function() {
    'use strict';

    const terminals = {}; // keyed by paneID
    let focusedPaneID = null;
    const dataInterceptors = [];

    function createTerminal(sessionID, paneID, container) {
        if (terminals[paneID]) {
            return terminals[paneID];
        }

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

        // Track focus for modifier toolbar
        if (term.textarea) {
            term.textarea.addEventListener('focus', function() {
                focusedPaneID = paneID;
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
                // Send initial size
                sendResize();
            };

            ws.onmessage = function(evt) {
                if (evt.data instanceof ArrayBuffer) {
                    term.write(new Uint8Array(evt.data));
                } else {
                    term.write(evt.data);
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
