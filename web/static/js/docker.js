// CCMux Docker Integration
(function() {
    'use strict';

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
            html += '<button onclick="CCMuxDocker.serviceAction(\'' + projectID + '\', \'' + svc.name + '\', \'restart\')" class="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-800 rounded">Restart</button>';
            html += '<button onclick="CCMuxDocker.serviceAction(\'' + projectID + '\', \'' + svc.name + '\', \'stop\')" class="px-2 py-1 text-xs text-gray-400 hover:text-white hover:bg-gray-800 rounded">Stop</button>';
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
                if (!resp.ok) throw new Error('Action failed');
                // Refresh after a short delay
                setTimeout(function() {
                    var container = document.getElementById('docker-services');
                    if (container) refreshServices(projectID, container);
                }, 1000);
            })
            .catch(function(err) {
                console.error('Docker action failed:', err);
            });
    }

    function composeUp(projectID) {
        fetch('/projects/' + projectID + '/api/docker/up', { method: 'POST' })
            .then(function() {
                setTimeout(function() {
                    var container = document.getElementById('docker-services');
                    if (container) refreshServices(projectID, container);
                }, 2000);
            });
    }

    function composeDown(projectID) {
        fetch('/projects/' + projectID + '/api/docker/down', { method: 'POST' })
            .then(function() {
                setTimeout(function() {
                    var container = document.getElementById('docker-services');
                    if (container) refreshServices(projectID, container);
                }, 2000);
            });
    }

    window.CCMuxDocker = {
        refresh: refreshServices,
        serviceAction: serviceAction,
        composeUp: composeUp,
        composeDown: composeDown,
    };
})();
