import { useMemo, useState } from "react";

import { api } from "../api/client";
import { useContestPoll } from "../hooks/useContestPoll";

export const NowPlayingPage = () => {
  const contest = useContestPoll();
  const [points, setPoints] = useState(1);
  const [phoneNum, setPhoneNum] = useState("");
  const [ownCountry, setOwnCountry] = useState("");

  const progress = useMemo(() => {
    if (!contest || contest.totalSongs === 0) return 0;
    return ((contest.currentIndex + 1) / contest.totalSongs) * 100;
  }, [contest]);

  if (!contest?.currentSong) {
    return (
      <section className="rounded-[1.75rem] border border-esc-border bg-white/90 p-8 text-center shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
        <h1 className="text-3xl font-bold text-esc-black">Running Now</h1>
        <p className="mt-3 text-sm text-esc-muted">No active contest song.</p>
      </section>
    );
  }

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <div className="flex flex-wrap items-end justify-between gap-4 border-b border-esc-border pb-5">
          <div>
            <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Live contest</p>
            <h1 className="mt-2 text-4xl font-bold text-esc-black">Running Now</h1>
          </div>
          <span className="inline-flex items-center gap-2 rounded-full border border-esc-pink/30 bg-esc-pink-soft px-4 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-esc-pink">
            <span className="h-2 w-2 rounded-full bg-esc-pink animate-stage-glow motion-reduce:animate-none" />
            Live
          </span>
        </div>

        <div className="mt-5 space-y-2">
          <div className="flex items-center justify-between text-xs uppercase tracking-[0.14em] text-esc-muted">
            <span>Song progress</span>
            <span>
              {contest.currentIndex + 1} / {contest.totalSongs}
            </span>
          </div>
          <div className="h-2.5 overflow-hidden rounded-full bg-esc-surface2">
            <div
              className="progress-shine h-2.5 rounded-full bg-gradient-to-r from-esc-pink-dim via-esc-pink to-esc-pink-dim"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
      </div>

      <div className="grid gap-6 xl:grid-cols-[minmax(0,1.1fr)_minmax(340px,0.9fr)]">
        <div className="space-y-6">
          {contest.currentSong.youtubeUrl ? (
            <div className="overflow-hidden rounded-[1.75rem] border border-esc-border bg-white/88 shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
              <iframe
                title="Current performance"
                className="aspect-video w-full"
                src={contest.currentSong.youtubeUrl}
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                allowFullScreen
              />
            </div>
          ) : null}

          <article className="rounded-[1.75rem] border border-esc-border bg-white/92 p-6 shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
            <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Current performance</p>
            <h2 className="mt-2 text-2xl font-bold text-esc-black">
              {contest.currentSong.countryName} - {contest.currentSong.songName}
            </h2>
            <p className="mt-2 text-sm text-esc-muted">
              {contest.currentSong.artistFirstName} {contest.currentSong.artistLastName}
            </p>
          </article>
        </div>

        <form
          className="rounded-[1.75rem] border border-esc-border bg-white/92 p-6 shadow-[0_16px_36px_rgba(0,0,0,0.05)]"
          onSubmit={(e) => {
            e.preventDefault();
            void api.submitVote({
              songID: contest.currentSong!.songId,
              phoneNum,
              ownCountry,
              points
            });
          }}
        >
          <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Quick vote</p>
          <h3 className="mt-2 text-2xl font-bold text-esc-black">Cast points now</h3>
          <div className="mt-6 grid gap-3">
            <input
              className="w-full rounded-xl border border-esc-border bg-white px-3 py-2 text-esc-black placeholder:text-esc-muted/70 focus:border-esc-pink"
              placeholder="Phone"
              value={phoneNum}
              onChange={(e) => setPhoneNum(e.target.value)}
            />
            <input
              className="w-full rounded-xl border border-esc-border bg-white px-3 py-2 text-esc-black placeholder:text-esc-muted/70 focus:border-esc-pink"
              placeholder="Own Country"
              value={ownCountry}
              onChange={(e) => setOwnCountry(e.target.value.toUpperCase())}
            />
            <input
              className="w-full rounded-xl border border-esc-border bg-white px-3 py-2 text-esc-black focus:border-esc-pink"
              type="number"
              min={1}
              max={20}
              value={points}
              onChange={(e) => setPoints(Number(e.target.value))}
            />
            <button className="mt-1 rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim hover:border-esc-pink-dim">
              Vote now
            </button>
          </div>
        </form>
      </div>
    </section>
  );
};

