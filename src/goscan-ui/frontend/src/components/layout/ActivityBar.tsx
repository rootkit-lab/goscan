import { clsx } from "clsx";
import { Layers, List, Settings } from "lucide-react";
import type { WorkbenchView } from "@/lib/workbenchView";

const ITEMS: { id: WorkbenchView; icon: typeof List; label: string }[] = [
  { id: "findings", icon: List, label: "Lista" },
  { id: "batch", icon: Layers, label: "Batch" },
  { id: "settings", icon: Settings, label: "Settings" }
];

type Props = {
  view: WorkbenchView;
  onViewChange: (view: WorkbenchView) => void;
  batchActive?: boolean;
};

export function ActivityBar({ view, onViewChange, batchActive }: Props) {
  return (
    <nav
      className="flex w-[52px] shrink-0 flex-col items-center gap-0.5 border-r border-gs-border bg-gs-surface py-3"
      aria-label="Navegação principal"
    >
      {ITEMS.map(({ id, icon: Icon, label }) => {
        const active = view === id;
        const pulse = id === "batch" && batchActive && !active;
        return (
          <button
            key={id}
            type="button"
            title={label}
            onClick={() => onViewChange(id)}
            className={clsx(
              "relative mb-1 flex h-11 w-11 items-center justify-center rounded-lg transition-colors",
              active
                ? "bg-gs-accent-muted text-gs-accent"
                : "text-gs-fg-muted hover:bg-gs-hover hover:text-gs-fg",
              pulse && "animate-pulse"
            )}
          >
            <Icon className="h-5 w-5" strokeWidth={active ? 2.25 : 1.75} />
            {active && (
              <span className="absolute left-0 top-1/2 h-6 w-0.5 -translate-y-1/2 rounded-r bg-gs-accent" />
            )}
          </button>
        );
      })}
    </nav>
  );
}
