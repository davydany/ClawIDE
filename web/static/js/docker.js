// ClawIDE Docker Integration — Unified service card view
(function() {
    'use strict';

    // Track in-flight operations to prevent double-clicks.
    var busy = false;

    // Cached state from last refresh.
    var lastStatus = null;

    // Active WebSocket connections for log streaming, keyed by service name.
    var activeLogSockets = {};

    function showToast(message, duration) {
        if (window.ClawIDEToast) {
            window.ClawIDEToast.show(message, duration || 2000);
        }
    }

    function setButtonsBusy(isBusy, label) {
        busy = isBusy;
        var upBtn = document.getElementById('docker-up-btn');
        var downBtn = document.getElementById('docker-down-btn');
        var refreshBtn = document.getElementById('docker-refresh-btn');
        if (upBtn) {
            upBtn.disabled = isBusy;
            if (label === 'up') upBtn.textContent = isBusy ? 'Starting...' : 'Up';
        }
        if (downBtn) {
            downBtn.disabled = isBusy;
            if (label === 'down') downBtn.textContent = isBusy ? 'Stopping...' : 'Down';
        }
        if (refreshBtn) {
            refreshBtn.disabled = isBusy;
        }
    }

    // Disable all interactive buttons in the docker panel.
    function setControlsDisabled(disabled) {
        var ids = ['docker-up-btn', 'docker-down-btn', 'docker-refresh-btn'];
        ids.forEach(function(id) {
            var el = document.getElementById(id);
            if (el) el.disabled = disabled;
        });
    }

    // Update the "Open Web App" link in the tab bar.
    function updateWebAppLink(url) {
        var link = document.getElementById('web-app-link');
        if (!link) return;
        if (url) {
            link.href = url;
            link.style.display = '';
        } else {
            link.style.display = 'none';
        }
    }

    // ─── Rendering helpers ──────────────────────────────────

    function renderAlert(container, status) {
        var alertDiv = document.getElementById('docker-alert');
        if (!alertDiv) return;

        if (!status.daemon_running) {
            alertDiv.innerHTML =
                '<div class="flex items-start gap-3 px-4 py-3 bg-red-900/30 border border-red-800/50 rounded-lg">' +
                    '<svg class="w-5 h-5 text-red-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                    '<div>' +
                        '<p class="text-sm font-medium text-red-300">Docker daemon is not running</p>' +
                        '<p class="text-xs text-red-400/80 mt-0.5">Please start Docker Desktop or the Docker service to manage containers.</p>' +
                    '</div>' +
                '</div>';
            setControlsDisabled(true);
            return;
        }

        if (!status.compose_file) {
            alertDiv.innerHTML =
                '<div class="flex items-start gap-3 px-4 py-3 bg-amber-900/20 border border-amber-800/40 rounded-lg">' +
                    '<svg class="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"/></svg>' +
                    '<div>' +
                        '<p class="text-sm font-medium text-amber-300">No docker-compose.yml found</p>' +
                        '<p class="text-xs text-amber-400/80 mt-0.5">Add a docker-compose.yml to this project to manage services here.</p>' +
                    '</div>' +
                '</div>';
            // Hide compose controls
            var upBtn = document.getElementById('docker-up-btn');
            var downBtn = document.getElementById('docker-down-btn');
            if (upBtn) upBtn.style.display = 'none';
            if (downBtn) downBtn.style.display = 'none';
            return;
        }

        // Compose file exists, daemon running — show controls, check for errors
        var upBtn = document.getElementById('docker-up-btn');
        var downBtn = document.getElementById('docker-down-btn');
        if (upBtn) upBtn.style.display = '';
        if (downBtn) downBtn.style.display = '';
        setControlsDisabled(false);

        if (status.error) {
            alertDiv.innerHTML =
                '<div class="flex items-start gap-3 px-4 py-3 bg-amber-900/20 border border-amber-800/40 rounded-lg">' +
                    '<svg class="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                    '<div class="min-w-0">' +
                        '<p class="text-sm font-medium text-amber-300">Docker reported an issue</p>' +
                        '<p class="text-xs text-amber-400/80 mt-0.5 break-all">' + escapeHtml(status.error) + '</p>' +
                    '</div>' +
                '</div>';
        } else {
            alertDiv.innerHTML = '';
        }
    }

    // Determine the health indicator color class for a service.
    function healthColor(svc) {
        var state = (svc.state || '').toLowerCase();
        var health = (svc.health || '').toLowerCase();

        if (state === 'exited' || state === 'dead' || state === 'removing') return 'bg-red-500';
        if (health === 'unhealthy') return 'bg-red-500';
        if (state === 'created' || state === 'restarting') return 'bg-amber-500';
        if (health === 'starting') return 'bg-amber-500';
        if (state === 'paused') return 'bg-amber-500';
        if (state === 'running') return 'bg-green-500';
        return 'bg-gray-500';
    }

    function healthLabel(svc) {
        var state = (svc.state || '').toLowerCase();
        var health = (svc.health || '').toLowerCase();

        if (state === 'exited' || state === 'dead' || state === 'removing') return 'stopped';
        if (health === 'unhealthy') return 'unhealthy';
        if (state === 'created') return 'created';
        if (state === 'restarting') return 'restarting';
        if (health === 'starting') return 'starting';
        if (state === 'paused') return 'paused';
        if (state === 'running' && health === 'healthy') return 'healthy';
        if (state === 'running') return 'running';
        return state || 'unknown';
    }

    function healthLabelClass(svc) {
        var color = healthColor(svc);
        if (color === 'bg-green-500') return 'text-green-400 bg-green-900/30';
        if (color === 'bg-amber-500') return 'text-amber-400 bg-amber-900/30';
        if (color === 'bg-red-500') return 'text-red-400 bg-red-900/30';
        return 'text-gray-400 bg-gray-800';
    }

    // Build a lookup map from runtime services keyed by compose service name.
    function buildRuntimeMap(runtimeServices) {
        var map = {};
        if (!runtimeServices) return map;
        runtimeServices.forEach(function(svc) {
            var key = svc.service || svc.name;
            map[key] = svc;
        });
        return map;
    }

    // Render unified compose service cards with merged runtime status.
    function renderComposeServices(projectID, composeServices, runtimeServices, container) {
        if (!container) return;
        if (!composeServices || composeServices.length === 0) {
            container.innerHTML = '';
            return;
        }

        var runtimeMap = buildRuntimeMap(runtimeServices);

        var html = '<div class="px-4 py-3">';
        html += '<div class="space-y-2">';

        composeServices.forEach(function(svc) {
            var svcId = 'compose-svc-' + svc.name.replace(/[^a-zA-Z0-9]/g, '-');
            var runtime = runtimeMap[svc.name];
            var isRunning = runtime && runtime.state && runtime.state.toLowerCase() === 'running';
            var escapedName = escapeHtml(svc.name);

            html += '<div class="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">';

            // ── Header: two-part layout ──
            html += '<div class="flex items-start justify-between p-3">';

            // Left: clickable expand toggle with two rows
            html += '<button onclick="toggleComposeDetail(\'' + svcId + '\')" class="flex-1 min-w-0 text-left">';

            // Row 1: chevron + status dot + service name
            html += '<div class="flex items-center gap-2">';
            // Chevron
            html += '<svg id="' + svcId + '-chevron" class="w-3 h-3 text-gray-500 transition-transform flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/></svg>';
            // Status dot
            if (runtime) {
                html += '<span class="w-2 h-2 rounded-full flex-shrink-0 ' + healthColor(runtime) + '"></span>';
            } else {
                html += '<span class="w-2 h-2 rounded-full flex-shrink-0 bg-gray-600"></span>';
            }
            // Service name
            html += '<span class="text-sm font-mono text-white font-medium">' + escapedName + '</span>';
            html += '</div>';

            // Row 2: health badge + status text + image/build + ports (indented under name)
            html += '<div class="flex items-center gap-1.5 ml-7 mt-1 flex-wrap">';
            // Health badge
            if (runtime) {
                html += '<span class="text-xs px-1.5 py-0.5 rounded ' + healthLabelClass(runtime) + '">' + healthLabel(runtime) + '</span>';
            }
            // Status text
            if (runtime && runtime.status) {
                html += '<span class="text-xs text-gray-500">' + escapeHtml(runtime.status) + '</span>';
            } else {
                html += '<span class="text-xs text-gray-600 italic">Not running</span>';
            }
            // Image or build badge
            if (svc.image) {
                html += '<span class="text-xs bg-indigo-900/40 text-indigo-300 px-1.5 py-0.5 rounded truncate max-w-[200px]">' + escapeHtml(svc.image) + '</span>';
            } else if (svc.build) {
                html += '<span class="text-xs bg-amber-900/40 text-amber-300 px-1.5 py-0.5 rounded truncate max-w-[200px]">build: ' + escapeHtml(svc.build) + '</span>';
            }
            // Port badges
            if (svc.ports) {
                svc.ports.forEach(function(port) {
                    if (port.container_port) {
                        html += '<span class="text-xs font-mono bg-indigo-900/30 text-indigo-300 px-1.5 py-0.5 rounded">:' + port.container_port + '</span>';
                    }
                });
            }
            html += '</div>';

            html += '</button>';

            // Right: prominent action buttons
            html += '<div class="flex items-center gap-2 flex-shrink-0 ml-3 pt-0.5">';
            if (isRunning) {
                html += '<button onclick="event.stopPropagation(); ClawIDEDocker.viewLogs(\'' + projectID + '\', \'' + escapedName + '\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-blue-700/50 bg-blue-900/30 text-blue-300 hover:bg-blue-800/50 transition-colors">Logs</button>';
                html += '<button onclick="event.stopPropagation(); ClawIDEDocker.serviceAction(\'' + projectID + '\', \'' + escapedName + '\', \'restart\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-amber-700/50 bg-amber-900/30 text-amber-300 hover:bg-amber-800/50 transition-colors">Restart</button>';
                html += '<button onclick="event.stopPropagation(); ClawIDEDocker.serviceAction(\'' + projectID + '\', \'' + escapedName + '\', \'stop\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-red-700/50 bg-red-900/30 text-red-300 hover:bg-red-800/50 transition-colors">Stop</button>';
            } else {
                html += '<button onclick="event.stopPropagation(); ClawIDEDocker.serviceAction(\'' + projectID + '\', \'' + escapedName + '\', \'start\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-green-700/50 bg-green-900/30 text-green-300 hover:bg-green-800/50 transition-colors">Start</button>';
            }
            html += '</div>';

            html += '</div>'; // end header

            // ── Detail section (hidden by default) ──
            html += '<div id="' + svcId + '-detail" class="hidden border-t border-gray-800 px-3 py-2 space-y-2 text-xs">';
            if (svc.container_name) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Container</span><span class="text-gray-300 font-mono">' + escapeHtml(svc.container_name) + '</span></div>';
            }
            if (svc.command) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Command</span><span class="text-gray-300 font-mono break-all">' + escapeHtml(svc.command) + '</span></div>';
            }
            if (svc.restart) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Restart</span><span class="text-gray-300">' + escapeHtml(svc.restart) + '</span></div>';
            }
            // Healthcheck
            if (svc.healthcheck) {
                var hc = svc.healthcheck;
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Healthcheck</span><div>';
                if (hc.test) {
                    html += '<span class="text-gray-300 font-mono break-all">' + escapeHtml(hc.test) + '</span>';
                }
                if (hc.disabled) {
                    html += '<span class="text-red-400 ml-1">(disabled)</span>';
                }
                // Sub-badges: interval, timeout, retries, start_period
                var badges = [];
                if (hc.interval) badges.push('interval: ' + hc.interval);
                if (hc.timeout) badges.push('timeout: ' + hc.timeout);
                if (hc.retries) badges.push('retries: ' + hc.retries);
                if (hc.start_period) badges.push('start: ' + hc.start_period);
                if (badges.length > 0) {
                    html += '<div class="flex flex-wrap gap-1 mt-1">';
                    badges.forEach(function(b) {
                        html += '<span class="bg-gray-800 text-gray-400 font-mono px-1.5 py-0.5 rounded">' + escapeHtml(b) + '</span>';
                    });
                    html += '</div>';
                }
                html += '</div></div>';
            }
            if (svc.depends_on && svc.depends_on.length > 0) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Depends on</span><div class="flex flex-wrap gap-1">';
                svc.depends_on.forEach(function(dep) {
                    html += '<span class="bg-gray-800 text-gray-400 px-1.5 py-0.5 rounded">' + escapeHtml(dep) + '</span>';
                });
                html += '</div></div>';
            }
            if (svc.environment && svc.environment.length > 0) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Environment</span><div class="flex flex-wrap gap-1">';
                svc.environment.forEach(function(env) {
                    html += '<span class="bg-gray-800 text-gray-400 font-mono px-1.5 py-0.5 rounded">' + escapeHtml(env) + '</span>';
                });
                html += '</div></div>';
            }
            if (svc.volumes && svc.volumes.length > 0) {
                html += '<div class="flex gap-2"><span class="text-gray-500 w-28 flex-shrink-0">Volumes</span><div class="flex flex-col gap-0.5">';
                svc.volumes.forEach(function(vol) {
                    html += '<span class="text-gray-400 font-mono">' + escapeHtml(vol) + '</span>';
                });
                html += '</div></div>';
            }
            html += '</div>';

            // ── Inline logs container (hidden by default) ──
            html += '<div id="' + svcId + '-logs" class="hidden border-t border-gray-800">';
            html += '<div class="flex items-center justify-between px-3 py-2">';
            html += '<span class="text-xs text-gray-500">Logs (last 250 lines)</span>';
            html += '<button onclick="ClawIDEDocker.closeLogs(\'' + escapedName + '\')" class="text-xs text-gray-500 hover:text-white transition-colors">Close</button>';
            html += '</div>';
            html += '<pre id="' + svcId + '-logs-output" class="px-3 pb-3 text-xs font-mono text-gray-400 max-h-80 overflow-y-auto whitespace-pre-wrap"></pre>';
            html += '</div>';

            html += '</div>'; // end card
        });

        html += '</div></div>';
        container.innerHTML = html;
    }

    // ─── Main refresh ──────────────────────────────────────

    function refreshStatus(projectID) {
        // Close all active log sockets before re-rendering
        closeAllLogs();

        fetch('/projects/' + projectID + '/api/docker/status')
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to fetch docker status');
                return resp.json();
            })
            .then(function(status) {
                lastStatus = status;

                // Render alert
                renderAlert(document.getElementById('docker-alert'), status);

                // Render unified compose + runtime cards
                renderComposeServices(projectID, status.compose_services, status.services, document.getElementById('docker-compose-services'));

                // Update web app link
                updateWebAppLink(status.web_app_url);
            })
            .catch(function(err) {
                var container = document.getElementById('docker-compose-services');
                if (container) {
                    container.innerHTML = '<div class="text-red-400 text-sm px-4 py-3">Failed to load Docker status: ' + escapeHtml(err.message) + '</div>';
                }
            });
    }

    // ─── Actions ───────────────────────────────────────────

    function serviceAction(projectID, service, action) {
        fetch('/projects/' + projectID + '/api/docker/' + service + '/' + action, {
            method: 'POST',
        })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Action failed');
                    });
                }
                showToast('Docker ' + action + ' ' + service + ' succeeded');
                setTimeout(function() { refreshStatus(projectID); }, 1000);
            })
            .catch(function(err) {
                showToast('Docker ' + action + ' failed: ' + err.message, 3000);
            });
    }

    function composeUp(projectID) {
        if (busy) return;
        setButtonsBusy(true, 'up');
        var container = document.getElementById('docker-compose-services');
        if (container) {
            container.innerHTML = '<div class="text-gray-400 text-sm px-4 py-3">Starting services...</div>';
        }

        fetch('/projects/' + projectID + '/api/docker/up', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Failed to start stack');
                    });
                }
                showToast('Docker Compose stack started');
                setTimeout(function() { refreshStatus(projectID); }, 2000);
            })
            .catch(function(err) {
                showToast('Docker up failed: ' + err.message, 4000);
                refreshStatus(projectID);
            })
            .finally(function() {
                setButtonsBusy(false, 'up');
            });
    }

    function composeDown(projectID) {
        if (busy) return;
        setButtonsBusy(true, 'down');
        var container = document.getElementById('docker-compose-services');
        if (container) {
            container.innerHTML = '<div class="text-gray-400 text-sm px-4 py-3">Stopping services...</div>';
        }

        fetch('/projects/' + projectID + '/api/docker/down', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Failed to stop stack');
                    });
                }
                showToast('Docker Compose stack stopped');
                setTimeout(function() { refreshStatus(projectID); }, 2000);
            })
            .catch(function(err) {
                showToast('Docker down failed: ' + err.message, 4000);
                refreshStatus(projectID);
            })
            .finally(function() {
                setButtonsBusy(false, 'down');
            });
    }

    // ─── Inline Log Viewer ─────────────────────────────────

    function viewLogs(projectID, service) {
        var svcId = 'compose-svc-' + service.replace(/[^a-zA-Z0-9]/g, '-');

        // Toggle off if already streaming
        if (activeLogSockets[service]) {
            closeLogs(service);
            return;
        }

        var logsContainer = document.getElementById(svcId + '-logs');
        var logsOutput = document.getElementById(svcId + '-logs-output');
        if (!logsContainer || !logsOutput) return;

        // Show the logs container
        logsContainer.classList.remove('hidden');
        logsOutput.textContent = 'Connecting...\n';

        // Also expand the detail section so logs are visible
        var detail = document.getElementById(svcId + '-detail');
        if (detail && detail.classList.contains('hidden')) {
            detail.classList.remove('hidden');
            var chevron = document.getElementById(svcId + '-chevron');
            if (chevron) chevron.classList.add('rotate-90');
        }

        // Open WebSocket
        var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        var wsUrl = proto + '//' + location.host + '/ws/docker/' + projectID + '/logs/' + service + '?tail=250';
        var ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            logsOutput.textContent = '';
        };

        ws.onmessage = function(event) {
            logsOutput.textContent += event.data;
            // Auto-scroll to bottom
            logsOutput.scrollTop = logsOutput.scrollHeight;
        };

        ws.onerror = function() {
            logsOutput.textContent += '\n[WebSocket error]\n';
        };

        ws.onclose = function() {
            logsOutput.textContent += '\n[Stream closed]\n';
            delete activeLogSockets[service];
        };

        activeLogSockets[service] = ws;
    }

    function closeLogs(service) {
        var svcId = 'compose-svc-' + service.replace(/[^a-zA-Z0-9]/g, '-');

        // Close the WebSocket
        if (activeLogSockets[service]) {
            activeLogSockets[service].close();
            delete activeLogSockets[service];
        }

        // Hide the logs container
        var logsContainer = document.getElementById(svcId + '-logs');
        if (logsContainer) {
            logsContainer.classList.add('hidden');
        }

        // Clear output
        var logsOutput = document.getElementById(svcId + '-logs-output');
        if (logsOutput) {
            logsOutput.textContent = '';
        }
    }

    function closeAllLogs() {
        Object.keys(activeLogSockets).forEach(function(service) {
            if (activeLogSockets[service]) {
                activeLogSockets[service].close();
            }
        });
        activeLogSockets = {};
    }

    // ─── Utilities ─────────────────────────────────────────

    function escapeHtml(str) {
        if (!str) return '';
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Toggle compose service detail expansion
    window.toggleComposeDetail = function(svcId) {
        var detail = document.getElementById(svcId + '-detail');
        var chevron = document.getElementById(svcId + '-chevron');
        if (detail) {
            detail.classList.toggle('hidden');
        }
        if (chevron) {
            chevron.classList.toggle('rotate-90');
        }
    };

    // ─── Public API ────────────────────────────────────────

    window.ClawIDEDocker = {
        refresh: refreshStatus,
        serviceAction: serviceAction,
        composeUp: composeUp,
        composeDown: composeDown,
        viewLogs: viewLogs,
        closeLogs: closeLogs,
    };
})();
