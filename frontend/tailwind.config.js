/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // ブランドカラー（濃い目のパープル）。
        // 背景・面はニュートラルに保ち、violet はアクセントにだけ使う。
        brand: {
          50: '#f5f3ff',
          200: '#ddd6fe',
          300: '#c4b5fd',
          400: '#a78bfa',
          500: '#8b5cf6',
          600: '#7c3aed',
          700: '#6d28d9',
          900: '#4c1d95',
        },
        surface: {
          base: '#101012',
          raised: '#18181b',
          hover: '#212124',
          border: '#2b2b2f',
        },
      },
      fontFamily: {
        sans: ['Inter', 'Noto Sans JP', 'system-ui', 'sans-serif'],
        ko: ['Noto Sans KR', 'Noto Sans JP', 'sans-serif'],
      },
      spacing: {
        topbar: '56px',
        sidebar: '236px',
        'sidebar-collapsed': '60px',
      },
      keyframes: {
        flicker: {
          '0%, 18%, 22%, 25%, 53%, 57%, 100%': { opacity: '1' },
          '20%, 24%, 55%': { opacity: '0.35' },
        },
        'toast-in': {
          from: { opacity: '0', transform: 'translateY(8px) scale(0.98)' },
          to: { opacity: '1', transform: 'translateY(0) scale(1)' },
        },
      },
      animation: {
        flicker: 'flicker 4s linear infinite',
        'toast-in': 'toast-in 180ms ease-out',
      },
    },
  },
  plugins: [require('daisyui')],
  daisyui: {
    themes: [
      {
        'hangulme-dark': {
          primary: '#a78bfa',
          'primary-content': '#1c1030',
          secondary: '#5eead4',
          accent: '#c4b5fd',
          neutral: '#18181b',
          'base-100': '#101012',
          'base-200': '#18181b',
          'base-300': '#2b2b2f',
          'base-content': '#ececee',
          info: '#60a5fa',
          success: '#34d399',
          warning: '#fbbf24',
          error: '#fb7185',
        },
      },
      {
        'hangulme-light': {
          primary: '#6d28d9',
          'primary-content': '#ffffff',
          secondary: '#0f766e',
          accent: '#7c3aed',
          neutral: '#f0f0ef',
          'base-100': '#ffffff',
          'base-200': '#f7f7f6',
          'base-300': '#e2e1de',
          'base-content': '#1a1a1c',
          info: '#2563eb',
          success: '#059669',
          warning: '#d97706',
          error: '#e11d48',
        },
      },
    ],
    darkTheme: 'hangulme-dark',
    logs: false,
  },
};
