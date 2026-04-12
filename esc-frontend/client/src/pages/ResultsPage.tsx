import { ResultsRow } from "../components/results/ResultsRow";
import { useResultsPoll } from "../hooks/useResultsPoll";

export const ResultsPage = () => {
  const { results, countdown, paused, setPaused } = useResultsPoll();

  return (
    <section className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold">Live Results</h1>
        <div className="flex items-center gap-3 text-sm">
          <span>Refresh in {countdown}s</span>
          <button className="border border-esc-muted px-2 py-1" onClick={() => setPaused(!paused)}>
            {paused ? "Resume" : "Pause"}
          </button>
        </div>
      </div>
      <div className="border border-esc-muted">
        {results.map((item) => (
          <ResultsRow key={item.id} item={item} />
        ))}
      </div>
    </section>
  );
};

