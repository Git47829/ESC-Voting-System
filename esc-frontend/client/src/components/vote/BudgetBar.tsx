export const BudgetBar = ({ remaining, total }: { remaining: number; total: number }) => {
  const safeRemaining = Math.max(0, Math.min(remaining, total));
  const width = total > 0 ? (safeRemaining / total) * 100 : 0;

  return (
    <div className="vote-panel rounded-2xl p-5">
      <div className="mb-3 flex items-end justify-between">
        <span className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">
          Votes remaining
        </span>
        <span className="text-base font-semibold text-esc-black">
          {safeRemaining}
          <span className="text-esc-muted">/{total}</span>
        </span>
      </div>

      <div className="h-2.5 overflow-hidden rounded-full bg-black/8">
        <div
          className="h-2.5 rounded-full bg-esc-pink transition-[width] duration-300 ease-out"
          style={{ width: `${width}%` }}
        />
      </div>
    </div>
  );
};