(function() {
    'use strict';

    var TOURS = {
        dashboard: [
            {
                selector: '[data-tour="sidebar"]',
                title: 'Sidebar Navigation',
                description: 'Access the dashboard, your projects, and settings from here.',
                position: 'right'
            },
            {
                selector: '[data-tour="new-project"]',
                title: 'Create a Project',
                description: 'Create a new project by providing a name and path on your filesystem.',
                position: 'bottom'
            },
            {
                selector: '[data-tour="project-grid"]',
                title: 'Your Projects',
                description: 'Your projects appear here as cards. Click one to open its workspace.',
                position: 'top'
            },
            {
                selector: '[data-tour="settings-link"]',
                title: 'Settings',
                description: 'Configure your preferences like projects directory, max sessions, and more.',
                position: 'right'
            }
        ],
        workspace: [
            {
                selector: '[data-tour="new-session"]',
                title: 'Create a Session',
                description: 'Start a new terminal session. Each session gives you split panes with Claude Code integration.',
                position: 'right'
            },
            {
                selector: '[data-tour="tab-terminal"]',
                title: 'Terminal Tab',
                description: 'Manage your terminal sessions with split panes. Run commands and interact with Claude Code.',
                position: 'bottom'
            },
            {
                selector: '[data-tour="tab-files"]',
                title: 'Files Tab',
                description: 'Browse and edit your project files with CodeMirror syntax highlighting and multi-tab support.',
                position: 'bottom'
            },
            {
                selector: '[data-tour="tab-docker"]',
                title: 'Docker Tab',
                description: 'Manage Docker Compose services. Start, stop, and view logs for your containers.',
                position: 'bottom'
            },
            {
                selector: '[data-tour="tab-ports"]',
                title: 'Ports Tab',
                description: 'Auto-detect running services on your machine and access them directly from here.',
                position: 'bottom'
            }
        ]
    };

    var overlay = null;
    var spotlight = null;
    var tooltip = null;
    var currentTour = null;
    var currentStep = 0;
    var onCompleteCallback = null;

    function createElements() {
        overlay = document.createElement('div');
        overlay.className = 'tour-overlay';

        spotlight = document.createElement('div');
        spotlight.className = 'tour-spotlight';

        tooltip = document.createElement('div');
        tooltip.className = 'tour-tooltip';

        document.body.appendChild(overlay);
        document.body.appendChild(spotlight);
        document.body.appendChild(tooltip);
    }

    function removeElements() {
        if (overlay) { overlay.remove(); overlay = null; }
        if (spotlight) { spotlight.remove(); spotlight = null; }
        if (tooltip) { tooltip.remove(); tooltip = null; }
    }

    function isElementVisible(el) {
        var rect = el.getBoundingClientRect();
        return rect.width > 0 && rect.height > 0;
    }

    function showStep(stepIndex) {
        var steps = TOURS[currentTour];
        if (!steps || stepIndex < 0 || stepIndex >= steps.length) {
            finish();
            return;
        }

        var step = steps[stepIndex];
        var target = document.querySelector(step.selector);

        // Skip steps where target doesn't exist or isn't visible
        if (!target || !isElementVisible(target)) {
            if (stepIndex < steps.length - 1) {
                currentStep = stepIndex + 1;
                showStep(currentStep);
            } else {
                finish();
            }
            return;
        }

        currentStep = stepIndex;
        var rect = target.getBoundingClientRect();
        var padding = 8;

        // Position spotlight
        spotlight.style.top = (rect.top - padding) + 'px';
        spotlight.style.left = (rect.left - padding) + 'px';
        spotlight.style.width = (rect.width + padding * 2) + 'px';
        spotlight.style.height = (rect.height + padding * 2) + 'px';

        // Build tooltip content
        var totalSteps = steps.length;
        tooltip.innerHTML =
            '<div style="margin-bottom:12px">' +
                '<h4 style="font-size:14px;font-weight:600;color:#f9fafb;margin:0 0 4px 0">' + step.title + '</h4>' +
                '<p style="font-size:13px;color:#9ca3af;margin:0;line-height:1.4">' + step.description + '</p>' +
            '</div>' +
            '<div style="display:flex;align-items:center;justify-content:space-between">' +
                '<span style="font-size:12px;color:#6b7280">' + (stepIndex + 1) + ' / ' + totalSteps + '</span>' +
                '<div style="display:flex;gap:8px;align-items:center">' +
                    '<button class="tour-skip" style="font-size:12px;color:#6b7280;background:none;border:none;cursor:pointer;padding:4px 8px">Skip Tour</button>' +
                    (stepIndex > 0
                        ? '<button class="tour-back" style="font-size:13px;color:#d1d5db;background:#374151;border:1px solid #4b5563;border-radius:6px;cursor:pointer;padding:6px 14px">Back</button>'
                        : '') +
                    '<button class="tour-next" style="font-size:13px;color:#fff;background:#4f46e5;border:none;border-radius:6px;cursor:pointer;padding:6px 14px">' +
                        (stepIndex === totalSteps - 1 ? 'Finish' : 'Next') +
                    '</button>' +
                '</div>' +
            '</div>';

        // Position tooltip
        positionTooltip(rect, step.position);

        // Bind button events
        var nextBtn = tooltip.querySelector('.tour-next');
        var backBtn = tooltip.querySelector('.tour-back');
        var skipBtn = tooltip.querySelector('.tour-skip');

        if (nextBtn) {
            nextBtn.onclick = function() {
                if (stepIndex === totalSteps - 1) {
                    finish();
                } else {
                    showStep(stepIndex + 1);
                }
            };
        }
        if (backBtn) {
            backBtn.onclick = function() { showStep(stepIndex - 1); };
        }
        if (skipBtn) {
            skipBtn.onclick = function() { finish(); };
        }
    }

    function positionTooltip(targetRect, position) {
        var tooltipWidth = 300;
        var gap = 16;
        var top, left;

        tooltip.style.width = tooltipWidth + 'px';

        switch (position) {
            case 'right':
                top = targetRect.top;
                left = targetRect.right + gap;
                break;
            case 'left':
                top = targetRect.top;
                left = targetRect.left - tooltipWidth - gap;
                break;
            case 'bottom':
                top = targetRect.bottom + gap;
                left = targetRect.left + (targetRect.width / 2) - (tooltipWidth / 2);
                break;
            case 'top':
                top = targetRect.top - gap;
                left = targetRect.left + (targetRect.width / 2) - (tooltipWidth / 2);
                break;
            default:
                top = targetRect.bottom + gap;
                left = targetRect.left;
        }

        // Viewport clamping
        var viewportWidth = window.innerWidth;
        var viewportHeight = window.innerHeight;

        if (left < 12) left = 12;
        if (left + tooltipWidth > viewportWidth - 12) left = viewportWidth - tooltipWidth - 12;
        if (top < 12) top = 12;

        // If tooltip would go off-screen at bottom when position is 'top', measure it
        if (position === 'top') {
            // We need to account for tooltip height; estimate ~120px
            top = top - 120;
            if (top < 12) top = targetRect.bottom + gap;
        }

        if (top > viewportHeight - 160) top = viewportHeight - 160;

        tooltip.style.top = top + 'px';
        tooltip.style.left = left + 'px';
    }

    function finish() {
        removeElements();
        window.removeEventListener('resize', onResize);
        currentTour = null;
        currentStep = 0;
        if (onCompleteCallback) {
            var cb = onCompleteCallback;
            onCompleteCallback = null;
            cb();
        }
    }

    function onResize() {
        if (currentTour) {
            showStep(currentStep);
        }
    }

    function start(tourName, onComplete) {
        if (!TOURS[tourName]) return;

        currentTour = tourName;
        currentStep = 0;
        onCompleteCallback = onComplete || null;

        createElements();
        window.addEventListener('resize', onResize);

        // Allow overlay click to skip
        overlay.onclick = function() { finish(); };

        showStep(0);
    }

    window.ClawIDETour = {
        start: start,
        finish: finish
    };
})();
