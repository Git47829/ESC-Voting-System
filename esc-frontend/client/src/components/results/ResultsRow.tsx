import type { VoteResult } from "../../types";
import { flagUrl } from "../../utils/flagUrl";

export const ResultsRow = ({
  item,
  leaderTotal
}: {
  item: VoteResult;
  leaderTotal: number;
}) => {
  const rank = item.rank ?? 0;
  const isLeader = rank === 1;
  const gapToLeader = Math.max(leaderTotal - item.totalPts, 0);

  return (
    <article
      className={`results-row-card grid grid-cols-[4.5rem_minmax(0,1fr)_7rem] items-center gap-4 border-b border-black/8 px-5 py-4 transition-colors last:border-b-0 sm:grid-cols-[4.5rem_minmax(0,1fr)_9rem_9rem] ${
        isLeader ? "bg-esc-pink/[0.045]" : "bg-transparent hover:bg-black/[0.018]"
      }`}
    >
      <div className="flex items-center gap-3">
        <span
          className={`results-row-rank inline-flex h-11 w-11 items-center justify-center rounded-[1rem] text-sm font-semibold ${
            isLeader ? "bg-esc-black text-white" : "bg-black/5 text-esc-black"
          }`}
        >
          #{rank}
        </span>
      </div>

      <div className="flex min-w-0 items-center gap-4">
        <div className="results-row-flag rounded-[1rem] border border-black/8 bg-white/86 p-1.5 shadow-[0_8px_20px_rgba(17,17,17,0.04)]">
          <img
            src={flagUrl(item.countryId)}
            alt={item.country}
            className="h-9 w-12 rounded-[0.8rem] object-cover"
          />
        </div>

        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <p className="truncate text-base font-semibold text-esc-black">{item.country}</p>
            {isLeader ? (
              <span className="inline-flex items-center rounded-full bg-esc-pink px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-white">
                Leader
              </span>
            ) : null}
          </div>
          <p className="truncate text-sm text-esc-black-soft/75">{item.name}</p>
          <p className="mt-1 text-[11px] uppercase tracking-[0.16em] text-esc-muted">
            {isLeader ? "Currently in first place" : `${gapToLeader} pts behind first`}
          </p>
        </div>
      </div>

      <div className="hidden text-right sm:block">
        <p className="text-[11px] uppercase tracking-[0.16em] text-esc-muted">Public / Jury</p>
        <p className="mt-1 text-sm font-medium text-esc-black">
          {item.escPublicPts} / {item.juryPts}
        </p>
      </div>

      <div className="text-right">
        <p className="text-[11px] uppercase tracking-[0.16em] text-esc-muted">Total</p>
        <p className={`results-row-total mt-1 text-2xl font-semibold tracking-[-0.04em] ${isLeader ? "text-esc-pink" : "text-esc-black"}`}>
          {item.totalPts}
        </p>
      </div>
    </article>
  );
};
