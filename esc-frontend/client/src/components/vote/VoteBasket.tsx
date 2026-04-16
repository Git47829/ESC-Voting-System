export const VoteBasket = ({
  total,
  onSubmit,
  disabled = false
}: {
  total: number;
  onSubmit: () => void;
  disabled?: boolean;
}) => {
  return (
    <div className="sticky bottom-4 z-30 w-full sm:bottom-5">
      <div className="flex items-center justify-between rounded-xl border border-esc-pink/12 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(255,245,250,0.94))] px-4 py-3 shadow-[0_8px_28px_rgba(255,4,144,0.06)] backdrop-blur-md">
        <span className="text-sm text-esc-muted">
          Selected points:{" "}
          <span className="font-semibold text-esc-black">{total}</span>
        </span>
        <button
          className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim disabled:cursor-not-allowed disabled:opacity-50"
          onClick={onSubmit}
          disabled={disabled}
        >
          Submit Votes
        </button>
      </div>
    </div>
  );
};