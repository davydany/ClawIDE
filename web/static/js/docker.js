// ClawIDE Docker Integration — Unified service card view
(function() {
    'use strict';

    // Track in-flight operations to prevent double-clicks.
    var busy = false;

    // Cached state from last refresh.
    var lastStatus = null;

    // Last projectID used for refresh (needed by copyEnvFiles).
    var lastProjectID = '';

    // Active WebSocket connections for log streaming, keyed by service name.
    var activeLogSockets = {};

    // Active WebSocket connections for build streaming, keyed by service name.
    var activeBuildSockets = {};

    function showToast(message, duration) {
        if (window.ClawIDEToast) {
            window.ClawIDEToast.show(message, duration || 2000);
        }
    }

    function setButtonsBusy(isBusy, label) {
        busy = isBusy;
        var upBtn = document.getElementById('docker-up-btn');
        var downBtn = document.getElementById('docker-down-btn');
        var restartBtn = document.getElementById('docker-restart-btn');
        var refreshBtn = document.getElementById('docker-refresh-btn');
        if (upBtn) {
            upBtn.disabled = isBusy;
            if (label === 'up') upBtn.textContent = isBusy ? 'Starting...' : 'Up';
        }
        if (downBtn) {
            downBtn.disabled = isBusy;
            if (label === 'down') downBtn.textContent = isBusy ? 'Stopping...' : 'Down';
        }
        if (restartBtn) {
            restartBtn.disabled = isBusy;
            if (label === 'restart') restartBtn.textContent = isBusy ? 'Restarting...' : 'Restart';
        }
        if (refreshBtn) {
            refreshBtn.disabled = isBusy;
        }
    }

    // Disable all interactive buttons in the docker panel.
    function setControlsDisabled(disabled) {
        var ids = ['docker-up-btn', 'docker-down-btn', 'docker-restart-btn', 'docker-refresh-btn'];
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
            var restartBtn = document.getElementById('docker-restart-btn');
            if (upBtn) upBtn.style.display = 'none';
            if (downBtn) downBtn.style.display = 'none';
            if (restartBtn) restartBtn.style.display = 'none';
            return;
        }

        // Compose file exists, daemon running — show controls, check for errors
        var upBtn = document.getElementById('docker-up-btn');
        var downBtn = document.getElementById('docker-down-btn');
        var restartBtn = document.getElementById('docker-restart-btn');
        if (upBtn) upBtn.style.display = '';
        if (downBtn) downBtn.style.display = '';
        if (restartBtn) restartBtn.style.display = '';
        setControlsDisabled(false);

        var alertHtml = '';

        // Missing env files alert with fix button
        if (status.missing_env_files && status.missing_env_files.length > 0) {
            var fileList = status.missing_env_files.map(function(f) { return escapeHtml(f); }).join(', ');
            alertHtml +=
                '<div class="flex items-start gap-3 px-4 py-3 bg-amber-900/20 border border-amber-800/40 rounded-lg mb-2">' +
                    '<svg class="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                    '<div class="flex-1 min-w-0">' +
                        '<p class="text-sm font-medium text-amber-300">Missing env files</p>' +
                        '<p class="text-xs text-amber-400/80 mt-0.5">The following files are missing from this worktree: <span class="font-mono">' + fileList + '</span></p>' +
                        '<p class="text-xs text-amber-400/60 mt-0.5">These are needed by Docker Compose but aren\'t tracked by git.</p>' +
                    '</div>' +
                    '<button onclick="ClawIDEDocker.copyEnvFiles()" class="flex-shrink-0 px-3 py-1.5 text-xs font-medium bg-amber-600 hover:bg-amber-500 text-white rounded transition-colors">' +
                        'Copy from Main' +
                    '</button>' +
                '</div>';
        }

        if (status.error) {
            alertHtml +=
                '<div class="flex items-start gap-3 px-4 py-3 bg-amber-900/20 border border-amber-800/40 rounded-lg">' +
                    '<svg class="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"/></svg>' +
                    '<div class="min-w-0">' +
                        '<p class="text-sm font-medium text-amber-300">Docker reported an issue</p>' +
                        '<p class="text-xs text-amber-400/80 mt-0.5 break-all">' + escapeHtml(status.error) + '</p>' +
                    '</div>' +
                '</div>';
        }

        alertDiv.innerHTML = alertHtml;
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
        return 'bg-th-text-faint';
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
        return 'text-th-text-muted bg-surface-raised';
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

            html += '<div class="bg-surface-base border border-th-border rounded-lg overflow-hidden">';

            // ── Header: two-part layout ──
            html += '<div class="flex items-start justify-between p-3">';

            // Left: clickable expand toggle with two rows
            html += '<button onclick="toggleComposeDetail(\'' + svcId + '\')" class="flex-1 min-w-0 text-left">';

            // Row 1: chevron + status dot + service name
            html += '<div class="flex items-center gap-2">';
            // Chevron
            html += '<svg id="' + svcId + '-chevron" class="w-3 h-3 text-th-text-faint transition-transform flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 01.02-1.06L11.168 10 7.23 6.29a.75.75 0 111.04-1.08l4.5 4.25a.75.75 0 010 1.08l-4.5 4.25a.75.75 0 01-1.06-.02z" clip-rule="evenodd"/></svg>';
            // Status dot
            if (runtime) {
                html += '<span class="w-2 h-2 rounded-full flex-shrink-0 ' + healthColor(runtime) + '"></span>';
            } else {
                html += '<span class="w-2 h-2 rounded-full flex-shrink-0 bg-th-border-muted"></span>';
            }
            // Service name
            html += '<span class="text-sm font-mono text-th-text-primary font-medium">' + escapedName + '</span>';
            html += '</div>';

            // Row 2: health badge + status text + image/build + ports (indented under name)
            html += '<div class="flex items-center gap-1.5 ml-7 mt-1 flex-wrap">';
            // Health badge
            if (runtime) {
                html += '<span class="text-xs px-1.5 py-0.5 rounded ' + healthLabelClass(runtime) + '">' + healthLabel(runtime) + '</span>';
            }
            // Status text
            if (runtime && runtime.status) {
                html += '<span class="text-xs text-th-text-faint">' + escapeHtml(runtime.status) + '</span>';
            } else {
                html += '<span class="text-xs text-th-text-ghost italic">Not running</span>';
            }
            // Image or build badge
            if (svc.image) {
                html += '<span class="text-xs bg-accent-muted/40 text-accent-text px-1.5 py-0.5 rounded truncate max-w-[200px]">' + escapeHtml(svc.image) + '</span>';
            } else if (svc.build) {
                html += '<span class="text-xs bg-amber-900/40 text-amber-300 px-1.5 py-0.5 rounded truncate max-w-[200px]">build: ' + escapeHtml(svc.build) + '</span>';
            }
            // Port badges
            if (svc.ports) {
                svc.ports.forEach(function(port) {
                    if (port.container_port) {
                        html += '<span class="text-xs font-mono bg-accent-muted/30 text-accent-text px-1.5 py-0.5 rounded">:' + port.container_port + '</span>';
                    }
                });
            }
            html += '</div>';

            html += '</button>';

            // Right: prominent action buttons
            html += '<div class="flex items-center gap-2 flex-shrink-0 ml-3 pt-0.5">';
            // Build button (for services with a Dockerfile)
            if (svc.build) {
                var isBuildActive = !!activeBuildSockets[svc.name];
                if (isBuildActive) {
                    html += '<button onclick="event.stopPropagation(); ClawIDEDocker.buildService(\'' + projectID + '\', \'' + escapedName + '\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-purple-500/70 bg-purple-700/50 text-purple-200 transition-colors">Building\u2026</button>';
                } else {
                    html += '<button onclick="event.stopPropagation(); ClawIDEDocker.buildService(\'' + projectID + '\', \'' + escapedName + '\')" class="px-2.5 py-1 text-xs font-medium rounded-lg border border-purple-700/50 bg-purple-900/30 text-purple-300 hover:bg-purple-800/50 transition-colors">Build</button>';
                }
            }
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
            html += '<div id="' + svcId + '-detail" class="hidden border-t border-th-border px-3 py-2 space-y-2 text-xs">';
            if (svc.container_name) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Container</span><span class="text-th-text-tertiary font-mono">' + escapeHtml(svc.container_name) + '</span></div>';
            }
            if (svc.command) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Command</span><span class="text-th-text-tertiary font-mono break-all">' + escapeHtml(svc.command) + '</span></div>';
            }
            if (svc.restart) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Restart</span><span class="text-th-text-tertiary">' + escapeHtml(svc.restart) + '</span></div>';
            }
            // Healthcheck
            if (svc.healthcheck) {
                var hc = svc.healthcheck;
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Healthcheck</span><div>';
                if (hc.test) {
                    html += '<span class="text-th-text-tertiary font-mono break-all">' + escapeHtml(hc.test) + '</span>';
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
                        html += '<span class="bg-surface-raised text-th-text-muted font-mono px-1.5 py-0.5 rounded">' + escapeHtml(b) + '</span>';
                    });
                    html += '</div>';
                }
                html += '</div></div>';
            }
            if (svc.depends_on && svc.depends_on.length > 0) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Depends on</span><div class="flex flex-wrap gap-1">';
                svc.depends_on.forEach(function(dep) {
                    html += '<span class="bg-surface-raised text-th-text-muted px-1.5 py-0.5 rounded">' + escapeHtml(dep) + '</span>';
                });
                html += '</div></div>';
            }
            if (svc.environment && svc.environment.length > 0) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Environment</span><div class="flex flex-wrap gap-1">';
                svc.environment.forEach(function(env) {
                    html += '<span class="bg-surface-raised text-th-text-muted font-mono px-1.5 py-0.5 rounded">' + escapeHtml(env) + '</span>';
                });
                html += '</div></div>';
            }
            if (svc.volumes && svc.volumes.length > 0) {
                html += '<div class="flex gap-2"><span class="text-th-text-faint w-28 flex-shrink-0">Volumes</span><div class="flex flex-col gap-0.5">';
                svc.volumes.forEach(function(vol) {
                    html += '<span class="text-th-text-muted font-mono">' + escapeHtml(vol) + '</span>';
                });
                html += '</div></div>';
            }
            html += '</div>';

            // ── Inline logs container (hidden by default) ──
            html += '<div id="' + svcId + '-logs" class="hidden border-t border-th-border">';
            html += '<div class="flex items-center justify-between px-3 py-2">';
            html += '<span class="text-xs text-th-text-faint">Logs (last 250 lines)</span>';
            html += '<button onclick="ClawIDEDocker.closeLogs(\'' + escapedName + '\')" class="text-xs text-th-text-faint hover:text-th-text-primary transition-colors">Close</button>';
            html += '</div>';
            html += '<pre id="' + svcId + '-logs-output" class="px-3 pb-3 text-xs font-mono text-th-text-muted max-h-80 overflow-y-auto whitespace-pre-wrap"></pre>';
            html += '</div>';

            // ── Inline build output container (hidden by default) ──
            html += '<div id="' + svcId + '-build" class="hidden border-t border-th-border">';
            html += '<div class="flex items-center justify-between px-3 py-2">';
            html += '<span class="text-xs text-purple-400">Build Output</span>';
            html += '<button onclick="ClawIDEDocker.closeBuild(\'' + escapedName + '\')" class="text-xs text-th-text-faint hover:text-th-text-primary transition-colors">Close</button>';
            html += '</div>';
            html += '<pre id="' + svcId + '-build-output" class="px-3 pb-3 text-xs font-mono text-th-text-muted max-h-80 overflow-y-auto whitespace-pre-wrap"></pre>';
            html += '</div>';

            html += '</div>'; // end card
        });

        html += '</div></div>';
        container.innerHTML = html;
    }

    // ─── Main refresh ──────────────────────────────────────

    // Optional base path override for feature-scoped Docker.
    // Set via ClawIDEDocker.setBasePath() before calling refresh().
    var customBasePath = '';
    var customWSPath = '';

    function apiBase(projectID) {
        if (customBasePath) return customBasePath;
        return '/projects/' + projectID;
    }

    function wsBase(projectID) {
        if (customWSPath) return customWSPath;
        return '/ws/docker/' + projectID;
    }

    function refreshStatus(projectID) {
        lastProjectID = projectID;
        // Close all active log sockets before re-rendering
        closeAllLogs();

        fetch(apiBase(projectID) + '/api/docker/status')
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
        fetch(apiBase(projectID) + '/api/docker/' + service + '/' + action, {
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
            container.innerHTML = '<div class="text-th-text-muted text-sm px-4 py-3">Starting services...</div>';
        }

        fetch(apiBase(projectID) + '/api/docker/up', { method: 'POST' })
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
            container.innerHTML = '<div class="text-th-text-muted text-sm px-4 py-3">Stopping services...</div>';
        }

        fetch(apiBase(projectID) + '/api/docker/down', { method: 'POST' })
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

    function composeRestart(projectID) {
        if (busy) return;
        setButtonsBusy(true, 'restart');
        var container = document.getElementById('docker-compose-services');
        if (container) {
            container.innerHTML = '<div class="text-th-text-muted text-sm px-4 py-3">Restarting services...</div>';
        }

        fetch(apiBase(projectID) + '/api/docker/restart', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Failed to restart stack');
                    });
                }
                showToast('Docker Compose stack restarted');
                setTimeout(function() { refreshStatus(projectID); }, 2000);
            })
            .catch(function(err) {
                showToast('Docker restart failed: ' + err.message, 4000);
                refreshStatus(projectID);
            })
            .finally(function() {
                setButtonsBusy(false, 'restart');
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
        var wsUrl = proto + '//' + location.host + wsBase(projectID) + '/logs/' + service + '?tail=250';
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
        Object.keys(activeBuildSockets).forEach(function(service) {
            if (activeBuildSockets[service]) {
                activeBuildSockets[service].close();
            }
        });
        activeBuildSockets = {};
    }

    // ─── Inline Build Viewer ──────────────────────────────

    function buildService(projectID, service) {
        var svcId = 'compose-svc-' + service.replace(/[^a-zA-Z0-9]/g, '-');

        // Toggle off if already building
        if (activeBuildSockets[service]) {
            closeBuild(service);
            return;
        }

        var buildContainer = document.getElementById(svcId + '-build');
        var buildOutput = document.getElementById(svcId + '-build-output');
        if (!buildContainer || !buildOutput) return;

        // Show the build container
        buildContainer.classList.remove('hidden');
        buildOutput.textContent = 'Starting build...\n';

        // Also expand the detail section so build output is visible
        var detail = document.getElementById(svcId + '-detail');
        if (detail && detail.classList.contains('hidden')) {
            detail.classList.remove('hidden');
            var chevron = document.getElementById(svcId + '-chevron');
            if (chevron) chevron.classList.add('rotate-90');
        }

        // Open WebSocket
        var proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
        var wsUrl = proto + '//' + location.host + wsBase(projectID) + '/build/' + service;
        var ws = new WebSocket(wsUrl);

        ws.onopen = function() {
            buildOutput.textContent = '';
        };

        ws.onmessage = function(event) {
            buildOutput.textContent += event.data;
            // Auto-scroll to bottom
            buildOutput.scrollTop = buildOutput.scrollHeight;
        };

        ws.onerror = function() {
            buildOutput.textContent += '\n[WebSocket error]\n';
        };

        ws.onclose = function() {
            delete activeBuildSockets[service];
        };

        activeBuildSockets[service] = ws;
    }

    function closeBuild(service) {
        var svcId = 'compose-svc-' + service.replace(/[^a-zA-Z0-9]/g, '-');

        // Close the WebSocket
        if (activeBuildSockets[service]) {
            activeBuildSockets[service].close();
            delete activeBuildSockets[service];
        }

        // Hide the build container
        var buildContainer = document.getElementById(svcId + '-build');
        if (buildContainer) {
            buildContainer.classList.add('hidden');
        }

        // Clear output
        var buildOutput = document.getElementById(svcId + '-build-output');
        if (buildOutput) {
            buildOutput.textContent = '';
        }
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

    // Copy missing .env files from the main project to the feature worktree.
    function copyEnvFiles() {
        if (!customBasePath) {
            showToast('Copy env files is only available for feature worktrees', 3000);
            return;
        }

        fetch(customBasePath + '/api/docker/copy-env-files', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to copy env files');
                return resp.json();
            })
            .then(function(result) {
                if (result.copied && result.copied.length > 0) {
                    showToast('Copied: ' + result.copied.join(', '), 3000);
                }
                if (result.errors && result.errors.length > 0) {
                    showToast('Some files could not be copied: ' + result.errors.join('; '), 5000);
                }
                // Refresh to clear the alert
                if (lastProjectID) {
                    setTimeout(function() { refreshStatus(lastProjectID); }, 500);
                }
            })
            .catch(function(err) {
                showToast('Failed to copy env files: ' + err.message, 4000);
            });
    }

    // Configure feature-scoped paths for Docker API and WebSocket.
    function setBasePath(apiPath, wsPath) {
        customBasePath = apiPath;
        customWSPath = wsPath;
    }

    function resetBasePath() {
        customBasePath = '';
        customWSPath = '';
    }

    window.ClawIDEDocker = {
        refresh: refreshStatus,
        serviceAction: serviceAction,
        composeUp: composeUp,
        composeDown: composeDown,
        composeRestart: composeRestart,
        viewLogs: viewLogs,
        closeLogs: closeLogs,
        buildService: buildService,
        closeBuild: closeBuild,
        copyEnvFiles: copyEnvFiles,
        setBasePath: setBasePath,
        resetBasePath: resetBasePath,
    };
})();
