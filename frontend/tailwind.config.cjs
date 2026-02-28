module.exports = {
  presets: [require("@vercel/examples-ui/tailwind")],
  content: [
    "./index.html",
    "./src/**/*.{js,jsx,ts,tsx}",
    "./node_modules/@vercel/examples-ui/dist/**/*.js",
  ],
  theme: {
    extend: {
      fontFamily: {
        sans: ["Manrope", "ui-sans-serif", "system-ui", "sans-serif"],
        display: ["Sora", "Manrope", "ui-sans-serif", "system-ui", "sans-serif"],
      },
      boxShadow: {
        glow: "0 0 0 1px rgba(88,101,242,.35), 0 12px 32px rgba(17,24,39,.35)",
      },
    },
  },
};
