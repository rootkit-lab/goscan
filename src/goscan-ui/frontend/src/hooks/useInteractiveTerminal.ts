import { useCallback, useEffect, useRef } from "react";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";

type Options = {
  enabled: boolean;
  readOnly?: boolean;
  onData?: (data: string) => void;
  onResize?: (cols: number, rows: number) => void;
};

export function useInteractiveTerminal(containerRef: React.RefObject<HTMLDivElement | null>, opts: Options) {
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const onDataRef = useRef(opts.onData);
  const onResizeRef = useRef(opts.onResize);

  onDataRef.current = opts.onData;
  onResizeRef.current = opts.onResize;

  useEffect(() => {
    if (!opts.enabled || !containerRef.current) return;

    const readOnly = opts.readOnly ?? false;
    const term = new Terminal({
      convertEol: true,
      cursorBlink: !readOnly,
      disableStdin: readOnly,
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

    const notifySize = () => {
      fit.fit();
      onResizeRef.current?.(term.cols, term.rows);
    };

    if (!readOnly) {
      term.onData((data) => onDataRef.current?.(data));
    }

    const ro = new ResizeObserver(() => notifySize());
    ro.observe(containerRef.current);
    notifySize();

    return () => {
      ro.disconnect();
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
    };
  }, [opts.enabled, opts.readOnly, containerRef]);

  const write = useCallback((data: string) => {
    termRef.current?.write(data);
  }, []);

  const reset = useCallback(() => {
    termRef.current?.reset();
    const fit = fitRef.current;
    const term = termRef.current;
    if (fit && term) {
      fit.fit();
      onResizeRef.current?.(term.cols, term.rows);
    }
  }, []);

  const focus = useCallback(() => {
    termRef.current?.focus();
  }, []);

  const fitTerminal = useCallback(() => {
    fitRef.current?.fit();
    const term = termRef.current;
    if (term) onResizeRef.current?.(term.cols, term.rows);
  }, []);

  return { write, reset, focus, fitTerminal };
}
