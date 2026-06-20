/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        vscode: {
          bg: "var(--vscode-bg)",
          sidebar: "var(--vscode-sidebar)",
          panel: "var(--vscode-panel)",
          input: "var(--vscode-input)",
          border: "var(--vscode-border)",
          fg: "var(--vscode-fg)",
          muted: "var(--vscode-fg-muted)",
          dim: "var(--vscode-fg-dim)",
          accent: "var(--vscode-accent)",
          selection: "var(--vscode-selection)",
          hover: "var(--vscode-hover)",
          error: "var(--vscode-error)",
          warning: "var(--vscode-warning)",
          "tab-inactive": "var(--vscode-tab-inactive)",
          "tab-active": "var(--vscode-tab-active)"
        }
      },
      fontSize: {
        vscode: "var(--vscode-font-size)"
      }
    }
  },
  plugins: []
};
