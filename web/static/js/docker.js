// ClawIDE Docker Integration
(function() {
    'use strict';

    // Track in-flight operations to prevent double-clicks and show loading state.
    var busy = false;

    function showToast(message, duration) {
        if (window.ClawIDEToast) {
            window.ClawIDEToast.show(message, duration || 2000);
        }
    }

    function setButtonsBusy(isBusy) {
        busy = isBusy;
        var upBtn = document.getElementById('docker-up-btn');
        var downBtn = document.getElementById('docker-down-btn');
        if (upBtn) {
            upBtn.disabled = isBusy;
            upBtn.textContent = isBusy ? 'Starting...' : 'Up';
        }
        if (downBtn) {
            downBtn.disabled = isBusy;
            downBtn.textContent = isBusy ? 'Stopping...' : 'Down';
        }
    }

    function refreshServices(projectID, container) {
        fetch('/projects/' + projectID + '/api/docker/ps')
            .then(function(resp) {
                if (!resp.ok) throw new Error('Failed to fetch services');
                return resp.json();
            })
            .then(function(services) {
                renderServices(projectID, services, container);
            })
            .catch(function(err) {
                container.innerHTML = '<div class="text-gray-500 text-sm p-4">No Docker Compose services found</div>';
            });
    }

    function renderServices(projectID, services, container) {
        if (!services || services.length === 0) {
            container.innerHTML = '<div class="text-gray-500 text-sm p-4">No services running</div>';
            return;
        }

        var html = '<div class="divide-y divide-gray-800">';
        services.forEach(function(svc) {
            var stateColor = svc.state === 'running' ? 'bg-green-500' : 'bg-red-500';
            html += '<div class="flex items-center justify-between px-4 py-3">';
            html += '<div class="flex items-center gap-2">';
            html += '<span class="w-2 h-2 rounded-full ' + stateColor + '"></span>';
            html += '<span class="text-sm text-white font-medium">' + svc.name + '</span>';
            html += '<span class="text-xs text-gray-500">' + svc.status + '</span>';
            html += '</div>';
            html += '<div class="flex items-center gap-1">';
            html += '<button onclick="ClawIDEDocker.serviceAction(\'' + projectID + '\', \'' + svc.name + '\', \'restart\')" class="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-800 rounded">Restart</button>';
            html += '<button onclick="ClawIDEDocker.serviceAction(\'' + projectID + '\', \'' + svc.name + '\', \'stop\')" class="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-800 rounded">Stop</button>';
            html += '</div>';
            html += '</div>';
        });
        html += '</div>';
        container.innerHTML = html;
    }

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
                setTimeout(function() {
                    var container = document.getElementById('docker-services');
                    if (container) refreshServices(projectID, container);
                }, 1000);
            })
            .catch(function(err) {
                showToast('Docker ' + action + ' failed: ' + err.message, 3000);
            });
    }

    function composeUp(projectID) {
        if (busy) return;
        setButtonsBusy(true);
        var container = document.getElementById('docker-services');
        if (container) {
            container.innerHTML = '<div class="text-gray-400 text-sm p-4">Starting services...</div>';
        }

        fetch('/projects/' + projectID + '/api/docker/up', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Failed to start stack');
                    });
                }
                showToast('Docker Compose stack started');
                setTimeout(function() {
                    if (container) refreshServices(projectID, container);
                }, 2000);
            })
            .catch(function(err) {
                showToast('Docker up failed: ' + err.message, 4000);
                if (container) refreshServices(projectID, container);
            })
            .finally(function() {
                setButtonsBusy(false);
            });
    }

    function composeDown(projectID) {
        if (busy) return;
        setButtonsBusy(true);
        var container = document.getElementById('docker-services');
        if (container) {
            container.innerHTML = '<div class="text-gray-400 text-sm p-4">Stopping services...</div>';
        }

        fetch('/projects/' + projectID + '/api/docker/down', { method: 'POST' })
            .then(function(resp) {
                if (!resp.ok) {
                    return resp.json().then(function(body) {
                        throw new Error(body.error || 'Failed to stop stack');
                    });
                }
                showToast('Docker Compose stack stopped');
                setTimeout(function() {
                    if (container) refreshServices(projectID, container);
                }, 2000);
            })
            .catch(function(err) {
                showToast('Docker down failed: ' + err.message, 4000);
                if (container) refreshServices(projectID, container);
            })
            .finally(function() {
                setButtonsBusy(false);
            });
    }

    window.ClawIDEDocker = {
        refresh: refreshServices,
        serviceAction: serviceAction,
        composeUp: composeUp,
        composeDown: composeDown,
    };
})();
