import { ResultsRow } from "../components/results/ResultsRow";
import { useResultsPoll } from "../hooks/useResultsPoll";
import type { VoteResult } from "../types";
import { flagUrl } from "../utils/flagUrl";

const RESULTS_MARKERS = [
  "Live updates every 10 seconds",
  "Public and jury points combined",
  "Follow the current top three"
];

const StatTile = ({
  label,
  value,
  detail,
  inverted = false
}: {
  label: string;
  value: number | string;
  detail: string;
  inverted?: boolean;
}) => {
  return (
    <article
      className={`rounded-[1.8rem] border px-5 py-5 shadow-[0_18px_40px_rgba(17,17,17,0.06)] ${
        inverted
          ? "border-esc-pink/25 bg-[linear-gradient(180deg,rgba(20,20,20,0.96),rgba(37,25,32,0.94))] text-white"
          : "border-black/[0.08] bg-white/88 text-esc-black"
      }`}
    >
      <p
        className={`text-[11px] uppercase tracking-[0.2em] ${
          inverted ? "text-white/55" : "text-esc-muted"
        }`}
      >
        {label}
      </p>
      <p className="mt-3 text-3xl font-semibold tracking-[-0.05em]">{value}</p>
      <p
        className={`mt-2 text-sm leading-6 ${
          inverted ? "text-white/70" : "text-esc-black-soft/75"
        }`}
      >
        {detail}
      </p>
    </article>
  );
};

const PodiumCard = ({
  item,
  leaderTotal
}: {
  item: VoteResult;
  leaderTotal: number;
}) => {
  const rank = item.rank ?? 0;
  const isLeader = rank === 1;
  const gapToLeader = Math.max(leaderTotal - item.totalPts, 0);
  const paceWidth =
    leaderTotal > 0 && item.totalPts > 0
      ? Math.max(Math.round((item.totalPts / leaderTotal) * 100), 12)
      : 0;

  const heightClass =
    rank === 1 ? "lg:min-h-[29rem]" : rank === 2 ? "lg:min-h-[24rem]" : "lg:min-h-[22rem]";

  return (
    <article
      className={`relative overflow-hidden rounded-[2.1rem] border p-6 shadow-[0_26px_56px_rgba(17,17,17,0.09)] ${heightClass} ${
        isLeader
          ? "border-esc-pink/35 bg-[linear-gradient(180deg,rgba(19,19,19,0.97),rgba(42,24,34,0.94))] text-white"
          : "border-black/[0.08] bg-[linear-gradient(180deg,rgba(255,255,255,0.95),rgba(244,238,243,0.92))] text-esc-black"
      }`}
    >
      <div className={`absolute inset-x-0 top-0 h-1 ${isLeader ? "bg-esc-pink" : "bg-black/10"}`} />
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <p
            className={`text-[11px] uppercase tracking-[0.18em] ${
              isLeader ? "text-white/55" : "text-esc-muted"
            }`}
          >
            {rank === 1 ? "Current leader" : rank === 2 ? "Second place" : "Third place"}
          </p>
          <div className="mt-4 flex items-center gap-3">
            <div
              className={`rounded-[1.15rem] border p-1.5 ${
                isLeader ? "border-white/10 bg-white/10" : "border-white/80 bg-white/80"
              }`}
            >
              <img
                src={flagUrl(item.countryId)}
                alt={item.country}
                className="h-10 w-14 rounded-xl object-cover"
              />
            </div>
            <div className="min-w-0">
              <p className="truncate text-2xl font-semibold tracking-[-0.04em]">{item.country}</p>
              <p
                className={`truncate text-sm ${
                  isLeader ? "text-white/70" : "text-esc-black-soft/75"
                }`}
              >
                {item.name}
              </p>
            </div>
          </div>
        </div>

        <div
          className={`flex h-14 w-14 shrink-0 items-center justify-center rounded-[1.15rem] text-lg font-semibold ${
            isLeader ? "bg-white text-esc-black" : "bg-esc-black text-white"
          }`}
        >
          #{rank}
        </div>
      </div>

      <div className="mt-8 grid grid-cols-3 gap-2 text-center">
        <div
          className={`rounded-[1.15rem] border px-3 py-3 ${
            isLeader ? "border-white/10 bg-white/10" : "border-black/[0.08] bg-white/74"
          }`}
        >
          <p
            className={`text-[10px] uppercase tracking-[0.16em] ${
              isLeader ? "text-white/55" : "text-esc-muted"
            }`}
          >
            Public
          </p>
          <p className="mt-1.5 text-lg font-semibold">{item.escPublicPts}</p>
        </div>
        <div
          className={`rounded-[1.15rem] border px-3 py-3 ${
            isLeader ? "border-white/10 bg-white/10" : "border-black/[0.08] bg-white/74"
          }`}
        >
          <p
            className={`text-[10px] uppercase tracking-[0.16em] ${
              isLeader ? "text-white/55" : "text-esc-muted"
            }`}
          >
            Jury
          </p>
          <p className="mt-1.5 text-lg font-semibold">{item.juryPts}</p>
        </div>
        <div
          className={`rounded-[1.15rem] border px-3 py-3 ${
            isLeader
              ? "border-esc-pink/40 bg-esc-pink text-white"
              : "border-black/[0.08] bg-esc-black text-white"
          }`}
        >
          <p
            className={`text-[10px] uppercase tracking-[0.16em] ${
              isLeader ? "text-white/70" : "text-white/55"
            }`}
          >
            Total
          </p>
          <p className="mt-1.5 text-lg font-semibold">{item.totalPts}</p>
        </div>
      </div>

      <div className="mt-8">
        <div
          className={`flex items-center justify-between text-[11px] uppercase tracking-[0.18em] ${
            isLeader ? "text-white/55" : "text-esc-muted"
          }`}
        >
          <span>{isLeader ? "Share of the lead" : "Gap to first"}</span>
          <span>{isLeader ? `${paceWidth}%` : `${gapToLeader} pts`}</span>
        </div>
        <div className={`mt-2 h-2 overflow-hidden rounded-full ${isLeader ? "bg-white/10" : "bg-black/10"}`}>
          <div
            className={`h-full rounded-full ${isLeader ? "bg-esc-pink" : "bg-esc-black"}`}
            style={{ width: `${paceWidth}%` }}
          />
        </div>
      </div>
    </article>
  );
};

