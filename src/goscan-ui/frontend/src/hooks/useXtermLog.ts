import { useCallback, useEffect, useRef, useState } from "react";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

type Options = {
  enabled: boolean;
  autoStick?: boolean;
};

export function useXtermLog(containerRef: React.RefObject<HTMLDivElement | null>, opts: Options) {
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  useEffect(() => {
    if (!opts.enabled || !containerRef.current) return;
    const term = new Terminal({
      convertEol: true,
      fontSize: 12,
      fontFamily: "var(--vscode-font-mono)",
      theme: {
        background: "#1e1e1e",
        foreground: "#cccccc",
        cursor: "#007acc",
        selectionBackground: "#264f78"
      },
      scrollback: 8000
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(containerRef.current);
    fit.fit();
    termRef.current = term;
    fitRef.current = fit;
    const ro = new ResizeObserver(() => fit.fit());
    ro.observe(containerRef.current);
    return () => {
      ro.disconnect();
      term.dispose();
      termRef.current = null;
    };
  }, [opts.enabled, containerRef]);

  const writeln = useCallback(
    (line: string) => {
      const term = termRef.current;
      if (!term) return;
      term.writeln(line.replace(/\n/g, "\r\n"));
      if (opts.autoStick !== false) term.scrollToBottom();
    },
    [opts.autoStick]
  );

  const reset = useCallback(() => {
    termRef.current?.reset();
  }, []);

  return { writeln, reset };
}
