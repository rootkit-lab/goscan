import { ChevronDown, ChevronRight } from "lucide-react";
import { useState, type ReactNode } from "react";

export function CollapsibleSection({
  title,
  defaultOpen = true,
  children
}: {
  title: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <div className="border-b border-vscode-border">
      <button
        type="button"
        className="flex w-full items-center gap-1 px-2 py-1.5 text-left text-vscode-fg hover:bg-vscode-hover"
        onClick={() => setOpen(!open)}
      >
        {open ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />}
        <span className="text-[11px] font-semibold uppercase tracking-wide">{title}</span>
      </button>
      {open && <div className="px-2 pb-2">{children}</div>}
    </div>
  );
}
