import type { ReactNode } from "react";

type Props = {
  sidebar: ReactNode;
  editor: ReactNode;
  actions: ReactNode;
  terminal: ReactNode;
  statusBar: ReactNode;
  sidebarCollapsed?: boolean;
};

export function WorkbenchLayout({ sidebar, editor, actions, terminal, statusBar, sidebarCollapsed }: Props) {
  return (
    <div className="flex h-screen flex-col bg-vscode-bg text-vscode-fg">
      <div className="flex min-h-0 flex-1">
        {!sidebarCollapsed && (
          <aside className="flex w-[260px] shrink-0 flex-col border-r border-vscode-border bg-vscode-sidebar">{sidebar}</aside>
        )}
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex min-h-0 flex-1">
            <main className="min-w-0 flex-1 bg-vscode-bg">{editor}</main>
            <aside className="flex w-[280px] shrink-0 flex-col border-l border-vscode-border bg-vscode-sidebar">{actions}</aside>
          </div>
          {terminal}
        </div>
      </div>
      {statusBar}
    </div>
  );
}
