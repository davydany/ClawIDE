/** @type {import('tailwindcss').Config} */

// Helper: wraps a CSS variable in color-mix so Tailwind opacity modifiers work.
// When Tailwind generates e.g. bg-surface-base/50, it calls this function with
// the alpha value and we produce: color-mix(in srgb, var(--surface-base) 50%, transparent)
function withAlpha(varName) {
  return ({ opacityValue }) => {
    if (opacityValue !== undefined) {
      // opacityValue can be a numeric string ("0.5") for /50 modifiers,
      // or a CSS var reference ("var(--tw-bg-opacity)") for base classes.
      // Only use color-mix for actual fractional numeric values.
      var num = parseFloat(opacityValue);
      if (!isNaN(num) && num < 1) {
        return `color-mix(in srgb, var(${varName}) ${Math.round(num * 100)}%, transparent)`;
      }
    }
    return `var(${varName})`;
  };
}

module.exports = {
  content: [
    './web/templates/**/*.html',
    './web/static/js/**/*.js',
  ],
  theme: {
    extend: {
      fontFamily: {
        mono: ['JetBrains Mono', 'ui-monospace', 'SFMono-Regular', 'monospace'],
      },
      colors: {
        surface: {
          deepest: withAlpha('--surface-deepest'),
          base: withAlpha('--surface-base'),
          raised: withAlpha('--surface-raised'),
          overlay: withAlpha('--surface-overlay'),
          hover: 'var(--surface-hover)',
        },
        'th-border': {
          DEFAULT: withAlpha('--border-default'),
          strong: withAlpha('--border-strong'),
          muted: withAlpha('--border-muted'),
        },
        'th-text': {
          primary: withAlpha('--text-primary'),
          secondary: withAlpha('--text-secondary'),
          tertiary: withAlpha('--text-tertiary'),
          muted: withAlpha('--text-muted'),
          faint: withAlpha('--text-faint'),
          ghost: withAlpha('--text-ghost'),
        },
        accent: {
          DEFAULT: withAlpha('--accent'),
          hover: withAlpha('--accent-hover'),
          text: withAlpha('--accent-text'),
          border: withAlpha('--accent-border'),
          muted: withAlpha('--accent-muted'),
          glow: 'var(--accent-glow)',
        },
      },
    },
  },
  plugins: [],
}
