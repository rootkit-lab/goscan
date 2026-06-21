import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";

type Props = {
  title: string;
  defaultHeight?: number;
  minHeight?: number;
  maxHeight?: number;
  headerRight?: ReactNode;
  children: ReactNode;
};

export function ResizablePanel({ title, defaultHeight = 180, minHeight = 80, maxHeight = 480, headerRight, children }: Props) {
  const [height, setHeight] = useState(defaultHeight);
  const dragging = useRef(false);

  const onMouseMove = useCallback(
    (e: MouseEvent) => {
      if (!dragging.current) return;
      const next = Math.min(maxHeight, Math.max(minHeight, window.innerHeight - e.clientY - 22));
      setHeight(next);
    },
    [minHeight, maxHeight]
  );

  const onMouseUp = useCallback(() => {
    dragging.current = false;
  }, []);

  useEffect(() => {
    window.addEventListener("mousemove", onMouseMove);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("mousemove", onMouseMove);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, [onMouseMove, onMouseUp]);

  return (
    <div className="flex shrink-0 flex-col border-t border-gs-border bg-gs-surface" style={{ height }}>
      <div
        className="flex h-[22px] shrink-0 cursor-row-resize items-center justify-between border-b border-gs-border bg-gs-surface px-2 text-[11px] uppercase tracking-wide text-gs-muted"
        onMouseDown={() => {
          dragging.current = true;
        }}
      >
        <span>{title}</span>
        {headerRight}
      </div>
      <div className="min-h-0 flex-1 overflow-hidden p-1">{children}</div>
    </div>
  );
}
