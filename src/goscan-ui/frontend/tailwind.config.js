/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx,css}"],
  theme: {
    extend: {
      colors: {
        gs: {
          bg: "var(--gs-bg)",
          surface: "var(--gs-surface)",
          "surface-raised": "var(--gs-surface-raised)",
          border: "var(--gs-border)",
          fg: "var(--gs-fg)",
          muted: "var(--gs-fg-muted)",
          dim: "var(--gs-fg-dim)",
          accent: "var(--gs-accent)",
          "accent-muted": "var(--gs-accent-muted)",
          selection: "var(--gs-selection)",
          hover: "var(--gs-hover)",
          error: "var(--gs-error)",
          warning: "var(--gs-warning)",
          success: "var(--gs-success)",
          "zone-header": "var(--gs-zone-header)",
          statusbar: "var(--gs-statusbar)",
          "statusbar-fg": "var(--gs-statusbar-fg)",
          "tab-inactive": "var(--gs-tab-inactive)",
          "tab-active": "var(--gs-tab-active)"
        },
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
        vscode: "var(--vscode-font-size)",
        gs: "var(--gs-font-size)"
      }
    }
  },
  plugins: []
};
