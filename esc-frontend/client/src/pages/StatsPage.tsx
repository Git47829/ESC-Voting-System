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
      <section className="-mx-4 sm:-mx-6 lg:-mx-8">
        <div className="relative isolate overflow-hidden px-4 pb-10 pt-8 sm:px-6 sm:pb-12 lg:px-8">
          <div className="pointer-events-none absolute inset-0 vote-section-wash" />
          <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
          <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
          <div className="pointer-events-none absolute left-[-22%] top-[4%] h-[54rem] w-[92rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.4)_0%,_rgba(255,4,144,0.22)_30%,_rgba(255,4,144,0.08)_52%,_rgba(255,4,144,0)_80%)] blur-[190px] opacity-94" />
          <div className="pointer-events-none absolute right-[-20%] top-[16%] h-[48rem] w-[80rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.28)_0%,_rgba(255,4,144,0.15)_34%,_rgba(255,4,144,0.06)_54%,_rgba(255,4,144,0)_82%)] blur-[182px] opacity-86" />

          <div className="relative z-10 mx-auto max-w-7xl">
            <section className="vote-panel rounded-[1.75rem] p-8 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
              <p className="text-sm text-esc-muted">Loading stats...</p>
            </section>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate overflow-hidden px-4 pb-10 pt-8 sm:px-6 sm:pb-12 lg:px-8">
        <div className="pointer-events-none absolute inset-0 vote-section-wash" />
        <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
        <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
        <div className="pointer-events-none absolute left-[-22%] top-[4%] h-[54rem] w-[92rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.4)_0%,_rgba(255,4,144,0.22)_30%,_rgba(255,4,144,0.08)_52%,_rgba(255,4,144,0)_80%)] blur-[190px] opacity-94" />
        <div className="pointer-events-none absolute right-[-20%] top-[16%] h-[48rem] w-[80rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.28)_0%,_rgba(255,4,144,0.15)_34%,_rgba(255,4,144,0.06)_54%,_rgba(255,4,144,0)_82%)] blur-[182px] opacity-86" />

        <div className="relative z-10 mx-auto max-w-7xl space-y-6">
          <div className="vote-panel rounded-[2rem] p-6 shadow-[0_18px_44px_rgba(0,0,0,0.2)] sm:p-8">
            <p className="text-xs uppercase tracking-[0.16em] text-esc-pink/72">Analytics</p>
            <h1 className="mt-2 text-4xl font-bold text-esc-black">Statistics</h1>
          </div>

          <div className="grid gap-4 md:grid-cols-2">
            <article className="vote-panel-soft rounded-[1.5rem] p-6 shadow-[0_14px_30px_rgba(0,0,0,0.18)]">
              <p className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">Total public votes</p>
              <p className="mt-3 text-4xl font-bold text-esc-black">{stats.totalPublic}</p>
            </article>
            <article className="vote-panel-soft rounded-[1.5rem] p-6 shadow-[0_14px_30px_rgba(0,0,0,0.18)]">
              <p className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">Total jury votes</p>
              <p className="mt-3 text-4xl font-bold text-esc-black">{stats.totalJury}</p>
            </article>
          </div>

          <div className="vote-panel rounded-[1.75rem] p-5 shadow-[0_16px_36px_rgba(0,0,0,0.2)] sm:p-6">
            <div className="mb-4 flex items-center justify-between border-b border-esc-pink/12 pb-3">
              <p className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">By country</p>
              <span className="text-xs uppercase tracking-[0.12em] text-esc-pink/72">Total votes</span>
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
        </div>
      </div>
    </section>
  );
};
