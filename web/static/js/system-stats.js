// formatBytes converts a byte count to a human-readable string (e.g. "8.0 GB").
function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    var units = ['B', 'KB', 'MB', 'GB', 'TB'];
    var i = Math.floor(Math.log(bytes) / Math.log(1024));
    if (i >= units.length) i = units.length - 1;
    return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i];
}

// generateQR renders a QR code SVG into the given container element.
// Caches by data-qr-text attribute so the 10s poll cycle doesn't re-render.
function generateQR(container, text, size) {
    if (!container || !text) return;
    if (container.getAttribute('data-qr-text') === text) return;
    var qr = qrcode(0, 'M');
    qr.addData(text);
    qr.make();
    container.innerHTML = qr.createSvgTag({ cellSize: size || 4, margin: 0 });
    container.setAttribute('data-qr-text', text);
}
