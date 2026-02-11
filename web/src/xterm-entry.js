import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';

// Export to window for use by terminal.js
window.XtermTerminal = Terminal;
window.XtermFitAddon = FitAddon;
window.XtermWebLinksAddon = WebLinksAddon;
