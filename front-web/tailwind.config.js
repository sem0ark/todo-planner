/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        navy: '#003448',
        slate: {
          blue: '#91a6be',
          grey: '#afb6cf',
        },
        cloud: '#dee2ef',
        snow: '#f0f0f0',
        error: '#ff3b30',
        success: '#34c759',
        warning: '#ffcc00',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Courier New', 'monospace'],
      },
      borderRadius: {
        outer: '24px',
        inner: '16px',
      },
      transitionDuration: {
        micro: '120ms',
      },
    },
  },
  plugins: [],
}
