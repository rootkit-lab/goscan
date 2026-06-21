import type { ReactNode } from "react";

type Props = {
  title?: string;
  children: ReactNode;
  centered?: boolean;
  className?: string;
};

export function ViewShell({ title, children, centered, className = "" }: Props) {
  return (
    <div className={`flex h-full min-h-0 w-full flex-1 flex-col overflow-auto bg-gs-bg ${className}`}>
      {title && (
        <div className="border-b border-gs-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-gs-zone-header">
          {title}
        </div>
      )}
      <div className={centered ? "flex flex-1 items-start justify-center p-6" : "flex-1 p-3"}>{children}</div>
    </div>
  );
}
