// ClawIDE Terminal Manager
(function() {
    'use strict';

    var TERMINAL_THEMES = {
        default: {
            background: '#0a0a0a',
            foreground: '#e4e4e7',
            cursor: '#a5b4fc',
            selectionBackground: '#4338ca88',
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
        claude: {
            background: '#1a1510',
            foreground: '#ece5d8',
            cursor: '#e89e7e',
            selectionBackground: '#6b4c2a88',
            black: '#1a1510',
            red: '#e5736a',
            green: '#7eba6b',
            yellow: '#d4a643',
            blue: '#6a9fd4',
            magenta: '#b888c8',
            cyan: '#5bb8b0',
            white: '#ece5d8',
            brightBlack: '#635143',
            brightRed: '#f0918a',
            brightGreen: '#9cd48a',
            brightYellow: '#e4bf65',
            brightBlue: '#8ab8e8',
            brightMagenta: '#d0a5dc',
            brightCyan: '#7dd0c8',
            brightWhite: '#f5f0e8',
        },
        mono: {
            background: '#000000',
            foreground: '#e5e5e5',
            cursor: '#ffffff',
            selectionBackground: '#40404088',
            black: '#000000',
            red: '#ff6b6b',
            green: '#69db7c',
            yellow: '#ffd43b',
            blue: '#74c0fc',
            magenta: '#da77f2',
            cyan: '#66d9e8',
            white: '#e5e5e5',
            brightBlack: '#525252',
            brightRed: '#ff8787',
            brightGreen: '#8ce99a',
            brightYellow: '#ffe066',
            brightBlue: '#a5d8ff',
            brightMagenta: '#e599f7',
            brightCyan: '#99e9f2',
            brightWhite: '#ffffff',
        },
    };

    var TERMINAL_THEMES_LIGHT = {
        default: {
            background: '#ffffff',
            foreground: '#1e293b',
            cursor: '#4f46e5',
            selectionBackground: '#c7d2fe88',
            black: '#1e293b',
            red: '#dc2626',
            green: '#16a34a',
            yellow: '#ca8a04',
            blue: '#2563eb',
            magenta: '#9333ea',
            cyan: '#0891b2',
            white: '#f8fafc',
            brightBlack: '#64748b',
            brightRed: '#ef4444',
            brightGreen: '#22c55e',
            brightYellow: '#eab308',
            brightBlue: '#3b82f6',
            brightMagenta: '#a855f7',
            brightCyan: '#06b6d4',
            brightWhite: '#ffffff',
        },
        claude: {
            background: '#ffffff',
            foreground: '#2a2219',
            cursor: '#c96842',
            selectionBackground: '#fde8de88',
            black: '#1a1510',
            red: '#c53030',
            green: '#2f855a',
            yellow: '#b7791f',
            blue: '#2b6cb0',
            magenta: '#805ad5',
            cyan: '#0e7490',
            white: '#faf8f5',
            brightBlack: '#7d6e5c',
            brightRed: '#e53e3e',
            brightGreen: '#38a169',
            brightYellow: '#d69e2e',
            brightBlue: '#4299e1',
            brightMagenta: '#9f7aea',
            brightCyan: '#0891b2',
            brightWhite: '#ffffff',
        },
        mono: {
            background: '#ffffff',
            foreground: '#171717',
            cursor: '#404040',
            selectionBackground: '#e5e5e588',
            black: '#0a0a0a',
            red: '#dc2626',
            green: '#16a34a',
            yellow: '#ca8a04',
            blue: '#2563eb',
            magenta: '#9333ea',
            cyan: '#0891b2',
            white: '#fafafa',
            brightBlack: '#525252',
            brightRed: '#ef4444',
            brightGreen: '#22c55e',
            brightYellow: '#eab308',
            brightBlue: '#3b82f6',
            brightMagenta: '#a855f7',
            brightCyan: '#06b6d4',
            brightWhite: '#ffffff',
        },
    };

    function getCurrentThemeName() {
        return document.documentElement.dataset.theme || 'default';
    }

    function getCurrentMode() {
        return document.documentElement.dataset.mode || 'dark';
    }

    function getTerminalTheme() {
        var pool = (getCurrentMode() === 'light') ? TERMINAL_THEMES_LIGHT : TERMINAL_THEMES;
        return pool[getCurrentThemeName()] || TERMINAL_THEMES['default'];
    }

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
            theme: getTerminalTheme(),
            allowProposedApi: true,
        });

        const fitAddon = new window.XtermFitAddon();
        term.loadAddon(fitAddon);

        const webLinksAddon = new window.XtermWebLinksAddon();
        term.loadAddon(webLinksAddon);

        term.open(container);
        fitAddon.fit();

        // Clipboard helpers (defined before key handler so they're in scope)
        function showCopyToast() {
            if (window.ClawIDEToast) {
                window.ClawIDEToast.show('✓ Copied');
            }
        }

        function copyToClipboard(text) {
            // Try async Clipboard API first (requires secure context)
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(text).then(function() {
                    console.log('Copied via Clipboard API');
                    showCopyToast();
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
                showCopyToast();
            } catch (e) {
                console.error('Copy failed:', e);
            }
            document.body.removeChild(textarea);
        }

        // Auto-copy on mouse selection (highlight-to-copy)
        // Try xterm's native selection first; if empty (tmux mouse mode captures
        // the selection instead), fall back to reading tmux's paste buffer.
        var selectionCopyTimer = null;
        var lastCopiedText = '';
        term.onSelectionChange(function() {
            clearTimeout(selectionCopyTimer);
            selectionCopyTimer = setTimeout(function() {
                var selection = term.getSelection();
                if (!selection) {
                    lastCopiedText = '';
                    return;
                }
                if (selection !== lastCopiedText) {
                    lastCopiedText = selection;
                    copyToClipboard(selection);
                }
            }, 150);
        });

        var didMouseDrag = false;
        term.element.addEventListener('mousedown', function() {
            didMouseDrag = false;
        });
        term.element.addEventListener('mousemove', function() {
            didMouseDrag = true;
        });

        term.element.addEventListener('mouseup', function() {
            if (!didMouseDrag) return; // Simple click/focus — not a selection gesture
            setTimeout(function() {
                var selection = term.getSelection();
                if (selection) return; // xterm handled it
                // Tmux mouse mode captured the selection — read from tmux buffer
                fetch('/api/tmux/buffer').then(function(r) {
                    if (!r.ok) return;
                    return r.json();
                }).then(function(data) {
                    if (data && data.text) {
                        copyToClipboard(data.text);
                    }
                }).catch(function() {});
            }, 50);
        });

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
                // Try to read clipboard - check for images first, then text
                if (navigator.clipboard) {
                    ev.preventDefault();
                    handleClipboardPaste();
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

            // Handle paste events for mobile (context menu paste, not keyboard)
            term.textarea.addEventListener('paste', function(e) {
                if (!e.clipboardData) return;

                var items = e.clipboardData.items;
                if (!items || items.length === 0) return;

                // Check for images first
                for (var i = 0; i < items.length; i++) {
                    var item = items[i];
                    if (item.kind === 'file' && item.type.startsWith('image/')) {
                        e.preventDefault();
                        var blob = item.getAsFile();
                        if (blob) {
                            handleImagePaste(blob);
                        }
                        return;
                    }
                }
            });
        }

        // Handle clipboard paste - check for images first, then text
        function handleClipboardPaste() {
            if (!navigator.clipboard || !navigator.clipboard.read) {
                console.warn('Clipboard read not supported');
                return;
            }

            navigator.clipboard.read().then(function(items) {
                // Look for image data first
                for (var i = 0; i < items.length; i++) {
                    var item = items[i];
                    var imageType = Array.from(item.types).find(function(type) {
                        return type.startsWith('image/');
                    });

                    if (imageType) {
                        item.getType(imageType).then(function(blob) {
                            handleImagePaste(blob);
                        }).catch(function(err) {
                            console.error('Failed to read image:', err);
                        });
                        return;
                    }
                }

                // No image found, try text
                var textType = Array.from(items[0].types).find(function(type) {
                    return type === 'text/plain';
                });

                if (textType && items[0]) {
                    items[0].getType(textType).then(function(blob) {
                        return blob.text();
                    }).then(function(text) {
                        if (text) sendData(text);
                    }).catch(function(err) {
                        console.error('Failed to read text:', err);
                    });
                }
            }).catch(function(err) {
                console.warn('Clipboard read denied:', err);
            });
        }

        function handleImagePaste(blob) {
            var reader = new FileReader();
            reader.onload = function(e) {
                var base64Data = e.target.result;
                // Extract just the base64 part (after the comma)
                var base64String = base64Data.split(',')[1];
                if (base64String) {
                    // Send image data to terminal with a special prefix
                    // Claude Code can detect this and handle the image appropriately
                    sendData('__IMAGE_PASTED__:' + base64String + '\n');
                    if (window.ClawIDEToast) {
                        window.ClawIDEToast.show('✓ Image pasted');
                    }
                }
            };
            reader.onerror = function(err) {
                console.error('Failed to read image blob:', err);
            };
            reader.readAsDataURL(blob);
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
            resizeObserver: resizeObserver,
            container: container,
            sendResize: sendResize,
            sendInput: function(data) {
                sendData(data);
            },
            paste: function() {
                handleClipboardPaste();
            },
            detach: function() {
                // Disconnect observer but keep terminal and WS alive
                this.resizeObserver.disconnect();
            },
            reattach: function(newContainer) {
                this.container = newContainer;
                newContainer.appendChild(this.term.element);
                var self = this;
                this.resizeObserver = new ResizeObserver(function() {
                    self.fitAddon.fit();
                    self.sendResize();
                });
                this.resizeObserver.observe(newContainer);
                this.fitAddon.fit();
                this.sendResize();
            },
            destroy: function() {
                this.closed = true;
                if (reconnectTimer) clearTimeout(reconnectTimer);
                if (ws) ws.close();
                this.resizeObserver.disconnect();
                term.dispose();
                if (focusedPaneID === paneID) {
                    focusedPaneID = null;
                }
                delete terminals[paneID];
            }
        };

        terminals[paneID] = termState;

        // Attach touch gesture handling for mobile scrolling and text selection
        if (window.ClawIDETouchTerminal) window.ClawIDETouchTerminal.attach(termState);

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
        detachAll: function() {
            Object.keys(terminals).forEach(function(id) {
                terminals[id].detach();
            });
        },
        reattach: function(paneID, newContainer) {
            if (terminals[paneID]) {
                terminals[paneID].reattach(newContainer);
            }
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
        paste: function(paneID) {
            var ts = terminals[paneID];
            if (ts && ts.paste) {
                ts.paste();
            }
        },
        addDataInterceptor: function(fn) {
            dataInterceptors.push(fn);
        },
        setTheme: function(themeName, mode) {
            var pool = (mode === 'light') ? TERMINAL_THEMES_LIGHT : TERMINAL_THEMES;
            var theme = pool[themeName] || TERMINAL_THEMES['default'];
            Object.keys(terminals).forEach(function(id) {
                terminals[id].term.options.theme = theme;
            });
        }
    };

    // Listen for theme changes
    window.addEventListener('clawide:theme-changed', function(e) {
        var themeName = e.detail && e.detail.theme || 'default';
        var mode = e.detail && e.detail.mode || 'dark';
        window.ClawIDETerminal.setTheme(themeName, mode);
    });

    // Close all WebSocket connections before page navigation to prevent
    // HTTP/1.1 connection exhaustion (Chrome allows max 6 per host).
    // Without this, old connections linger during navigation, blocking the
    // new page's document request from acquiring a connection slot.
    window.addEventListener('beforeunload', function() {
        window.ClawIDETerminal.destroyAll();
    });
})();
