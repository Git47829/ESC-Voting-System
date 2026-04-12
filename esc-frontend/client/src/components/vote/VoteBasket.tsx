export const VoteBasket = ({
  total,
  onSubmit
}: {
  total: number;
  onSubmit: () => void;
}) => {
  return (
    <div className="fixed bottom-0 left-0 right-0 z-40 border-t border-esc-border bg-white/90 p-3 backdrop-blur-sm">
      <div className="mx-auto flex max-w-7xl items-center justify-between rounded-xl border border-esc-border bg-esc-surface px-4 py-3 shadow-[0_8px_28px_rgba(0,0,0,0.06)]">
        <span className="text-sm text-esc-muted">
          Selected points:{" "}
          <span className="font-semibold text-esc-black">{total}</span>
        </span>
        <button
          className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim"
          onClick={onSubmit}
        >
          Submit Votes
        </button>
      </div>
    </div>
  );
};