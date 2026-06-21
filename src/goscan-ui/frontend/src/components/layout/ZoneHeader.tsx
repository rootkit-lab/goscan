type Props = {
  children: React.ReactNode;
  className?: string;
};

export function ZoneHeader({ children, className = "" }: Props) {
  return (
    <div
      className={`border-b border-gs-border px-3 py-2 text-[11px] font-semibold uppercase tracking-wide text-gs-zone-header ${className}`}
    >
      {children}
    </div>
  );
}
