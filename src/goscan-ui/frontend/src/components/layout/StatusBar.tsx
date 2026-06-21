import { checkerFilterLabel, type CheckerResultFilter } from "@/lib/checkerFilters";

type Props = {
  findingLabel?: string;
  findingsCount: number;
  unopenedCount?: number;
  unopenedFilter?: boolean;
  checkerFilter?: CheckerResultFilter;
  scanRunning?: boolean;
  scanStats?: string;
  error?: string;
  mode?: string;
  dataDir?: string;
};

export function StatusBar({
  findingLabel,
  findingsCount,
  unopenedCount,
  unopenedFilter,
  checkerFilter,
  scanRunning,
  scanStats,
  error,
  mode,
  dataDir
}: Props) {
  const countLabel = checkerFilter
    ? checkerFilterLabel(checkerFilter).toLowerCase()
    : unopenedFilter
      ? "novos"
      : "findings";

  return (
    <footer className="flex h-[22px] shrink-0 items-center gap-3 border-t border-gs-border bg-gs-statusbar px-2 text-[12px] text-gs-statusbar-fg">
      <span className="font-medium">goscan</span>
      <span className="opacity-90">
        {findingsCount} {countLabel}
      </span>
      {!unopenedFilter && !checkerFilter && unopenedCount !== undefined && unopenedCount > 0 && (
        <span className="opacity-90">{unopenedCount} novos</span>
      )}
      {findingLabel && <span className="truncate opacity-90">{findingLabel}</span>}
      {scanRunning && (
        <span className="ml-auto animate-pulse truncate">
          scan {scanStats ?? "…"}
        </span>
      )}
      {!scanRunning && error && (
        <span className="ml-auto truncate text-red-200" title={error}>
          {error}
        </span>
      )}
      {!scanRunning && !error && (
        <span className="ml-auto truncate opacity-75" title={dataDir}>
          {mode ?? "dev"}
          {dataDir ? ` · ${dataDir.length > 36 ? "…" + dataDir.slice(-35) : dataDir}` : ""}
        </span>
      )}
    </footer>
  );
}
