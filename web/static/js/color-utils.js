/**
 * ClawIDEColor — Client-side color shade generation and feature color picker.
 * Mirrors the Go internal/color package for consistent HSL shade generation.
 */
var ClawIDEColor = (function() {
    'use strict';

    function hexToRGB(hex) {
        hex = hex.replace(/^#/, '');
        if (hex.length !== 6) return null;
        var r = parseInt(hex.substring(0, 2), 16);
        var g = parseInt(hex.substring(2, 4), 16);
        var b = parseInt(hex.substring(4, 6), 16);
        if (isNaN(r) || isNaN(g) || isNaN(b)) return null;
        return { r: r, g: g, b: b };
    }

    function rgbToHex(r, g, b) {
        return '#' + [r, g, b].map(function(c) {
            var hex = Math.round(c).toString(16);
            return hex.length === 1 ? '0' + hex : hex;
        }).join('');
    }

    function rgbToHSL(r, g, b) {
        r /= 255; g /= 255; b /= 255;
        var max = Math.max(r, g, b);
        var min = Math.min(r, g, b);
        var h = 0, s = 0, l = (max + min) / 2;

        if (max !== min) {
            var d = max - min;
            s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
            if (max === r) {
                h = (g - b) / d + (g < b ? 6 : 0);
            } else if (max === g) {
                h = (b - r) / d + 2;
            } else {
                h = (r - g) / d + 4;
            }
            h *= 60;
        }
        return { h: h, s: s, l: l };
    }

    function hslToRGB(h, s, l) {
        if (s === 0) {
            var v = Math.round(l * 255);
            return { r: v, g: v, b: v };
        }
        var q = l < 0.5 ? l * (1 + s) : l + s - l * s;
        var p = 2 * l - q;
        var hNorm = h / 360;

        function toRGB(t) {
            if (t < 0) t += 1;
            if (t > 1) t -= 1;
            if (t < 1/6) return p + (q - p) * 6 * t;
            if (t < 1/2) return q;
            if (t < 2/3) return p + (q - p) * (2/3 - t) * 6;
            return p;
        }

        return {
            r: Math.round(toRGB(hNorm + 1/3) * 255),
            g: Math.round(toRGB(hNorm) * 255),
            b: Math.round(toRGB(hNorm - 1/3) * 255)
        };
    }

    function hexToHSL(hex) {
        var rgb = hexToRGB(hex);
        if (!rgb) return null;
        return rgbToHSL(rgb.r, rgb.g, rgb.b);
    }

    function hslToHex(h, s, l) {
        var rgb = hslToRGB(h, s, l);
        return rgbToHex(rgb.r, rgb.g, rgb.b);
    }

    /**
     * Generate n shade hex strings from a base color by varying lightness 25%–85%.
     */
    function generateShades(hex, n) {
        n = n || 8;
        var rgb = hexToRGB(hex);
        if (!rgb) return [];
        var hsl = rgbToHSL(rgb.r, rgb.g, rgb.b);
        var shades = [];
        for (var i = 0; i < n; i++) {
            var l = n === 1 ? 0.55 : 0.25 + (0.60 * i / (n - 1));
            shades.push(hslToHex(hsl.h, hsl.s, l));
        }
        return shades;
    }

    /**
     * Populate a container with shade swatches for a feature color picker.
     * @param {string} containerId - DOM element ID to populate
     * @param {string} projectColor - The project's base hex color
     * @param {string} featureId - The feature ID
     * @param {string} projectId - The project ID
     * @param {string} currentColor - The feature's current color (for highlighting)
     */
    function renderShadePicker(containerId, projectColor, featureId, projectId, currentColor) {
        var container = document.getElementById(containerId);
        if (!container || !projectColor) return;

        var shades = generateShades(projectColor, 8);
        container.innerHTML = '';

        var label = document.createElement('p');
        label.className = 'text-[10px] text-gray-500 uppercase mb-1.5';
        label.textContent = 'Feature Color';
        container.appendChild(label);

        var grid = document.createElement('div');
        grid.className = 'grid grid-cols-5 gap-1';

        shades.forEach(function(shade) {
            var btn = document.createElement('button');
            btn.className = 'w-5 h-5 rounded-full hover:ring-2 ring-white/50 transition-all';
            btn.style.backgroundColor = shade;
            if (currentColor && shade.toLowerCase() === currentColor.toLowerCase()) {
                btn.className += ' ring-2 ring-white';
            }
            btn.addEventListener('click', function(e) {
                e.preventDefault();
                e.stopPropagation();
                setFeatureColor(projectId, featureId, shade);
            });
            grid.appendChild(btn);
        });

        // Clear button
        var clearBtn = document.createElement('button');
        clearBtn.className = 'w-5 h-5 rounded-full border border-gray-600 hover:ring-2 ring-white/50 flex items-center justify-center';
        clearBtn.title = 'Clear color';
        clearBtn.innerHTML = '<svg class="w-3 h-3 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/></svg>';
        clearBtn.addEventListener('click', function(e) {
            e.preventDefault();
            e.stopPropagation();
            setFeatureColor(projectId, featureId, '');
        });
        grid.appendChild(clearBtn);

        container.appendChild(grid);
    }

    /**
     * PATCH the feature color via API, then reload the page.
     */
    function setFeatureColor(projectId, featureId, colorHex) {
        fetch('/projects/' + projectId + '/features/' + featureId + '/color', {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ color: colorHex })
        }).then(function() {
            location.reload();
        }).catch(function(err) {
            console.error('Failed to update feature color:', err);
        });
    }

    return {
        hexToHSL: hexToHSL,
        hslToHex: hslToHex,
        generateShades: generateShades,
        renderShadePicker: renderShadePicker,
        setFeatureColor: setFeatureColor
    };
})();
