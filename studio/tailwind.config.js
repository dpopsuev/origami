/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        "origami-fire": "#DC143C",
        "origami-water": "#007BA7",
        "origami-earth": "#0047AB",
        "origami-air": "#FFBF00",
        "origami-diamond": "#0F52BA",
        "origami-lightning": "#DC143C",
        "origami-iron": "#48494B",
        "rh-red": "#EE0000",
        "rh-dark": "#151515",
      },
    },
  },
  plugins: [],
};
