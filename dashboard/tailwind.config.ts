import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        gray: {
          800: '#1f2937',
          900: '#111827',
        },
      },
    },
  },
  plugins: [],
};

export default config;

