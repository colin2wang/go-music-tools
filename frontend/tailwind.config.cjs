const animate = require('tailwindcss-animate')

module.exports = {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {
      colors: {
        ink: {
          950: '#070B16',
          900: '#0B1020',
          800: '#131A2E',
          700: '#1B2440',
          600: '#263356',
        },
        brand: {
          300: '#22D3EE',
          400: '#8B7CF6',
          500: '#6C5CE7',
          600: '#5A4BD1',
        },
        state: {
          ok: '#22C55E',
          warn: '#F59E0B',
          err: '#EF4444',
          info: '#38BDF8',
        },
      },
      fontFamily: {
        sans: ['"Source Han Sans SC"', '"Noto Sans SC"', '"Microsoft YaHei"', 'system-ui', 'sans-serif'],
        mono: ['"JetBrains Mono"', '"Cascadia Code"', 'Consolas', 'monospace'],
      },
      borderRadius: {
        card: '14px',
      },
      boxShadow: {
        panel: '0 12px 32px -12px rgba(0,0,0,0.65)',
        glow: '0 0 0 1px rgba(139,124,246,0.35), 0 8px 28px -8px rgba(108,92,231,0.55)',
      },
      keyframes: {
        'fade-up': {
          '0%': { opacity: '0', transform: 'translateY(8px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'log-in': {
          '0%': { opacity: '0', transform: 'translateX(-6px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
        'pulse-soft': {
          '0%, 100%': { opacity: '0.55' },
          '50%': { opacity: '1' },
        },
      },
      animation: {
        'fade-up': 'fade-up 180ms ease-out both',
        'log-in': 'log-in 160ms ease-out both',
        'pulse-soft': 'pulse-soft 2.2s ease-in-out infinite',
      },
    },
  },
  plugins: [animate],
}
