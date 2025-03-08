/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts,css}'],
  plugins: [
    require('@tailwindcss/forms')
  ]
}
