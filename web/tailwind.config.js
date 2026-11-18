export default {
  presets: [require('maco-ui/tailwind-preset')],
  darkMode: ['class'],
  content: [
    './index.html',
    './src/**/*.{ts,tsx,js,jsx}',
    './node_modules/maco-ui/dist/**/*.js',
  ],
  plugins: [require('tailwindcss-animate')],
}
