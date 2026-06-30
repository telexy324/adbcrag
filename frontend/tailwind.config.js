/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        border: '#d7dde4',
        background: '#f6f8fb',
        foreground: '#17202a',
        primary: '#0f766e',
        muted: '#eef2f6',
        warning: '#b45309',
        danger: '#b91c1c',
      },
    },
  },
  plugins: [],
}
