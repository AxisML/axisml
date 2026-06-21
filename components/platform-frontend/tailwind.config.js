/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  // Dark mode follows the <html data-theme="dark"> attribute that app/store.tsx
  // already toggles (and AntD's darkAlgorithm reads). Keep them in sync.
  darkMode: ["selector", '[data-theme="dark"]'],
  // Map Tailwind's color/spacing/radius scales onto the AntD design tokens so a
  // Tailwind utility (e.g. text-fg, bg-surface, border-default) renders the same
  // value AntD components use. Tokens live in src/styles/tokens.css.
  theme: {
    extend: {
      colors: {
        bg: "var(--bg)",
        surface: "var(--surface)",
        "surface-warm": "var(--surface-warm)",
        fg: "var(--fg)",
        "fg-2": "var(--fg-2)",
        muted: "var(--muted)",
        accent: "var(--accent)",
        "accent-on": "var(--accent-on)",
        success: "var(--success)",
        warn: "var(--warn)",
        danger: "var(--danger)",
        info: "var(--info)",
        "border-default": "var(--border)",
        "border-soft": "var(--border-soft)",
      },
      borderColor: {
        DEFAULT: "var(--border)",
      },
      borderRadius: {
        sm: "var(--radius-sm)",
        md: "var(--radius-md)",
        lg: "var(--radius-lg)",
      },
      fontFamily: {
        mono: ["var(--font-mono)"],
      },
    },
  },
  plugins: [],
};
