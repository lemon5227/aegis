/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        "warm-bg": "#F4EFE6",
        "warm-sidebar": "#EAE2D6",
        "warm-border": "#D6CEC3",
        "warm-text-primary": "#3E2C23",
        "warm-text-secondary": "#6B5D55",
        "warm-accent": "#D97706",
        "warm-accent-hover": "#B45309",
        "warm-card": "#FAF9F6",
        "background-dark": "#0f1115",
        "surface-dark": "#181b21",
        "surface-lighter": "#22262e",
        "border-dark": "#2d323b",
      },
      fontFamily: {
        "display": ["Inter", "sans-serif"],
      },
    },
  },
  plugins: [],
}
