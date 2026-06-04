import type { Config } from 'tailwindcss'

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
        card: 'var(--card)',
        border: 'var(--border)',
        text: 'var(--text)',
        'text-dim': 'var(--text-dim)',
        'text-faint': 'var(--text-faint)',
        ok: 'var(--ok)',
        warn: 'var(--warn)',
        danger: 'var(--danger)',
      },
      fontFamily: {
        display: ['var(--font-display)'],
        body: ['var(--font-body)'],
      },
      borderRadius: {
        sm: 'var(--r-sm)',
        md: 'var(--r-md)',
        lg: 'var(--r-lg)',
        xl: 'var(--r-xl)',
        pill: 'var(--r-pill)',
      },
    },
  },
  plugins: [],
} satisfies Config
