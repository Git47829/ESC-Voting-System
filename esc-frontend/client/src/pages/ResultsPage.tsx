import { ResultsRow } from "../components/results/ResultsRow";
import { useResultsPoll } from "../hooks/useResultsPoll";

export const ResultsPage = () => {
  const { results, countdown, paused, setPaused, error } = useResultsPoll();

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <div className="flex flex-wrap items-end justify-between gap-4 border-b border-esc-border pb-5">
          <div>
            <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Live ranking</p>
            <h1 className="mt-2 text-4xl font-bold text-esc-black">Live Results</h1>
          </div>
          <div className="flex items-center gap-3 text-sm text-esc-muted">
            <span>{paused ? "Paused" : `Refresh in ${countdown}s`}</span>
            <button
              className="rounded-xl border border-esc-border px-3 py-1.5 transition-colors hover:border-esc-pink hover:text-esc-pink"
              onClick={() => setPaused(!paused)}
            >
              {paused ? "Resume" : "Pause"}
            </button>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-2">
          <span className="inline-flex items-center rounded-full border border-esc-border bg-esc-surface2 px-3 py-1 text-xs uppercase tracking-[0.12em] text-esc-muted">
            Top ranked countries
          </span>
          <span className="inline-flex items-center rounded-full border border-esc-border bg-esc-surface2 px-3 py-1 text-xs uppercase tracking-[0.12em] text-esc-muted">
            Jury + public points
          </span>
        </div>
      </div>

      <div className="overflow-hidden rounded-[1.75rem] border border-esc-border bg-white/92 shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
        <div className="grid grid-cols-[3.5rem_1fr_6rem_6rem_6rem] border-b border-esc-border bg-esc-surface2 px-3 py-3 text-xs font-semibold uppercase tracking-[0.12em] text-esc-muted">
          <span>Rank</span>
          <span>Entry</span>
          <span>Public</span>
          <span>Jury</span>
          <span>Total</span>
        </div>
        <div>
          {error ? (
            <div className="px-4 py-8 text-center text-sm text-red-600">{error}</div>
          ) : results.length === 0 ? (
            <div className="px-4 py-8 text-center text-sm text-esc-muted">
              No ranking data available yet.
            </div>
          ) : (
            results.map((item) => (
              <ResultsRow key={item.id} item={item} />
            ))
          )}
        </div>
      </div>
    </section>
  );
};
