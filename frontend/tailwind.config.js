/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        opanel: {
          bg: '#121212',
          panel: '#1E1E1E',
          sidebar: '#1A1A2E',
          border: '#2D2D3D',
          primary: '#00AEEF',
          'primary-hover': '#0096D6',
          success: '#4CAF50',
          warning: '#FF9800',
          danger: '#F44336',
          text: '#E0E0E0',
          'text-muted': '#9E9E9E',
        },
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