export const ResultsPage = () => {
  const { results, countdown, paused, setPaused } = useResultsPoll();

  const leader = results[0] ?? null;
  const leaderTotal = leader?.totalPts ?? 0;
  const totalPublic = results.reduce((sum, item) => sum + item.escPublicPts, 0);
  const totalJury = results.reduce((sum, item) => sum + item.juryPts, 0);
  const totalCombined = results.reduce((sum, item) => sum + item.totalPts, 0);
  const activeEntries = results.filter((item) => item.totalPts > 0).length;
  const leaderMargin = leader && results[1] ? leader.totalPts - results[1].totalPts : leader?.totalPts ?? 0;
  const podiumItems = [results[1], results[0], results[2]].filter(
    (item): item is VoteResult => Boolean(item)
  );

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate overflow-hidden bg-[linear-gradient(180deg,#f5edf2_0%,#fff8fb_38%,#ffffff_100%)]">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,4,144,0.18),transparent_38%),radial-gradient(circle_at_top_right,rgba(255,4,144,0.12),transparent_34%),linear-gradient(180deg,rgba(255,255,255,0)_0%,rgba(255,255,255,0.6)_100%)]" />
        <div className="pointer-events-none absolute inset-0 opacity-[0.42] [background-image:linear-gradient(rgba(17,17,17,0.045)_1px,transparent_1px),linear-gradient(90deg,rgba(17,17,17,0.045)_1px,transparent_1px)] [background-size:9rem_9rem]" />
        <div className="pointer-events-none absolute inset-y-0 left-[14%] hidden w-px bg-black/5 lg:block" />
        <div className="pointer-events-none absolute inset-y-0 right-[18%] hidden w-px bg-black/5 lg:block" />
        <div className="pointer-events-none absolute left-[-8rem] top-24 h-64 w-64 rounded-full bg-esc-pink/12 blur-3xl" />
        <div className="pointer-events-none absolute right-[-10rem] top-10 h-80 w-80 rounded-full bg-esc-pink/10 blur-3xl" />

        <div className="relative z-10 mx-auto max-w-7xl px-6 pb-16 pt-6 sm:px-10 lg:px-14 lg:pb-24 lg:pt-8">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-black/10 pb-4 text-[11px] uppercase tracking-[0.24em] text-esc-muted">
            <div>Live results</div>
            <div>{paused ? "Feed paused" : `Automatic sync in ${countdown}s`}</div>
          </div>

          <div className="mt-10 grid gap-8 xl:grid-cols-[minmax(0,1.14fr)_24rem] xl:items-start">
            <div className="w-full max-w-none pr-0 xl:pr-10">
              <div className="inline-flex items-center gap-2 rounded-full border border-black/10 bg-white/78 px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.22em] text-esc-black-soft shadow-[0_14px_34px_rgba(17,17,17,0.05)] backdrop-blur-sm">
                <span className="h-2 w-2 rounded-full bg-esc-pink" />
                Live ranking
              </div>

              <h1 className="mt-6 text-left text-[clamp(4.8rem,12vw,9.8rem)] font-bold leading-[0.88] tracking-[-0.095em] text-esc-black">
                <span className="block">Grand Final</span>
                <span className="block text-esc-pink">Scoreboard</span>
              </h1>

              <div className="mt-6 h-1 w-32 rounded-full bg-esc-pink" />

              <p className="mt-7 max-w-3xl text-base leading-8 text-esc-black-soft/80 sm:text-lg">
                Follow the latest standings as public and jury points come together. See who is
                leading right now, who is chasing the podium and how the full ranking evolves with
                every update.
              </p>

              <div className="mt-8 grid gap-3 sm:grid-cols-3 xl:max-w-4xl">
                {RESULTS_MARKERS.map((marker) => (
                  <div
                    key={marker}
                    className="rounded-[1.15rem] border border-black/10 bg-white/80 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-esc-black-soft shadow-[0_14px_34px_rgba(17,17,17,0.05)]"
                  >
                    {marker}
                  </div>
                ))}
              </div>
            </div>

            <aside className="relative overflow-hidden rounded-[2.2rem] border border-esc-pink/25 bg-[linear-gradient(180deg,rgba(18,18,18,0.97),rgba(34,24,29,0.95))] p-6 text-white shadow-[0_32px_70px_rgba(17,17,17,0.24)]">
              <div className="pointer-events-none absolute right-[-3rem] top-[-3rem] h-40 w-40 rounded-full bg-esc-pink/20 blur-3xl" />
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/55">Live leader</p>

              {leader ? (
                <>
                  <div className="mt-5 flex items-start justify-between gap-4">
                    <div className="min-w-0">
                      <p className="text-[11px] uppercase tracking-[0.18em] text-white/55">Now in first place</p>
                      <h2 className="mt-2 truncate text-4xl font-semibold tracking-[-0.05em]">{leader.country}</h2>
                      <p className="mt-2 truncate text-sm text-white/70">{leader.name}</p>
                    </div>
                    <div className="rounded-[1.25rem] border border-white/10 bg-white/10 p-2">
                      <img
                        src={flagUrl(leader.countryId)}
                        alt={leader.country}
                        className="h-12 w-16 rounded-xl object-cover"
                      />
                    </div>
                  </div>

                  <div className="mt-8 rounded-[1.6rem] border border-white/10 bg-white/10 px-5 py-5">
                    <p className="text-[11px] uppercase tracking-[0.18em] text-white/55">Current score</p>
                    <div className="mt-3 flex items-end justify-between gap-4">
                      <p className="text-5xl font-semibold tracking-[-0.06em]">{leader.totalPts}</p>
                      <p className="text-sm text-white/70">points total</p>
                    </div>
                  </div>
                </>
              ) : (
                <div className="mt-5 rounded-[1.6rem] border border-dashed border-white/15 bg-white/6 px-5 py-8 text-sm text-white/70">
                  The live leader will appear here as soon as the first scores come in.
                </div>
              )}

              <div className="mt-6 grid grid-cols-3 gap-2">
                <div className="rounded-[1.15rem] border border-white/10 bg-white/10 px-3 py-3">
                  <p className="text-[10px] uppercase tracking-[0.16em] text-white/55">Entries</p>
                  <p className="mt-1.5 text-lg font-semibold">{results.length}</p>
                </div>
                <div className="rounded-[1.15rem] border border-white/10 bg-white/10 px-3 py-3">
                  <p className="text-[10px] uppercase tracking-[0.16em] text-white/55">On board</p>
                  <p className="mt-1.5 text-lg font-semibold">{activeEntries}</p>
                </div>
                <div className="rounded-[1.15rem] border border-white/10 bg-white/10 px-3 py-3">
                  <p className="text-[10px] uppercase tracking-[0.16em] text-white/55">Lead</p>
                  <p className="mt-1.5 text-lg font-semibold">{leader ? leaderMargin : "-"}</p>
                </div>
              </div>

              <div className="mt-6 flex flex-wrap items-center gap-3">
                <button
                  className="inline-flex items-center gap-2 rounded-full bg-white px-5 py-2.5 text-sm font-semibold text-esc-black shadow-[0_16px_36px_rgba(0,0,0,0.18)] transition-transform duration-300 hover:-translate-y-0.5"
                  onClick={() => setPaused(!paused)}
                >
                  {paused ? "Resume live feed" : "Pause live feed"}
                </button>
                <span className="text-sm text-white/70">
                  {paused ? "Manual review mode" : `Next sync in ${countdown}s`}
                </span>
              </div>
            </aside>
          </div>

          <div className="mt-10 grid gap-4 md:grid-cols-3">
            <StatTile
              label="Ranked entries"
              value={results.length}
              detail="All countries currently visible in the live ranking."
            />
            <StatTile
              label="Points counted"
              value={totalCombined}
              detail={`Public ${totalPublic} and jury ${totalJury} are already included in this total.`}
            />
            <StatTile
              label="Lead over second"
              value={leader ? leaderMargin : "-"}
              detail={
                leader && results[1]
                  ? `${results[1].country} is currently the closest challenger.`
                  : leader
                    ? "Only one ranked country is showing so far."
                    : "Waiting for the first scores to come in."
              }
              inverted
            />
          </div>

          {results.length === 0 ? (
            <div className="mt-12 rounded-[2.2rem] border border-black/10 bg-[linear-gradient(180deg,rgba(255,255,255,0.88),rgba(245,240,244,0.9))] px-6 py-16 text-center shadow-[0_24px_56px_rgba(17,17,17,0.06)]">
              <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Live results</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-[-0.04em] text-esc-black">The scoreboard is ready</h2>
              <p className="mx-auto mt-4 max-w-2xl text-sm leading-7 text-esc-black-soft/75">
                As soon as the first points are counted, the top three and the full ranking will
                appear here automatically.
              </p>
            </div>
          ) : (
            <>
              <section className="relative left-1/2 mt-28 w-screen -translate-x-1/2 overflow-hidden bg-[linear-gradient(180deg,rgba(10,10,10,0.98),rgba(24,20,22,0.96))] py-14 sm:py-16 lg:py-20">
                <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(90deg,rgba(255,4,144,0.08),rgba(255,4,144,0)_18%,rgba(255,255,255,0.02)_48%,rgba(255,4,144,0)_82%,rgba(255,4,144,0.08))]" />
                <div className="pointer-events-none absolute inset-0 opacity-[0.12] [background-image:linear-gradient(rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:9rem_9rem]" />
                <div className="pointer-events-none absolute left-[-8rem] top-8 h-48 w-48 rounded-full bg-esc-pink/16 blur-3xl" />
                <div className="pointer-events-none absolute right-[-6rem] bottom-[-2rem] h-44 w-44 rounded-full bg-white/6 blur-3xl" />

                <div className="relative z-10 mx-auto max-w-7xl px-6 sm:px-10 lg:px-14">
                  <div className="flex flex-wrap items-end justify-between gap-4">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.2em] text-white/50">Current podium</p>
                      <h2 className="mt-2 text-3xl font-semibold tracking-[-0.04em] text-white">
                        Top three on stage
                      </h2>
                    </div>
                    <p className="max-w-xl text-sm leading-7 text-white">
                      The three leading countries right now, based on the latest combined public and jury points.
                    </p>
                  </div>

                  <div className="mt-10 grid gap-4 lg:grid-cols-[0.9fr,1.08fr,0.9fr] lg:items-end">
                    {podiumItems.map((item) => (
                      <PodiumCard key={item.id} item={item} leaderTotal={leaderTotal} />
                    ))}
                  </div>
                </div>
              </section>

              <section className="mt-14">
                <div className="flex flex-wrap items-end justify-between gap-4">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Complete ranking</p>
                    <h2 className="mt-2 text-3xl font-semibold tracking-[-0.04em] text-esc-black">
                      Every ranked entry
                    </h2>
                  </div>
                  <p className="max-w-xl text-sm leading-7 text-esc-black-soft/75">
                    Every country in the current order, from the leader down to the rest of the field.
                  </p>
                </div>

                <div className="mt-6 overflow-hidden rounded-[2rem] border border-black/10 bg-white/82 shadow-[0_24px_56px_rgba(17,17,17,0.06)] backdrop-blur-sm">
                  <div className="grid grid-cols-[4.5rem_minmax(0,1fr)_7rem] items-center gap-4 border-b border-black/10 px-5 py-4 text-[11px] uppercase tracking-[0.18em] text-esc-muted sm:grid-cols-[4.5rem_minmax(0,1fr)_9rem_9rem]">
                    <span>Rank</span>
                    <span>Entry</span>
                    <span className="hidden sm:block text-right">Split</span>
                    <span className="text-right">Total</span>
                  </div>
                  <div>
                    {results.map((item) => (
                      <ResultsRow key={item.id} item={item} leaderTotal={leaderTotal} />
                    ))}
                  </div>
                </div>
              </section>
            </>
          )}
        </div>
      </div>
    </section>
  );
};
