export const BudgetBar = ({ remaining, total }: { remaining: number; total: number }) => {
  const used = Math.max(total - remaining, 0);
  const width = Math.max(0, Math.min(100, (used / total) * 100));

  return (
    <div className="budget-pulse rounded-2xl border border-esc-border bg-esc-surface p-5 shadow-[0_10px_24px_rgba(0,0,0,0.04)]">
      <div className="mb-3 flex items-end justify-between">
        <span className="text-xs uppercase tracking-[0.14em] text-esc-muted">Votes remaining</span>
        <span className="text-base font-semibold text-esc-black">
          {remaining}
          <span className="text-esc-muted">/{total}</span>
        </span>
      </div>
      <div className="h-2.5 overflow-hidden rounded-full bg-esc-surface2">
        <div
          className="progress-shine h-2.5 rounded-full bg-gradient-to-r from-esc-pink-dim to-esc-pink"
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  );
};