type Props = {
  findingLabel?: string;
  findingsCount: number;
  scanRunning?: boolean;
  scanStats?: string;
  error?: string;
};

export function StatusBar({ findingLabel, findingsCount, scanRunning, scanStats, error }: Props) {
  return (
    <footer className="flex h-[22px] shrink-0 items-center gap-3 border-t border-vscode-border bg-vscode-accent px-2 text-[12px] text-white">
      <span className="font-medium">goscan</span>
      <span className="opacity-90">{findingsCount} findings</span>
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
      {!scanRunning && !error && <span className="ml-auto opacity-75">:dev :9280</span>}
    </footer>
  );
}
