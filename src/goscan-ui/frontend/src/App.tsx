import { useCallback, useEffect, useRef, useState } from "react";
import { ActionPanel } from "@/components/actions/ActionPanel";
import { CommandPalette } from "@/components/command/CommandPalette";
import { EditorArea } from "@/components/editor/EditorArea";
import { StatusBar } from "@/components/layout/StatusBar";
import { WorkbenchLayout } from "@/components/layout/WorkbenchLayout";
import { FindingsSidebar } from "@/components/sidebar/FindingsSidebar";
import { BottomPanel, type BottomTab } from "@/components/terminal/BottomPanel";
import { useInteractiveTerminal } from "@/hooks/useInteractiveTerminal";
import { api, Events, type FindingDetailDTO, type ScanProgressDTO, type ScriptDTO } from "@/lib/api";

function useDebounce<T>(value: T, ms: number): T {
  const [v, setV] = useState(value);
  useEffect(() => {
    const t = setTimeout(() => setV(value), ms);
    return () => clearTimeout(t);
  }, [value, ms]);
  return v;
}

export function App() {
  const [query, setQuery] = useState("");
  const debouncedQuery = useDebounce(query, 150);
  const [confidence, setConfidence] = useState("");
  const [findings, setFindings] = useState<Awaited<ReturnType<typeof api.searchFindings>>>([]);
  const [selectedId, setSelectedId] = useState<number | null>(null);
  const [detail, setDetail] = useState<FindingDetailDTO | null>(null);
  const [scripts, setScripts] = useState<ScriptDTO[]>([]);
  const [selectedScript, setSelectedScript] = useState("");
  const [scanProgress, setScanProgress] = useState<ScanProgressDTO | null>(null);
  const [scanOpts, setScanOpts] = useState({ threads: 50, fast: false, rescan: false, timeoutSec: 8 });
  const [error, setError] = useState("");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [bottomTab, setBottomTab] = useState<BottomTab>("output");
  const [outputLines, setOutputLines] = useState<string[]>([]);
  const [terminalActive, setTerminalActive] = useState(false);

  const termRef = useRef<HTMLDivElement>(null);
  const { write, reset, focus, fitTerminal } = useInteractiveTerminal(termRef, {
    enabled: true,
    onData: (data) => void api.terminalInput(data),
    onResize: (cols, rows) => {
      if (bottomTab === "terminal") void api.terminalResize(cols, rows);
    }
  });

  const appendOutput = useCallback((line: string) => {
    setOutputLines((prev) => [...prev, line]);
  }, []);

  const loadFindings = useCallback(async () => {
    try {
      const list = await api.searchFindings(debouncedQuery, confidence, 200);
      setFindings(list);
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }, [debouncedQuery, confidence]);

  useEffect(() => {
    void loadFindings();
  }, [loadFindings]);

  useEffect(() => {
    if (bottomTab === "terminal") {
      const t = setTimeout(() => fitTerminal(), 100);
      return () => clearTimeout(t);
    }
  }, [bottomTab, fitTerminal]);

  useEffect(() => {
    const mq = window.matchMedia("(max-width: 1023px)");
    const apply = () => setSidebarCollapsed(mq.matches);
    apply();
    mq.addEventListener("change", apply);
    return () => mq.removeEventListener("change", apply);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaletteOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const selectFinding = async (id: number) => {
    setSelectedId(id);
    try {
      const d = await api.getFinding(id);
      setDetail(d);
      const compat = await api.compatibleScripts(id);
      setScripts(compat);
      setSelectedScript(compat[0]?.id ?? "");
    } catch (e) {
      setError(String(e));
    }
  };

  useEffect(() => {
    const data = (ev: unknown) => (ev as { data?: unknown }).data ?? ev;

    const offOutput = Events.On("scan:output", (ev) => appendOutput(String(data(ev))));
    const offProgress = Events.On("scan:progress", (ev) => {
      const p = data(ev) as ScanProgressDTO;
      setScanProgress(p);
      if (p.running && p.domainsScanned > 0 && p.domainsScanned % 25 === 0) {
        appendOutput(`… ${p.domainsScanned} domínios · ${p.vulnsFound} vulns`);
      }
    });
    const offFound = Events.On("scan:found", () => void loadFindings());

    const offTermStart = Events.On("terminal:start", (ev) => {
      const p = data(ev) as { scriptId: string; label?: string; python?: string };
      setBottomTab("terminal");
      setTerminalActive(true);
      reset();
      const pyLine = p.python ? `\r\n\x1b[90m${p.python}\x1b[0m` : "";
      write(`\r\n\x1b[1m${p.label ?? p.scriptId}\x1b[0m${pyLine}\r\n`);
      setTimeout(() => {
        focus();
        fitTerminal();
      }, 50);
    });
    const offTermData = Events.On("terminal:data", (ev) => write(String(data(ev))));
    const offTermExit = Events.On("terminal:exit", (ev) => {
      const p = data(ev) as { exitCode: number; scriptId: string };
      write(`\r\n--- exit ${p.scriptId}: code ${p.exitCode} ---\r\n`);
      setTerminalActive(false);
    });

    return () => {
      offOutput();
      offProgress();
      offFound();
      offTermStart();
      offTermData();
      offTermExit();
    };
  }, [appendOutput, loadFindings, write, reset, focus, fitTerminal]);

  const runScript = async () => {
    if (!selectedId || !selectedScript) return;
    const script = scripts.find((s) => s.id === selectedScript);
    try {
      if (script?.interactive) {
        setBottomTab("terminal");
      }
      await api.runScript(selectedScript, selectedId);
    } catch (e) {
      setError(String(e));
    }
  };

  const startScan = async () => {
    setBottomTab("output");
    try {
      await api.startScan(scanOpts);
    } catch (e) {
      setError(String(e));
    }
  };

  const scanStats = scanProgress
    ? `${scanProgress.domainsScanned} dom · ${scanProgress.vulnsFound} vulns`
    : undefined;

  return (
    <>
      <WorkbenchLayout
        sidebarCollapsed={sidebarCollapsed}
        sidebar={
          <FindingsSidebar
            query={query}
            onQueryChange={setQuery}
            confidence={confidence}
            onConfidenceChange={setConfidence}
            findings={findings}
            selectedId={selectedId}
            onSelect={(id) => void selectFinding(id)}
          />
        }
        editor={<EditorArea detail={detail} />}
        actions={
          <ActionPanel
            scripts={scripts}
            selectedScript={selectedScript}
            onScriptChange={setSelectedScript}
            onRunScript={() => void runScript()}
            canRunScript={!!selectedScript && !!selectedId}
            scanOpts={scanOpts}
            onScanOptsChange={setScanOpts}
            onStartScan={() => void startScan()}
            onCancelScan={() => void api.cancelScan()}
            onCancelScript={() => void api.cancelScript()}
            terminalActive={terminalActive}
          />
        }
        terminal={
          <BottomPanel
            tab={bottomTab}
            onTabChange={setBottomTab}
            outputLines={outputLines}
            onClearOutput={() => setOutputLines([])}
            termRef={termRef}
            onClearTerminal={reset}
            terminalActive={terminalActive}
          />
        }
        statusBar={
          <StatusBar
            findingLabel={detail ? `${detail.domain}${detail.path}` : undefined}
            findingsCount={findings.length}
            scanRunning={scanProgress?.running}
            scanStats={scanStats}
            error={error}
          />
        }
      />
      <CommandPalette
        open={paletteOpen}
        onClose={() => setPaletteOpen(false)}
        query={query}
        onQueryChange={setQuery}
        findings={findings}
        onSelectFinding={(id) => void selectFinding(id)}
        onRunScript={() => void runScript()}
        onStartScan={() => void startScan()}
      />
    </>
  );
}
