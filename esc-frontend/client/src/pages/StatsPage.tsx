import { useEffect, useState } from "react";

import { api } from "../api/client";

interface StatsData {
  totalPublic: number;
  totalJury: number;
  byCountry: Array<{ countryId: string; country: string; total: number }>;
}

export const StatsPage = () => {
  const [stats, setStats] = useState<StatsData | null>(null);

  useEffect(() => {
    void api.getStats().then(setStats);
  }, []);

  if (!stats) return <p>Loading stats...</p>;

  return (
    <section className="space-y-4">
      <h1 className="text-3xl font-bold">Statistics</h1>
      <div className="grid gap-3 md:grid-cols-2">
        <article className="border border-esc-muted p-4">
          <p>Total Public Votes: {stats.totalPublic}</p>
        </article>
        <article className="border border-esc-muted p-4">
          <p>Total Jury Votes: {stats.totalJury}</p>
        </article>
      </div>
      <div className="border border-esc-muted p-4">
        {stats.byCountry.map((entry) => (
          <div key={entry.countryId} className="mb-2 flex justify-between border-b border-esc-muted/30 py-1">
            <span>{entry.country}</span>
            <span>{entry.total}</span>
          </div>
        ))}
      </div>
    </section>
  );
};

