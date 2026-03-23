// ClawIDE Command Palette — VS Code-style unified palette
// Cmd+P = file search (default), Cmd+Shift+P = command mode (> prefix)
// Typing ">" switches to command mode; removing it switches back to file search.
(function() {
    'use strict';

    var RECENT_COMMANDS_KEY = 'editor.preferences.recentCommands';
    var RECENT_FILES_KEY = 'editor.preferences.recentFiles';
    var MAX_RECENT_COMMANDS = 5;
    var MAX_RECENT_FILES = 10;

    // --- Heroicon SVG paths (outline, 24x24 viewBox) ---
    var ICONS = {
        sort: '<path stroke-linecap="round" stroke-linejoin="round" d="M3 4h13M3 8h9m-9 4h6m4 0l4-4m0 0l4 4m-4-4v12"/>',
        text: '<path stroke-linecap="round" stroke-linejoin="round" d="M4 6h16M4 12h16M4 18h7"/>',
        line: '<path stroke-linecap="round" stroke-linejoin="round" d="M5 12h14"/>',
        copy: '<path stroke-linecap="round" stroke-linejoin="round" d="M15.666 3.888A2.25 2.25 0 0 0 13.5 2.25h-3c-1.03 0-1.9.693-2.166 1.638m7.332 0c.055.194.084.4.084.612v0a.75.75 0 0 1-.75.75H9a.75.75 0 0 1-.75-.75v0c0-.212.03-.418.084-.612m7.332 0c.646.049 1.288.11 1.927.184 1.1.128 1.907 1.077 1.907 2.185V19.5a2.25 2.25 0 0 1-2.25 2.25H6.75A2.25 2.25 0 0 1 4.5 19.5V6.257c0-1.108.806-2.057 1.907-2.185a48.208 48.208 0 0 1 1.927-.184"/>',
        navigate: '<path stroke-linecap="round" stroke-linejoin="round" d="M3 4.5h14.25M3 9h9.75M3 13.5h5.25m5.25-.75L17.25 9m0 0L21 12.75M17.25 9v12"/>',
        select: '<path stroke-linecap="round" stroke-linejoin="round" d="M15.042 21.672L13.684 16.6m0 0l-2.51 2.225.569-9.47 5.227 7.917-3.286-.672zM12 2.25V4.5m5.834.166l-1.591 1.591M20.25 10.5H18M7.757 14.743l-1.59 1.59M6 10.5H3.75m4.007-4.243l-1.59-1.59"/>',
        indent: '<path stroke-linecap="round" stroke-linejoin="round" d="M17.25 8.25L21 12m0 0l-3.75 3.75M21 12H3"/>',
        comment: '<path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5h10.5m-10.5 3h7.5m-7.5 3h4.5M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"/>',
        file: '<path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9z"/>',
        transform: '<path stroke-linecap="round" stroke-linejoin="round" d="M7.5 21L3 16.5m0 0L7.5 12M3 16.5h13.5m0-13.5L21 7.5m0 0L16.5 12M21 7.5H7.5"/>',
        delete: '<path stroke-linecap="round" stroke-linejoin="round" d="M14.74 9l-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0"/>',
        sidebar: '<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6A2.25 2.25 0 0 1 6 3.75h2.25A2.25 2.25 0 0 1 10.5 6v12a2.25 2.25 0 0 1-2.25 2.25H6A2.25 2.25 0 0 1 3.75 18V6zM10.5 3.75h7.5A2.25 2.25 0 0 1 20.25 6v12a2.25 2.25 0 0 1-2.25 2.25h-7.5"/>',
    };

    function makeIcon(pathKey) {
        return '<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">' + (ICONS[pathKey] || ICONS.text) + '</svg>';
    }

    // --- File Type Icons ---
    var FILE_TYPE_ICONS = {
        // Code
        js: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#EAB308" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        ts: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#3B82F6" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        jsx: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#06B6D4" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        tsx: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#06B6D4" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        go: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#06B6D4" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        py: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#3B82F6" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        rb: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#EF4444" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        rs: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F97316" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        java: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F97316" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        c: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#64748B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        cpp: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#64748B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        h: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#64748B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        php: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#8B5CF6" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        swift: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F97316" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M17.25 6.75L22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3l-4.5 16.5"/></svg>',
        sh: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3M3 20.25V3.75A2.25 2.25 0 015.25 1.5h13.5A2.25 2.25 0 0121 3.75v16.5A2.25 2.25 0 0118.75 22.5H5.25A2.25 2.25 0 013 20.25z"/></svg>',
        bash: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3M3 20.25V3.75A2.25 2.25 0 015.25 1.5h13.5A2.25 2.25 0 0121 3.75v16.5A2.25 2.25 0 0118.75 22.5H5.25A2.25 2.25 0 013 20.25z"/></svg>',
        zsh: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M6.75 7.5l3 2.25-3 2.25m4.5 0h3M3 20.25V3.75A2.25 2.25 0 015.25 1.5h13.5A2.25 2.25 0 0121 3.75v16.5A2.25 2.25 0 0118.75 22.5H5.25A2.25 2.25 0 013 20.25z"/></svg>',
        // Web
        html: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F97316" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M12 21a9.004 9.004 0 008.716-6.747M12 21a9.004 9.004 0 01-8.716-6.747M12 21c2.485 0 4.5-4.03 4.5-9S14.485 3 12 3m0 18c-2.485 0-4.5-4.03-4.5-9S9.515 3 12 3m0 0a8.997 8.997 0 017.843 4.582M12 3a8.997 8.997 0 00-7.843 4.582m15.686 0A11.953 11.953 0 0112 10.5c-2.998 0-5.74-1.1-7.843-2.918m15.686 0A8.959 8.959 0 0121 12c0 .778-.099 1.533-.284 2.253m0 0A17.919 17.919 0 0112 16.5a17.92 17.92 0 01-8.716-2.247m0 0A8.966 8.966 0 013 12c0-1.264.26-2.466.732-3.558"/></svg>',
        css: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#8B5CF6" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.53 16.122a3 3 0 00-5.78 1.128 2.25 2.25 0 01-2.4 2.245 4.5 4.5 0 008.4-2.245c0-.399-.078-.78-.22-1.128zm0 0a15.998 15.998 0 003.388-1.62m-5.043-.025a15.994 15.994 0 011.622-3.395m3.42 3.42a15.995 15.995 0 004.764-4.648l3.876-5.814a1.151 1.151 0 00-1.597-1.597L14.146 6.32a15.996 15.996 0 00-4.649 4.764m3.42 3.42a6.776 6.776 0 00-3.42-3.42"/></svg>',
        scss: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#EC4899" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.53 16.122a3 3 0 00-5.78 1.128 2.25 2.25 0 01-2.4 2.245 4.5 4.5 0 008.4-2.245c0-.399-.078-.78-.22-1.128zm0 0a15.998 15.998 0 003.388-1.62m-5.043-.025a15.994 15.994 0 011.622-3.395m3.42 3.42a15.995 15.995 0 004.764-4.648l3.876-5.814a1.151 1.151 0 00-1.597-1.597L14.146 6.32a15.996 15.996 0 00-4.649 4.764m3.42 3.42a6.776 6.776 0 00-3.42-3.42"/></svg>',
        less: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#3B82F6" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M9.53 16.122a3 3 0 00-5.78 1.128 2.25 2.25 0 01-2.4 2.245 4.5 4.5 0 008.4-2.245c0-.399-.078-.78-.22-1.128zm0 0a15.998 15.998 0 003.388-1.62m-5.043-.025a15.994 15.994 0 011.622-3.395m3.42 3.42a15.995 15.995 0 004.764-4.648l3.876-5.814a1.151 1.151 0 00-1.597-1.597L14.146 6.32a15.996 15.996 0 00-4.649 4.764m3.42 3.42a6.776 6.776 0 00-3.42-3.42"/></svg>',
        // Config
        json: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#EAB308" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M11.42 15.17l-5.1-5.1a1.5 1.5 0 010-2.12l.88-.88a1.5 1.5 0 012.12 0l2.83 2.83 5.66-5.66a1.5 1.5 0 012.12 0l.88.88a1.5 1.5 0 010 2.12l-7.78 7.78a1.5 1.5 0 01-2.12 0z"/></svg>',
        yaml: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#EF4444" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.343 3.94c.09-.542.56-.94 1.11-.94h1.093c.55 0 1.02.398 1.11.94l.149.894c.07.424.384.764.78.93.398.164.855.142 1.205-.108l.737-.527a1.125 1.125 0 011.45.12l.773.774c.39.389.44 1.002.12 1.45l-.527.737c-.25.35-.272.806-.107 1.204.165.397.505.71.93.78l.893.15c.543.09.94.56.94 1.109v1.094c0 .55-.397 1.02-.94 1.11l-.893.149c-.425.07-.765.383-.93.78-.165.398-.143.854.107 1.204l.527.738c.32.447.269 1.06-.12 1.45l-.774.773a1.125 1.125 0 01-1.449.12l-.738-.527c-.35-.25-.806-.272-1.204-.107-.397.165-.71.505-.78.929l-.15.894c-.09.542-.56.94-1.11.94h-1.094c-.55 0-1.019-.398-1.11-.94l-.148-.894c-.071-.424-.384-.764-.781-.93-.398-.164-.854-.142-1.204.108l-.738.527c-.447.32-1.06.269-1.45-.12l-.773-.774a1.125 1.125 0 01-.12-1.45l.527-.737c.25-.35.273-.806.108-1.204-.165-.397-.506-.71-.93-.78l-.894-.15c-.542-.09-.94-.56-.94-1.109v-1.094c0-.55.398-1.02.94-1.11l.894-.149c.424-.07.765-.383.93-.78.165-.398.143-.854-.107-1.204l-.527-.738a1.125 1.125 0 01.12-1.45l.773-.773a1.125 1.125 0 011.45-.12l.737.527c.35.25.807.272 1.204.107.397-.165.71-.505.78-.929l.15-.894z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>',
        yml: null, // will fall through to yaml
        toml: null,
        ini: null,
        env: null,
        // Document
        md: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#64748B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m0 12.75h7.5m-7.5 3H12M10.5 2.25H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"/></svg>',
        txt: null,
        rst: null,
        // Image
        png: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#10B981" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M2.25 15.75l5.159-5.159a2.25 2.25 0 013.182 0l5.159 5.159m-1.5-1.5l1.409-1.409a2.25 2.25 0 013.182 0l2.909 2.909m-18 3.75h16.5a1.5 1.5 0 001.5-1.5V6a1.5 1.5 0 00-1.5-1.5H3.75A1.5 1.5 0 002.25 6v12a1.5 1.5 0 001.5 1.5zm10.5-11.25h.008v.008h-.008V8.25zm.375 0a.375.375 0 11-.75 0 .375.375 0 01.75 0z"/></svg>',
        jpg: null,
        jpeg: null,
        gif: null,
        svg: null,
        ico: null,
        webp: null,
        // Data
        sql: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F59E0B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M20.25 6.375c0 2.278-3.694 4.125-8.25 4.125S3.75 8.653 3.75 6.375m16.5 0c0-2.278-3.694-4.125-8.25-4.125S3.75 4.097 3.75 6.375m16.5 0v11.25c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125V6.375m16.5 0v3.75m-16.5-3.75v3.75m16.5 0v3.75C20.25 16.153 16.556 18 12 18s-8.25-1.847-8.25-4.125v-3.75m16.5 0c0 2.278-3.694 4.125-8.25 4.125s-8.25-1.847-8.25-4.125"/></svg>',
        csv: null,
        // Build / Config
        mod: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#64748B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M10.343 3.94c.09-.542.56-.94 1.11-.94h1.093c.55 0 1.02.398 1.11.94l.149.894c.07.424.384.764.78.93.398.164.855.142 1.205-.108l.737-.527a1.125 1.125 0 011.45.12l.773.774c.39.389.44 1.002.12 1.45l-.527.737c-.25.35-.272.806-.107 1.204.165.397.505.71.93.78l.893.15c.543.09.94.56.94 1.109v1.094c0 .55-.397 1.02-.94 1.11l-.893.149c-.425.07-.765.383-.93.78-.165.398-.143.854.107 1.204l.527.738c.32.447.269 1.06-.12 1.45l-.774.773a1.125 1.125 0 01-1.449.12l-.738-.527c-.35-.25-.806-.272-1.204-.107-.397.165-.71.505-.78.929l-.15.894c-.09.542-.56.94-1.11.94h-1.094c-.55 0-1.019-.398-1.11-.94l-.148-.894c-.071-.424-.384-.764-.781-.93-.398-.164-.854-.142-1.204.108l-.738.527c-.447.32-1.06.269-1.45-.12l-.773-.774a1.125 1.125 0 01-.12-1.45l.527-.737c.25-.35.273-.806.108-1.204-.165-.397-.506-.71-.93-.78l-.894-.15c-.542-.09-.94-.56-.94-1.109v-1.094c0-.55.398-1.02.94-1.11l.894-.149c.424-.07.765-.383.93-.78.165-.398.143-.854-.107-1.204l-.527-.738a1.125 1.125 0 01.12-1.45l.773-.773a1.125 1.125 0 011.45-.12l.737.527c.35.25.807.272 1.204.107.397-.165.71-.505.78-.929l.15-.894z"/><path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"/></svg>',
        sum: null,
        lock: null,
        // Folder
        _folder: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#F59E0B" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"/></svg>',
        // Default
        _default: '<svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="#94A3B8" stroke-width="1.5"><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 14.25v-2.625a3.375 3.375 0 00-3.375-3.375h-1.5A1.125 1.125 0 0113.5 7.125v-1.5a3.375 3.375 0 00-3.375-3.375H8.25m2.25 0H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 00-9-9z"/></svg>',
    };

    // Extension aliases — map to an icon that already has SVG defined
    var EXT_ALIASES = {
        yml: 'yaml', toml: 'yaml', ini: 'yaml', env: 'yaml',
        txt: 'md', rst: 'md',
        jpg: 'png', jpeg: 'png', gif: 'png', svg: 'png', ico: 'png', webp: 'png',
        csv: 'sql',
        sum: 'mod', lock: 'mod',
        mjs: 'js', cjs: 'js',
    };

    function getFileIcon(filename, isDir) {
        if (isDir) return FILE_TYPE_ICONS._folder;
        var ext = (filename.split('.').pop() || '').toLowerCase();
        // Special files
        if (filename === 'Makefile' || filename === 'Dockerfile' || filename === 'Vagrantfile') return FILE_TYPE_ICONS.mod;
        if (filename === '.gitignore' || filename === '.dockerignore') return FILE_TYPE_ICONS.mod;
        var icon = FILE_TYPE_ICONS[ext];
        if (icon) return icon;
        if (EXT_ALIASES[ext]) return FILE_TYPE_ICONS[EXT_ALIASES[ext]];
        return FILE_TYPE_ICONS._default;
    }

    // --- Command Registry ---
    var commands = [
        // Text Transformations
        { id: 'sortLinesAscending', name: 'Sort Lines Ascending', category: 'Text', icon: makeIcon('sort'), shortcut: '', handler: 'sortLinesAscending' },
        { id: 'sortLinesDescending', name: 'Sort Lines Descending', category: 'Text', icon: makeIcon('sort'), shortcut: '', handler: 'sortLinesDescending' },
        { id: 'transformToUppercase', name: 'Transform to UPPERCASE', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToUppercase' },
        { id: 'transformToLowercase', name: 'Transform to lowercase', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToLowercase' },
        { id: 'transformToTitleCase', name: 'Transform to Title Case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToTitleCase' },
        { id: 'transformToSnakeCase', name: 'Transform to snake_case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToSnakeCase' },
        { id: 'transformToCamelCase', name: 'Transform to camelCase', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToCamelCase' },
        { id: 'transformToKebabCase', name: 'Transform to kebab-case', category: 'Text', icon: makeIcon('transform'), shortcut: '', handler: 'transformToKebabCase' },
        { id: 'trimTrailingWhitespace', name: 'Trim Trailing Whitespace', category: 'Text', icon: makeIcon('text'), shortcut: '', handler: 'trimTrailingWhitespace' },
        { id: 'deleteEmptyLines', name: 'Delete Empty Lines', category: 'Text', icon: makeIcon('delete'), shortcut: '', handler: 'deleteEmptyLines' },

        // Line Operations
        { id: 'duplicateLine', name: 'Duplicate Line', category: 'Line', icon: makeIcon('copy'), shortcut: '', handler: 'duplicateLine' },
        { id: 'deleteLine', name: 'Delete Line', category: 'Line', icon: makeIcon('delete'), shortcut: '', handler: 'deleteLine' },
        { id: 'joinLines', name: 'Join Lines', category: 'Line', icon: makeIcon('line'), shortcut: '', handler: 'joinLines' },
        { id: 'reverseLines', name: 'Reverse Lines', category: 'Line', icon: makeIcon('sort'), shortcut: '', handler: 'reverseLines' },
        { id: 'removeDuplicateLines', name: 'Remove Duplicate Lines', category: 'Line', icon: makeIcon('delete'), shortcut: '', handler: 'removeDuplicateLines' },
        { id: 'indentSelection', name: 'Indent Selection', category: 'Line', icon: makeIcon('indent'), shortcut: 'Tab', handler: 'indentSelection' },
        { id: 'outdentSelection', name: 'Outdent Selection', category: 'Line', icon: makeIcon('indent'), shortcut: 'Shift+Tab', handler: 'outdentSelection' },
        { id: 'toggleComment', name: 'Toggle Comment', category: 'Line', icon: makeIcon('comment'), shortcut: 'Cmd+/', handler: 'toggleComment' },

        // Navigation
        { id: 'goToLine', name: 'Go to Line...', category: 'Navigation', icon: makeIcon('navigate'), shortcut: 'Ctrl+G', handler: 'goToLine' },

        // File Operations
        { id: 'newFile', name: 'New File', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'newFile' },
        { id: 'newFolder', name: 'New Folder', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'newFolder' },

        // File Info
        { id: 'copyFilePath', name: 'Copy File Path', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyFilePath' },
        { id: 'copyRelativePath', name: 'Copy Relative Path', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyRelativePath' },
        { id: 'copyFileName', name: 'Copy File Name', category: 'File', icon: makeIcon('file'), shortcut: '', handler: 'copyFileName' },

        // Selection
        { id: 'selectAll', name: 'Select All', category: 'Selection', icon: makeIcon('select'), shortcut: 'Cmd+A', handler: 'selectAll' },
        { id: 'selectLine', name: 'Select Line', category: 'Selection', icon: makeIcon('select'), shortcut: 'Cmd+L', handler: 'selectLine' },
        { id: 'selectWord', name: 'Select Word', category: 'Selection', icon: makeIcon('select'), shortcut: '', handler: 'selectWord' },

        // View
        { id: 'toggleSidebar', name: 'Toggle Sidebar', category: 'View', icon: makeIcon('sidebar'), shortcut: 'Cmd+B', handler: 'toggleSidebar' },
    ];

    // --- Fuzzy Search ---
    function fuzzyMatch(query, text) {
        query = query.toLowerCase();
        text = text.toLowerCase();

        if (text === query) return { match: true, score: 100 };
        if (text.indexOf(query) === 0) return { match: true, score: 80 };
        if (text.indexOf(query) !== -1) return { match: true, score: 60 };

        var qi = 0;
        var score = 0;
        var consecutive = 0;
        for (var ti = 0; ti < text.length && qi < query.length; ti++) {
            if (text.charAt(ti) === query.charAt(qi)) {
                qi++;
                consecutive++;
                score += consecutive * 2;
            } else {
                consecutive = 0;
            }
        }

        if (qi === query.length) {
            return { match: true, score: score };
        }

        return { match: false, score: 0 };
    }

    function searchCommands(query, cmds) {
        if (!query || query.trim() === '') return cmds;

        var results = [];
        for (var i = 0; i < cmds.length; i++) {
            var nameMatch = fuzzyMatch(query, cmds[i].name);
            var catMatch = fuzzyMatch(query, cmds[i].category);
            var bestScore = Math.max(nameMatch.score, catMatch.score);
            if (nameMatch.match || catMatch.match) {
                results.push({ command: cmds[i], score: bestScore });
            }
        }

        results.sort(function(a, b) { return b.score - a.score; });
        return results.map(function(r) { return r.command; });
    }

    // --- Fuzzy Highlight ---
    // Returns HTML with matched characters wrapped in <span class="text-indigo-400">
    function fuzzyHighlight(text, query) {
        if (!query) return escapeHtml(text);
        var qLower = query.toLowerCase();
        var tLower = text.toLowerCase();

        // If substring match, highlight the substring
        var idx = tLower.indexOf(qLower);
        if (idx !== -1) {
            return escapeHtml(text.substring(0, idx)) +
                '<span class="text-indigo-400 font-semibold">' + escapeHtml(text.substring(idx, idx + query.length)) + '</span>' +
                escapeHtml(text.substring(idx + query.length));
        }

        // Fuzzy character-by-character highlight
        var result = '';
        var qi = 0;
        for (var i = 0; i < text.length; i++) {
            if (qi < qLower.length && tLower.charAt(i) === qLower.charAt(qi)) {
                result += '<span class="text-indigo-400 font-semibold">' + escapeHtml(text.charAt(i)) + '</span>';
                qi++;
            } else {
                result += escapeHtml(text.charAt(i));
            }
        }
        return result;
    }

    function escapeHtml(str) {
        return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
    }

    // --- Recent Commands ---
    function loadRecentCommands() {
        try {
            var stored = localStorage.getItem(RECENT_COMMANDS_KEY);
            return stored ? JSON.parse(stored) : [];
        } catch (e) {
            return [];
        }
    }

    function saveRecentCommands(recentIds) {
        try {
            localStorage.setItem(RECENT_COMMANDS_KEY, JSON.stringify(recentIds.slice(0, MAX_RECENT_COMMANDS)));
        } catch (e) { /* ignore */ }
    }

    function addToRecentCommands(commandId) {
        var recent = loadRecentCommands();
        recent = recent.filter(function(id) { return id !== commandId; });
        recent.unshift(commandId);
        saveRecentCommands(recent);
    }

    function getRecentCommandObjects() {
        var recentIds = loadRecentCommands();
        var result = [];
        for (var i = 0; i < recentIds.length; i++) {
            for (var j = 0; j < commands.length; j++) {
                if (commands[j].id === recentIds[i]) {
                    result.push(commands[j]);
                    break;
                }
            }
        }
        return result;
    }

    // --- Recent Files ---
    function loadRecentFiles() {
        try {
            var stored = localStorage.getItem(RECENT_FILES_KEY);
            return stored ? JSON.parse(stored) : [];
        } catch (e) {
            return [];
        }
    }

    function saveRecentFiles(files) {
        try {
            localStorage.setItem(RECENT_FILES_KEY, JSON.stringify(files.slice(0, MAX_RECENT_FILES)));
        } catch (e) { /* ignore */ }
    }

    function addToRecentFiles(name, path) {
        var recent = loadRecentFiles();
        recent = recent.filter(function(f) { return f.path !== path; });
        recent.unshift({ name: name, path: path });
        saveRecentFiles(recent);
    }

    // --- Execute Command by ID ---
    function runCommandById(commandId) {
        var cmd = null;
        for (var i = 0; i < commands.length; i++) {
            if (commands[i].id === commandId) {
                cmd = commands[i];
                break;
            }
        }
        if (!cmd) return false;

        addToRecentCommands(commandId);

        if (typeof window.ClawIDECommands !== 'undefined' && typeof window.ClawIDECommands[cmd.handler] === 'function') {
            return window.ClawIDECommands[cmd.handler]();
        }
        console.warn('Command handler not found:', cmd.handler);
        return false;
    }

    // --- File size formatting ---
    function formatFileSize(bytes) {
        if (bytes === 0) return '0 B';
        if (bytes < 1024) return bytes + ' B';
        if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
        return (bytes / 1048576).toFixed(1) + ' MB';
    }

    // --- Alpine.js Component Data ---
    window._clawIDECommandPaletteData = function() {
        return {
            open: false,
            query: '',
            selectedIndex: 0,
            filteredCommands: [],
            recentCommands: [],
            recentFiles: [],

            // File search state
            fileSearchResults: [],
            fileSearchLoading: false,
            fileSearchDebounceTimer: null,

            // Computed-like getters
            get isCommandMode() {
                return this.query.charAt(0) === '>';
            },

            get commandQuery() {
                if (!this.isCommandMode) return '';
                return this.query.substring(1).trim();
            },

            get fileQuery() {
                if (this.isCommandMode) return '';
                return this.query.trim();
            },

            get currentListLength() {
                if (this.isCommandMode) return this.filteredCommands.length;
                if (this.fileQuery) return this.fileSearchResults.length;
                return this.recentFiles.length;
            },

            init: function() {
                this.recentCommands = getRecentCommandObjects();
                this.recentFiles = loadRecentFiles();
            },

            // Open in file search mode (Cmd+P)
            openPalette: function() {
                this.open = true;
                this.query = '';
                this.selectedIndex = 0;
                this.recentCommands = getRecentCommandObjects();
                this.recentFiles = loadRecentFiles();
                this.fileSearchResults = [];
                this.fileSearchLoading = false;
                this.filteredCommands = [];
                var self = this;
                this.$nextTick(function() {
                    var input = document.getElementById('command-palette-search');
                    if (input) input.focus();
                });
            },

            // Open in command mode (Cmd+Shift+P)
            openCommandMode: function() {
                this.open = true;
                this.query = '>';
                this.selectedIndex = 0;
                this.recentCommands = getRecentCommandObjects();
                this.recentFiles = loadRecentFiles();
                this.fileSearchResults = [];
                this.fileSearchLoading = false;
                this.updateFilteredCommands();
                var self = this;
                this.$nextTick(function() {
                    var input = document.getElementById('command-palette-search');
                    if (input) input.focus();
                });
            },

            // Legacy method — kept for backward compat with command palette public API
            openFileSearch: function() {
                this.openPalette();
            },

            close: function() {
                this.open = false;
                this.query = '';
                this.selectedIndex = 0;
                this.fileSearchResults = [];
                this.fileSearchLoading = false;
                if (this.fileSearchDebounceTimer) {
                    clearTimeout(this.fileSearchDebounceTimer);
                    this.fileSearchDebounceTimer = null;
                }
            },

            onSearchInput: function() {
                this.selectedIndex = 0;
                if (this.isCommandMode) {
                    this.updateFilteredCommands();
                } else {
                    this.doFileSearch();
                }
            },

            updateFilteredCommands: function() {
                var q = this.commandQuery;
                if (!q) {
                    var recentIds = this.recentCommands.map(function(c) { return c.id; });
                    var rest = commands.filter(function(c) { return recentIds.indexOf(c.id) === -1; });
                    this.filteredCommands = this.recentCommands.concat(rest);
                } else {
                    this.filteredCommands = searchCommands(q, commands);
                }
            },

            doFileSearch: function() {
                var self = this;
                var q = this.fileQuery;
                if (!q) {
                    this.fileSearchResults = [];
                    this.fileSearchLoading = false;
                    return;
                }
                if (this.fileSearchDebounceTimer) {
                    clearTimeout(this.fileSearchDebounceTimer);
                }
                this.fileSearchLoading = true;
                this.fileSearchDebounceTimer = setTimeout(function() {
                    self.fileSearchDebounceTimer = null;
                    var searchAPI = window._clawIDESearchAPI;
                    if (!searchAPI) {
                        self.fileSearchLoading = false;
                        return;
                    }
                    fetch(searchAPI + '?q=' + encodeURIComponent(q))
                        .then(function(r) { return r.ok ? r.json() : []; })
                        .then(function(results) {
                            self.fileSearchResults = results || [];
                            self.fileSearchLoading = false;
                            self.selectedIndex = 0;
                        })
                        .catch(function() {
                            self.fileSearchResults = [];
                            self.fileSearchLoading = false;
                        });
                }, 150);
            },

            onKeydown: function(e) {
                var listLen = this.currentListLength;

                if (e.key === 'ArrowDown') {
                    e.preventDefault();
                    if (this.selectedIndex < listLen - 1) this.selectedIndex++;
                    this.scrollSelectedIntoView();
                } else if (e.key === 'ArrowUp') {
                    e.preventDefault();
                    if (this.selectedIndex > 0) this.selectedIndex--;
                    this.scrollSelectedIntoView();
                } else if (e.key === 'Enter') {
                    e.preventDefault();
                    this.executeSelected();
                } else if (e.key === 'Escape') {
                    e.preventDefault();
                    this.close();
                }
            },

            executeSelected: function() {
                if (this.isCommandMode) {
                    if (this.filteredCommands.length > 0 && this.selectedIndex < this.filteredCommands.length) {
                        this.executeCommand(this.filteredCommands[this.selectedIndex]);
                    }
                } else if (this.fileQuery) {
                    if (this.fileSearchResults.length > 0 && this.selectedIndex < this.fileSearchResults.length) {
                        this.openFileResult(this.fileSearchResults[this.selectedIndex]);
                    }
                } else {
                    // Recent files
                    if (this.recentFiles.length > 0 && this.selectedIndex < this.recentFiles.length) {
                        this.openFileResult(this.recentFiles[this.selectedIndex]);
                    }
                }
            },

            openFileResult: function(result) {
                if (!result) return;
                // Allow opening directories in the folder tree (navigate to them)
                if (result.is_dir) return;
                var filePath = result.path;
                this.close();
                setTimeout(function() {
                    if (typeof window.featureLoadFile === 'function') {
                        window.featureLoadFile(filePath);
                    } else if (typeof window.ClawIDEEditor !== 'undefined') {
                        var editorRoot = document.getElementById('editor-pane-root');
                        var pid = editorRoot ? editorRoot.dataset.projectId : '';
                        window.ClawIDEEditor.loadFile(pid, filePath);
                    }
                }, 50);
            },

            scrollSelectedIntoView: function() {
                var idx = this.selectedIndex;
                this.$nextTick(function() {
                    var item = document.querySelector('[data-palette-index="' + idx + '"]');
                    if (item) {
                        item.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
                    }
                });
            },

            executeCommand: function(cmd) {
                this.close();
                setTimeout(function() {
                    runCommandById(cmd.id);
                }, 50);
            },

            isRecentCommand: function(cmd) {
                return this.recentCommands.some(function(c) { return c.id === cmd.id; });
            },

            // Template helpers
            fileIcon: function(filename, isDir) {
                return getFileIcon(filename, isDir);
            },

            fileDirPath: function(fullPath, name) {
                if (!fullPath || !name) return '';
                var dir = fullPath.substring(0, fullPath.length - name.length);
                // Remove trailing slash
                if (dir.length > 1 && dir.charAt(dir.length - 1) === '/') {
                    dir = dir.substring(0, dir.length - 1);
                }
                return dir || '';
            },

            highlightMatch: function(text, query) {
                return fuzzyHighlight(text, query);
            },

            formatSize: function(bytes) {
                return formatFileSize(bytes);
            },
        };
    };

    // --- Keyboard Shortcuts (global) ---
    document.addEventListener('keydown', function(e) {
        var isCmdK = (e.metaKey || e.ctrlKey) && e.key === 'k';
        var isCmdShiftP = (e.metaKey || e.ctrlKey) && e.shiftKey && e.key === 'P';
        var isCmdP = (e.metaKey || e.ctrlKey) && !e.shiftKey && e.key === 'p';

        if (isCmdK || isCmdShiftP) {
            e.preventDefault();
            if (typeof window.ClawIDEPalette !== 'undefined') {
                window.ClawIDEPalette.openCommandMode();
            }
        }

        if (isCmdP) {
            e.preventDefault();
            if (typeof window.ClawIDEPalette !== 'undefined') {
                window.ClawIDEPalette.open();
            }
        }
    });

    // --- Public API ---
    window.ClawIDEPalette = {
        toggle: function() {
            var data = _findPaletteData();
            if (!data) return;
            if (data.open) {
                data.close();
            } else {
                data.openPalette();
            }
        },
        open: function() {
            var data = _findPaletteData();
            if (data) data.openPalette();
        },
        close: function() {
            var data = _findPaletteData();
            if (data) data.close();
        },
        openCommandMode: function() {
            var data = _findPaletteData();
            if (data) data.openCommandMode();
        },
        openFileSearch: function() {
            var data = _findPaletteData();
            if (data) data.openPalette();
        },
        executeCommand: runCommandById,
        getCommands: function() { return commands.slice(); },
        addRecentFile: addToRecentFiles,
    };

    function _findPaletteData() {
        var els = document.querySelectorAll('[x-data]');
        for (var i = 0; i < els.length; i++) {
            var el = els[i];
            if (el._x_dataStack && el._x_dataStack[0] && typeof el._x_dataStack[0].openPalette === 'function') {
                return el._x_dataStack[0];
            }
        }
        return null;
    }
})();
