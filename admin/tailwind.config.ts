import type { Config } from 'tailwindcss'

// Brand tokens are duplicated from /web (see CONVENTIONS.md → "Admin app"):
// the pragmatic, no-extra-infrastructure choice. They map to the same CSS
// variables defined in src/styles/base.css so utilities resolve to themed
// tokens. Admin-specific fulfillment accents are added on top.
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: ['selector', '[data-theme="dark"]'],
  theme: {
    extend: {
      colors: {
        brand1: 'var(--brand-1)',
        brand2: 'var(--brand-2)',
        accent: 'var(--accent)',
        agent: 'var(--agent)',
        bg: 'var(--bg)',
        surface: 'var(--surface)',
        'surface-2': 'var(--surface-2)',
        'surface-3': 'var(--surface-3)',
        card: 'var(--card)',
        border: 'var(--border)',
        text: 'var(--text)',
        'text-dim': 'var(--text-dim)',
        'text-faint': 'var(--text-faint)',
        ok: 'var(--ok)',
        warn: 'var(--warn)',
        danger: 'var(--danger)',
        'ff-code': 'var(--ff-code)',
        'ff-credit': 'var(--ff-credit)',
        'ff-transfer': 'var(--ff-transfer)',
      },
      fontFamily: {
        display: ['var(--font-display)'],
        body: ['var(--font-body)'],
      },
      borderRadius: {
        sm: 'var(--ar-sm)',
        md: 'var(--ar-md)',
        lg: 'var(--ar-lg)',
        pill: 'var(--r-pill)',
      },
    },
  },
  plugins: [],
} satisfies Config
