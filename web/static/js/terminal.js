// CCMux Terminal Manager
(function() {
    'use strict';

    const terminals = {};

    function createTerminal(sessionID, container) {
        if (terminals[sessionID]) {
            return terminals[sessionID];
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

        // Connect WebSocket
        const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsURL = `${proto}//${window.location.host}/ws/terminal/${sessionID}`;
        let ws = null;
        let reconnectTimer = null;

        function connect() {
            ws = new WebSocket(wsURL);
            ws.binaryType = 'arraybuffer';

            ws.onopen = function() {
                console.log(`Terminal ${sessionID} connected`);
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
                console.log(`Terminal ${sessionID} disconnected`);
                // Reconnect after delay
                if (!terminals[sessionID]?.closed) {
                    reconnectTimer = setTimeout(connect, 2000);
                }
            };

            ws.onerror = function(err) {
                console.error(`Terminal ${sessionID} error:`, err);
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

        // Write to PTY
        term.onData(function(data) {
            if (ws && ws.readyState === WebSocket.OPEN) {
                ws.send(new TextEncoder().encode(data));
            }
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
            closed: false,
            destroy: function() {
                this.closed = true;
                if (reconnectTimer) clearTimeout(reconnectTimer);
                if (ws) ws.close();
                resizeObserver.disconnect();
                term.dispose();
                delete terminals[sessionID];
            }
        };

        terminals[sessionID] = termState;
        return termState;
    }

    // Expose to global scope
    window.CCMuxTerminal = {
        create: createTerminal,
        get: function(id) { return terminals[id]; },
        destroyAll: function() {
            Object.keys(terminals).forEach(function(id) {
                terminals[id].destroy();
            });
        }
    };
})();
