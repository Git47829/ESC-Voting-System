import { useEffect, useMemo, useState } from "react";

import { api } from "../api/client";
import { flagUrl } from "../utils/flagUrl";

interface StatsData {
  totalPublic: number;
  totalJury: number;
  byCountry: Array<{ countryId: string; country: string; total: number }>;
}

const formatNumber = (value: number) => new Intl.NumberFormat("en-US").format(value);

export const StatsPage = () => {
  const [stats, setStats] = useState<StatsData | null>(null);

  useEffect(() => {
    void api.getStats().then(setStats);
  }, []);

  const summary = useMemo(() => {
    if (!stats) return null;

    const rankedCountries = [...stats.byCountry].sort(
      (a, b) => b.total - a.total || a.country.localeCompare(b.country)
    );
    const combined = stats.totalPublic + stats.totalJury;
    const leader = rankedCountries[0] ?? null;
    const countryCount = rankedCountries.length;
    const maxCountryTotal = Math.max(...rankedCountries.map((entry) => entry.total), 1);

    return {
      combined,
      leader,
      countryCount,
      maxCountryTotal,
      rankedCountries
    };
  }, [stats]);

  if (!stats || !summary) {
    return (
      <section className="-mx-4 sm:-mx-6 lg:-mx-8">
        <div className="relative overflow-hidden bg-[linear-gradient(180deg,#faf7f9_0%,#ffffff_100%)]">
          <div className="pointer-events-none absolute inset-0 opacity-[0.34] [background-image:linear-gradient(rgba(17,17,17,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(17,17,17,0.04)_1px,transparent_1px)] [background-size:8rem_8rem]" />
          <div className="relative mx-auto max-w-[88rem] px-6 pb-16 pt-10 sm:px-8 lg:px-10 xl:px-12 lg:pb-20">
            <div className="border-b border-black/10 pb-4">
              <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Statistics</p>
              <h1 className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black">
                Voting numbers
              </h1>
            </div>
            <div className="py-16 text-sm text-esc-black-soft/70">Loading statistics...</div>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative overflow-hidden bg-[linear-gradient(180deg,#faf7f9_0%,#ffffff_100%)]">
        <div className="pointer-events-none absolute inset-0 opacity-[0.34] [background-image:linear-gradient(rgba(17,17,17,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(17,17,17,0.04)_1px,transparent_1px)] [background-size:8rem_8rem]" />
        <div className="pointer-events-none absolute left-[-5rem] top-10 h-44 w-44 rounded-full bg-esc-pink/8 blur-3xl" />
        <div className="pointer-events-none absolute right-[-8rem] top-0 h-56 w-56 rounded-full bg-esc-pink/6 blur-3xl" />

        <div className="relative mx-auto max-w-[88rem] px-6 pb-16 pt-10 sm:px-8 lg:px-10 xl:px-12 lg:pb-24">
          <header className="border-b border-black/10 pb-6">
            <div className="flex flex-wrap items-end justify-between gap-4">
              <div>
                <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Statistics</p>
                <h1 className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black sm:text-5xl">
                  Voting numbers
                </h1>
              </div>
              <p className="max-w-2xl text-sm leading-7 text-esc-black-soft/72">
                A compact overview of the current vote totals, the overall combined score and the
                country-by-country breakdown.
              </p>
            </div>
          </header>

          <section className="mt-10 border-y border-black/10 bg-white/55">
            <div className="grid gap-0 md:grid-cols-4">
              <div className="border-b border-black/10 px-0 py-5 md:border-b-0 md:border-r md:border-black/10 md:pr-6">
                <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Combined total</p>
                <p className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black">
                  {formatNumber(summary.combined)}
                </p>
              </div>
              <div className="border-b border-black/10 px-0 py-5 md:border-b-0 md:border-r md:border-black/10 md:px-6">
                <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Public votes</p>
                <p className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black">
                  {formatNumber(stats.totalPublic)}
                </p>
              </div>
              <div className="border-b border-black/10 px-0 py-5 md:border-b-0 md:border-r md:border-black/10 md:px-6">
                <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Jury votes</p>
                <p className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black">
                  {formatNumber(stats.totalJury)}
                </p>
              </div>
              <div className="px-0 py-5 md:pl-6">
                <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Leading country</p>
                <div className="mt-3 flex items-center gap-3">
                  {summary.leader ? (
                    <img
                      src={flagUrl(summary.leader.countryId)}
                      alt={summary.leader.country}
                      className="h-8 w-11 rounded-[0.7rem] border border-black/8 object-cover"
                    />
                  ) : null}
                  <p className="text-2xl font-semibold tracking-[-0.04em] text-esc-black">
                    {summary.leader?.country ?? "-"}
                  </p>
                </div>
                <p className="mt-2 text-sm text-esc-black-soft/72">
                  {summary.leader ? `${formatNumber(summary.leader.total)} total points` : "No scores yet"}
                </p>
              </div>
            </div>
          </section>

          <section className="mt-12">
            <div className="flex flex-wrap items-end justify-between gap-4 border-b border-black/10 pb-4">
              <div>
                <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">By country</p>
                <h2 className="mt-2 text-3xl font-semibold tracking-[-0.04em] text-esc-black">
                  Current ranking
                </h2>
              </div>
              <p className="max-w-2xl text-sm leading-7 text-esc-black-soft/72">
                {summary.countryCount} countries currently appear in the results table.
              </p>
            </div>

            <div className="mt-4">
              <div className="grid grid-cols-[4rem_minmax(0,1fr)_7rem] items-center gap-4 border-b border-black/10 px-1 py-3 text-[11px] uppercase tracking-[0.18em] text-esc-muted sm:grid-cols-[4rem_minmax(0,1.15fr)_16rem_8rem]">
                <span>Rank</span>
                <span>Country</span>
                <span className="hidden sm:block">Progress</span>
                <span className="text-right">Total</span>
              </div>

              <div>
                {summary.rankedCountries.map((entry, index) => {
                  const width = Math.max(Math.round((entry.total / summary.maxCountryTotal) * 100), 6);
                  const isLeader = index === 0;

                  return (
                    <article
                      key={entry.countryId}
                      className={`grid grid-cols-[4rem_minmax(0,1fr)_7rem] items-center gap-4 border-b border-black/8 px-1 py-4 sm:grid-cols-[4rem_minmax(0,1.15fr)_16rem_8rem] ${
                        isLeader ? "bg-esc-pink/[0.035]" : "bg-transparent"
                      }`}
                    >
                      <span
                        className={`inline-flex h-10 w-10 items-center justify-center rounded-[0.9rem] text-sm font-semibold ${
                          isLeader ? "bg-esc-black text-white" : "bg-black/5 text-esc-black"
                        }`}
                      >
                        #{index + 1}
                      </span>

                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-3">
                          <img
                            src={flagUrl(entry.countryId)}
                            alt={entry.country}
                            className="h-8 w-11 rounded-[0.7rem] border border-black/8 object-cover"
                          />
                          <p className="truncate text-base font-semibold text-esc-black">{entry.country}</p>
                          {isLeader ? (
                            <span className="inline-flex items-center rounded-full bg-esc-pink px-2 py-1 text-[10px] font-semibold uppercase tracking-[0.16em] text-white">
                              Leader
                            </span>
                          ) : null}
                        </div>
                        <p className="mt-1 text-[11px] uppercase tracking-[0.16em] text-esc-muted">
                          {isLeader ? "Highest total so far" : `${formatNumber(summary.leader!.total - entry.total)} points behind first`}
                        </p>
                      </div>

                      <div className="hidden sm:block">
                        <div className="h-2 overflow-hidden rounded-full bg-black/8">
                          <div
                            className={`h-full rounded-full ${isLeader ? "bg-esc-pink" : "bg-esc-black"}`}
                            style={{ width: `${width}%` }}
                          />
                        </div>
                      </div>

                      <div className="text-right">
                        <p className="text-[11px] uppercase tracking-[0.16em] text-esc-muted">Total</p>
                        <p className={`mt-1 text-2xl font-semibold tracking-[-0.04em] ${isLeader ? "text-esc-pink" : "text-esc-black"}`}>
                          {formatNumber(entry.total)}
                        </p>
                      </div>
                    </article>
                  );
                })}
              </div>
            </div>
          </section>
        </div>
      </div>
    </section>
  );
};
