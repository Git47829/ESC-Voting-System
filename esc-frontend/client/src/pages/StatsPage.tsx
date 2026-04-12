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

  if (!stats) {
    return (
      <section className="rounded-[1.75rem] border border-esc-border bg-white/90 p-8 shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
        <p className="text-sm text-esc-muted">Loading stats...</p>
      </section>
    );
  }

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Analytics</p>
        <h1 className="mt-2 text-4xl font-bold text-esc-black">Statistics</h1>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <article className="rounded-[1.5rem] border border-esc-border bg-white/92 p-6 shadow-[0_14px_30px_rgba(0,0,0,0.04)]">
          <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Total public votes</p>
          <p className="mt-3 text-4xl font-bold text-esc-black">{stats.totalPublic}</p>
        </article>
        <article className="rounded-[1.5rem] border border-esc-border bg-white/92 p-6 shadow-[0_14px_30px_rgba(0,0,0,0.04)]">
          <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Total jury votes</p>
          <p className="mt-3 text-4xl font-bold text-esc-black">{stats.totalJury}</p>
        </article>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <div className="mb-4 flex items-center justify-between border-b border-esc-border pb-3">
          <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">By country</p>
          <span className="text-xs uppercase tracking-[0.12em] text-esc-muted">Total votes</span>
        </div>
        <div className="space-y-2">
          {stats.byCountry.map((entry) => (
            <div
              key={entry.countryId}
              className="flex items-center justify-between rounded-xl border border-esc-border bg-white px-4 py-2.5"
            >
              <span className="text-sm font-medium text-esc-black">{entry.country}</span>
              <span className="text-sm font-semibold text-esc-pink">{entry.total}</span>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};
