// ClawIDE Touch Terminal — touch gesture handling for mobile terminals
(function() {
    'use strict';

    // Feature-detect touch support; no-op on desktop
    var isTouchDevice = ('ontouchstart' in window) || (navigator.maxTouchPoints > 0);

    // State machine states
    var STATE = {
        IDLE: 'idle',
        PENDING: 'pending',
        SCROLLING: 'scrolling',
        SELECTING: 'selecting',
        MOMENTUM: 'momentum',
    };

    // Configuration
    var LONG_PRESS_MS = 500;
    var SWIPE_THRESHOLD = 8;            // px before classifying as swipe
    var MOMENTUM_DECAY = 0.92;
    var MOMENTUM_MIN_VELOCITY = 0.5;    // px/ms threshold to stop
    var SCROLL_MULTIPLIER = 1.5;        // amplify swipe delta for smoother scrolling
    var VELOCITY_BUFFER_SIZE = 5;

    function attach(termState) {
        if (!isTouchDevice) return;

        var term = termState.term;
        var container = term.element;
        if (!container) return;

        var xtermScreen = container.querySelector('.xterm-screen');
        if (!xtermScreen) return;

        // --- Gesture state ---
        var state = STATE.IDLE;
        var startX = 0;
        var startY = 0;
        var lastY = 0;
        var longPressTimer = null;
        var momentumRAF = null;
        var velocityBuffer = [];        // {y, t} circular buffer
        var velocityIdx = 0;
        var selStartCol = 0;
        var selStartRow = 0;
        var isCarouselSwipe = false;    // true when swiping horizontally in phone carousel mode

        // --- Helpers ---

        function touchToCell(touchX, touchY) {
            var rect = xtermScreen.getBoundingClientRect();
            var x = touchX - rect.left;
            var y = touchY - rect.top;
            var cellWidth = rect.width / term.cols;
            var cellHeight = rect.height / term.rows;
            var col = Math.floor(x / cellWidth);
            var row = Math.floor(y / cellHeight);
            col = Math.max(0, Math.min(col, term.cols - 1));
            row = Math.max(0, Math.min(row, term.rows - 1));
            return { col: col, row: row };
        }

        function dispatchScroll(deltaY) {
            var evt = new WheelEvent('wheel', {
                deltaY: deltaY,
                deltaX: 0,
                deltaMode: 0, // DOM_DELTA_PIXEL
                bubbles: true,
                cancelable: true,
            });
            xtermScreen.dispatchEvent(evt);
        }

        function sampleVelocity(y) {
            velocityBuffer[velocityIdx % VELOCITY_BUFFER_SIZE] = {
                y: y,
                t: performance.now(),
            };
            velocityIdx++;
        }

        function computeVelocity() {
            var count = Math.min(velocityIdx, VELOCITY_BUFFER_SIZE);
            if (count < 2) return 0;

            // Get oldest and newest samples
            var newest = velocityBuffer[(velocityIdx - 1) % VELOCITY_BUFFER_SIZE];
            var oldestIdx = velocityIdx >= VELOCITY_BUFFER_SIZE
                ? velocityIdx % VELOCITY_BUFFER_SIZE
                : 0;
            var oldest = velocityBuffer[oldestIdx];

            var dt = newest.t - oldest.t;
            if (dt === 0) return 0;
            return (newest.y - oldest.y) / dt; // px/ms
        }

        function startMomentum(velocity) {
            state = STATE.MOMENTUM;
            var vel = velocity;
            var lastTime = performance.now();

            function step(now) {
                if (state !== STATE.MOMENTUM) return;

                var dt = now - lastTime;
                lastTime = now;

                vel *= Math.pow(MOMENTUM_DECAY, dt / 16); // normalize to ~60fps

                if (Math.abs(vel) < MOMENTUM_MIN_VELOCITY) {
                    state = STATE.IDLE;
                    return;
                }

                dispatchScroll(-vel * dt * SCROLL_MULTIPLIER);
                momentumRAF = requestAnimationFrame(step);
            }

            momentumRAF = requestAnimationFrame(step);
        }

        function cancelMomentum() {
            if (momentumRAF) {
                cancelAnimationFrame(momentumRAF);
                momentumRAF = null;
            }
            if (state === STATE.MOMENTUM) {
                state = STATE.IDLE;
            }
        }

        function cancelLongPress() {
            if (longPressTimer) {
                clearTimeout(longPressTimer);
                longPressTimer = null;
            }
        }


        function copyToClipboard(text) {
            if (navigator.clipboard && navigator.clipboard.writeText) {
                navigator.clipboard.writeText(text).catch(function() {
                    fallbackCopy(text);
                });
                return;
            }
            fallbackCopy(text);
        }

        function fallbackCopy(text) {
            var textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.cssText = 'position:fixed;left:-9999px;top:-9999px;opacity:0';
            document.body.appendChild(textarea);
            textarea.select();
            try { document.execCommand('copy'); } catch(e) { /* ignore */ }
            document.body.removeChild(textarea);
        }

        // --- Touch event handlers ---

        function onTouchStart(e) {
            // Multi-touch: let browser handle (pinch zoom)
            if (e.touches.length > 1) {
                cancelLongPress();
                cancelMomentum();
                state = STATE.IDLE;
                return;
            }

            var touch = e.touches[0];

            // Ignore touches on pane toolbar, resize handles
            var target = touch.target || e.target;
            if (target.closest && (
                target.closest('.pane-toolbar') ||
                target.closest('.pane-resize-handle') ||
                target.closest('.modifier-toolbar')
            )) {
                return;
            }

            // Cancel any existing momentum
            cancelMomentum();

            startX = touch.clientX;
            startY = touch.clientY;
            lastY = touch.clientY;

            // Reset velocity buffer
            velocityBuffer = [];
            velocityIdx = 0;
            sampleVelocity(touch.clientY);

            state = STATE.PENDING;

            // Start long-press timer for text selection
            longPressTimer = setTimeout(function() {
                if (state !== STATE.PENDING) return;

                state = STATE.SELECTING;
                container.classList.add('touch-selecting');

                // Haptic feedback if available
                if (navigator.vibrate) {
                    navigator.vibrate(50);
                }

                // Record selection start position
                var cell = touchToCell(startX, startY);
                selStartCol = cell.col;
                selStartRow = cell.row;

                // Initialize selection at touch point
                var bufferRow = term.buffer.active.viewportY + cell.row;
                term.select(cell.col, bufferRow, 1);
            }, LONG_PRESS_MS);
        }

        function onTouchMove(e) {
            if (e.touches.length > 1) {
                // Multi-touch appeared during gesture; bail out
                cancelLongPress();
                if (state === STATE.SCROLLING || state === STATE.SELECTING) {
                    container.classList.remove('touch-selecting');
                    state = STATE.IDLE;
                }
                return;
            }

            var touch = e.touches[0];

            if (state === STATE.PENDING) {
                var dx = touch.clientX - startX;
                var dy = touch.clientY - startY;

                // Classify gesture by dominant axis once past threshold
                if (Math.abs(dy) > SWIPE_THRESHOLD || Math.abs(dx) > SWIPE_THRESHOLD) {
                    cancelLongPress();

                    if (Math.abs(dy) >= Math.abs(dx)) {
                        // Vertical swipe -> scroll
                        state = STATE.SCROLLING;
                        e.preventDefault();
                    } else if (window.ClawIDEPaneLayout && window.ClawIDEPaneLayout.isPhoneLayout()) {
                        // Horizontal swipe in phone carousel mode -> track for navigation
                        isCarouselSwipe = true;
                        state = STATE.SCROLLING; // reuse scrolling state to block other gestures
                        e.preventDefault();
                    } else {
                        // Horizontal swipe on desktop -> let browser/xterm handle
                        state = STATE.IDLE;
                        return;
                    }
                }
            }

            if (state === STATE.SCROLLING) {
                e.preventDefault();
                var deltaY = lastY - touch.clientY;
                lastY = touch.clientY;
                sampleVelocity(touch.clientY);
                dispatchScroll(deltaY * SCROLL_MULTIPLIER);
            }

            if (state === STATE.SELECTING) {
                e.preventDefault();
                var cell = touchToCell(touch.clientX, touch.clientY);
                var bufRow = term.buffer.active.viewportY + cell.row;
                var startBufRow = term.buffer.active.viewportY + selStartRow;

                // Calculate selection span
                var startOffset = startBufRow * term.cols + selStartCol;
                var endOffset = bufRow * term.cols + cell.col;

                if (endOffset >= startOffset) {
                    term.select(selStartCol, startBufRow, endOffset - startOffset + 1);
                } else {
                    term.select(cell.col, bufRow, startOffset - endOffset + 1);
                }
            }
        }

        function onTouchEnd(e) {
            cancelLongPress();

            if (state === STATE.PENDING) {
                // Short tap — no significant movement, let xterm handle focus/click
                state = STATE.IDLE;
                return;
            }

            if (state === STATE.SCROLLING) {
                if (isCarouselSwipe) {
                    // Horizontal swipe in carousel mode -> navigate panes
                    isCarouselSwipe = false;
                    state = STATE.IDLE;
                    var lastTouch = e.changedTouches[0];
                    if (lastTouch) {
                        var swipeDx = lastTouch.clientX - startX;
                        if (Math.abs(swipeDx) > 50 && window.ClawIDEPaneLayout) {
                            // Find the session ID from the closest session pane container
                            var csStates = window.ClawIDEPaneLayout.getCarouselState();
                            var sessionIDs = Object.keys(csStates);
                            for (var si = 0; si < sessionIDs.length; si++) {
                                var cs = csStates[sessionIDs[si]];
                                if (cs) {
                                    // Swipe left = next, swipe right = prev
                                    var newIdx = swipeDx < 0 ? cs.currentIndex + 1 : cs.currentIndex - 1;
                                    window.ClawIDEPaneLayout.navigateCarousel(sessionIDs[si], newIdx);
                                    break;
                                }
                            }
                        }
                    }
                    return;
                }

                var velocity = computeVelocity();
                if (Math.abs(velocity) > MOMENTUM_MIN_VELOCITY) {
                    startMomentum(velocity);
                } else {
                    state = STATE.IDLE;
                }
                return;
            }

            if (state === STATE.SELECTING) {
                container.classList.remove('touch-selecting');
                state = STATE.IDLE;

                var selection = term.getSelection();
                if (selection) {
                    // Auto-copy and show toast notification
                    copyToClipboard(selection);
                    if (window.ClawIDEToast) {
                        window.ClawIDEToast.show('✓ Copied');
                    }
                }
                return;
            }

            state = STATE.IDLE;
        }

        function onTouchCancel() {
            cancelLongPress();
            cancelMomentum();
            isCarouselSwipe = false;
            container.classList.remove('touch-selecting');
            state = STATE.IDLE;
        }

        // --- Attach listeners with capture to intercept before xterm ---
        container.addEventListener('touchstart', onTouchStart, { capture: true, passive: false });
        container.addEventListener('touchmove', onTouchMove, { capture: true, passive: false });
        container.addEventListener('touchend', onTouchEnd, { capture: true, passive: false });
        container.addEventListener('touchcancel', onTouchCancel, { capture: true, passive: false });
    }

    // Expose globally
    window.ClawIDETouchTerminal = {
        attach: attach,
    };
})();
