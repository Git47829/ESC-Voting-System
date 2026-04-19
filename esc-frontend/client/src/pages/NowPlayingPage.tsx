import { useMemo, useState } from "react";

import { api } from "../api/client";
import { useContestPoll } from "../hooks/useContestPoll";
import { flagUrl } from "../utils/flagUrl";
import { normalizeYoutubeUrl } from "../utils/normalizeYoutubeUrl";

const fieldClass =
  "mt-2 w-full border-b border-white/14 bg-transparent pb-3 text-base text-white placeholder:text-white/28 focus:border-esc-pink focus:outline-none";

const stepButtonClass =
  "inline-flex h-11 w-11 items-center justify-center rounded-full border text-lg font-semibold transition-colors";

export const NowPlayingPage = () => {
  const contest = useContestPoll();
  const [points, setPoints] = useState(1);
  const [phoneNum, setPhoneNum] = useState("");
  const [ownCountry, setOwnCountry] = useState("");

  const currentSong = contest?.currentSong ?? null;
  const currentPosition = contest ? contest.currentIndex + 1 : 0;
  const totalSongs = contest?.totalSongs ?? 0;
  const contestActive = contest?.contestActive ?? false;

  const progress = useMemo(() => {
    if (!contest || totalSongs === 0) return 0;
    return ((contest.currentIndex + 1) / totalSongs) * 100;
  }, [contest, totalSongs]);

  const currentSongEmbedUrl = useMemo(() => {
    if (!currentSong?.youtubeUrl) return "";
    const normalized = normalizeYoutubeUrl(currentSong.youtubeUrl);
    return `${normalized}${normalized.includes("?") ? "&" : "?"}rel=0&modestbranding=1&playsinline=1`;
  }, [currentSong]);

  const performerName = currentSong
    ? `${currentSong.artistFirstName} ${currentSong.artistLastName}`.trim()
    : "";

  if (!currentSong) {
    return (
      <section className="-mx-4 sm:-mx-6 lg:-mx-8">
        <div className="relative isolate overflow-hidden bg-[#070707] px-4 pb-16 pt-8 text-white sm:px-6 sm:pb-20 lg:px-8">
          <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,4,144,0.18),transparent_34%),radial-gradient(circle_at_top_right,rgba(255,4,144,0.12),transparent_30%),linear-gradient(180deg,rgba(255,255,255,0.02),rgba(255,255,255,0)_38%)]" />
          <div className="pointer-events-none absolute inset-0 opacity-[0.18] [background-image:linear-gradient(rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:8.5rem_8.5rem]" />
          <div className="pointer-events-none absolute left-[-8rem] top-16 h-72 w-72 rounded-full bg-esc-pink/20 blur-[140px]" />
          <div className="pointer-events-none absolute right-[-8rem] top-8 h-80 w-80 rounded-full bg-esc-pink/16 blur-[160px]" />

          <div className="relative z-10 mx-auto max-w-7xl">
            <div className="border-b border-white/10 pb-4 text-[11px] uppercase tracking-[0.24em] text-white/48">
              Stage feed standby
            </div>

            <div className="mt-14 rounded-[2.3rem] border border-white/10 bg-white/[0.03] px-6 py-14 text-center shadow-[0_30px_70px_rgba(0,0,0,0.28)] backdrop-blur-sm sm:px-10">
              <p className="text-[11px] uppercase tracking-[0.22em] text-white/50">Running now</p>
              <h1 className="mt-4 text-4xl font-semibold tracking-[-0.05em] sm:text-6xl">
                No performance is currently on stage
              </h1>
              <p className="mx-auto mt-5 max-w-2xl text-sm leading-7 text-white/68 sm:text-base">
                As soon as the next contest entry starts, this page will switch to the live stage,
                show the performance video and open the quick voting panel.
              </p>
            </div>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate overflow-hidden bg-[#070707] px-4 pb-16 pt-8 text-white sm:px-6 sm:pb-20 lg:px-8">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,4,144,0.18),transparent_34%),radial-gradient(circle_at_top_right,rgba(255,4,144,0.14),transparent_30%),radial-gradient(circle_at_50%_42%,rgba(255,255,255,0.035),transparent_48%)]" />
        <div className="pointer-events-none absolute inset-0 opacity-[0.18] [background-image:linear-gradient(rgba(255,255,255,0.08)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.08)_1px,transparent_1px)] [background-size:8.5rem_8.5rem]" />
        <div className="pointer-events-none absolute left-[-9rem] top-10 h-[28rem] w-[28rem] rounded-full bg-esc-pink/20 blur-[160px]" />
        <div className="pointer-events-none absolute right-[-10rem] top-8 h-[30rem] w-[30rem] rounded-full bg-esc-pink/14 blur-[180px]" />
        <div className="pointer-events-none absolute bottom-[-10rem] left-1/3 h-[24rem] w-[24rem] rounded-full bg-white/[0.035] blur-[150px]" />

        <div className="relative z-10 mx-auto max-w-7xl">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 pb-4 text-[11px] uppercase tracking-[0.24em] text-white/48">
            <div className="flex items-center gap-2 text-white/62">
              <span className="stage-glow h-2 w-2 rounded-full bg-esc-pink" />
              Live stage feed
            </div>
            <div className="rounded-full border border-white/10 bg-white/[0.04] px-3 py-1.5 text-white/62">
              Song {currentPosition} of {totalSongs}
            </div>
          </div>

          <div className="mt-10 grid gap-10 xl:grid-cols-[minmax(0,1.08fr)_22rem] xl:items-start">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/[0.05] px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.2em] text-white/68 backdrop-blur-sm">
                <span className="stage-glow h-2 w-2 rounded-full bg-esc-pink" />
                Running now
              </div>

              <h1 className="mt-6 text-[clamp(4.6rem,12vw,9.2rem)] font-bold leading-[0.88] tracking-[-0.095em] text-white">
                <span className="now-stage-title-line block">Live</span>
                <span className="now-stage-title-highlight block text-esc-pink">Stage</span>
              </h1>

              <p className="mt-7 max-w-3xl text-base leading-8 text-white/70 sm:text-lg">
                Follow the current performance, keep an eye on where the show is in the running
                order and cast your points while the song is live on stage.
              </p>

              <div className="mt-8 flex flex-wrap gap-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-white/70">
                <span className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2">
                  {currentSong.countryName}
                </span>
                <span className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2">
                  {contestActive ? "Voting open" : "Voting paused"}
                </span>
                <span className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2">
                  Refresh every 5s
                </span>
              </div>
            </div>

            <aside className="rounded-[2rem] border border-white/10 bg-white/[0.04] p-6 shadow-[0_28px_68px_rgba(0,0,0,0.28)] backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/50">On stage now</p>
              <div className="mt-5 flex items-center gap-4">
                <div className="rounded-[1.15rem] border border-white/10 bg-white/[0.08] p-1.5">
                  <img
                    src={flagUrl(currentSong.countryId)}
                    alt={currentSong.countryName}
                    className="h-12 w-16 rounded-xl object-cover"
                  />
                </div>
                <div className="min-w-0">
                  <p className="truncate text-2xl font-semibold tracking-[-0.04em] text-white">
                    {currentSong.countryName}
                  </p>
                  <p className="mt-1 truncate text-sm text-white/68">{currentSong.songName}</p>
                </div>
              </div>

              <div className="mt-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
                <div className="rounded-[1.25rem] border border-white/10 bg-black/25 px-4 py-4">
                  <p className="text-[10px] uppercase tracking-[0.18em] text-white/45">Artist</p>
                  <p className="mt-2 text-sm font-medium text-white">{performerName}</p>
                </div>
                <div className="rounded-[1.25rem] border border-white/10 bg-black/25 px-4 py-4">
                  <p className="text-[10px] uppercase tracking-[0.18em] text-white/45">Running order</p>
                  <p className="mt-2 text-sm font-medium text-white">
                    {currentPosition} / {totalSongs}
                  </p>
                </div>
              </div>

              <div className="mt-6">
                <div className="flex items-center justify-between text-[11px] uppercase tracking-[0.18em] text-white/50">
                  <span>Show progress</span>
                  <span>{Math.round(progress)}%</span>
                </div>
                <div className="mt-3 h-2 overflow-hidden rounded-full bg-white/10">
                  <div
                    className="h-full rounded-full bg-esc-pink transition-all duration-700"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              </div>
            </aside>
          </div>

          <div className="mt-12 grid gap-6 xl:grid-cols-[minmax(0,1.16fr)_minmax(320px,0.84fr)] xl:items-start">
            <section className="overflow-hidden rounded-[2.1rem] border border-white/10 bg-white/[0.035] shadow-[0_30px_70px_rgba(0,0,0,0.28)] backdrop-blur-sm">
              <div className="flex flex-wrap items-center justify-between gap-4 border-b border-white/10 px-6 py-5">
                <div>
                  <p className="text-[11px] uppercase tracking-[0.2em] text-white/50">Current performance</p>
                  <h2 className="mt-2 text-2xl font-semibold tracking-[-0.04em] text-white sm:text-3xl">
                    {currentSong.countryName}
                    <span className="text-esc-pink"> / </span>
                    {currentSong.songName}
                  </h2>
                </div>
                <span className="rounded-full border border-white/10 bg-white/[0.05] px-3 py-1.5 text-[11px] font-semibold uppercase tracking-[0.16em] text-white/64">
                  {currentSong.youtubeUrl ? "Live video" : "Audio only"}
                </span>
              </div>

              {currentSong.youtubeUrl ? (
                <div className="bg-black">
                  <iframe
                    title="Current performance"
                    className="aspect-video w-full"
                    src={currentSongEmbedUrl}
                    loading="eager"
                    referrerPolicy="strict-origin-when-cross-origin"
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                    allowFullScreen
                  />
                </div>
              ) : (
                <div className="flex aspect-video items-center justify-center bg-black/45 px-6 text-center text-sm text-white/60">
                  A live video is not available for this performance right now.
                </div>
              )}

              <div className="flex flex-wrap items-start justify-between gap-4 border-t border-white/10 px-6 py-5">
                <div>
                  <p className="text-[11px] uppercase tracking-[0.18em] text-white/45">Performed by</p>
                  <p className="mt-2 text-base font-medium text-white">{performerName}</p>
                  <p className="mt-2 max-w-2xl text-sm leading-7 text-white/64">
                    Keep this page open while the current act performs. The voting panel on the right
                    stays ready so points can be submitted without leaving the live stage.
                  </p>
                </div>

                <div className="flex flex-wrap gap-3 text-[11px] font-semibold uppercase tracking-[0.16em] text-white/65">
                  <span className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2">
                    Stage feed
                  </span>
                  <span className="rounded-full border border-white/10 bg-white/[0.05] px-4 py-2">
                    Quick vote ready
                  </span>
                </div>
              </div>
            </section>

            <aside className="rounded-[2.1rem] border border-esc-pink/20 bg-[linear-gradient(180deg,rgba(20,20,20,0.98),rgba(34,24,29,0.95))] p-6 shadow-[0_30px_70px_rgba(0,0,0,0.32)]">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/50">Quick vote</p>
              <h3 className="mt-3 text-3xl font-semibold tracking-[-0.05em] text-white">
                Cast points now
              </h3>
              <p className="mt-3 text-sm leading-7 text-white/68">
                Enter your phone number, add your own country code and choose how many points you
                want to assign to this performance.
              </p>

              <form
                className="mt-7 space-y-5"
                onSubmit={(event) => {
                  event.preventDefault();
                  void api.submitVote({
                    songID: currentSong.songId,
                    phoneNum,
                    ownCountry,
                    points
                  });
                }}
              >
                <div>
                  <label className="text-[11px] uppercase tracking-[0.18em] text-white/48">Phone number</label>
                  <input
                    type="tel"
                    className={fieldClass}
                    placeholder="Enter your phone number"
                    value={phoneNum}
                    onChange={(event) => setPhoneNum(event.target.value)}
                  />
                </div>

                <div>
                  <label className="text-[11px] uppercase tracking-[0.18em] text-white/48">Own country</label>
                  <input
                    className={fieldClass}
                    placeholder="For example DEU"
                    value={ownCountry}
                    maxLength={3}
                    onChange={(event) => setOwnCountry(event.target.value.toUpperCase().slice(0, 3))}
                  />
                </div>

                <div className="rounded-[1.7rem] border border-white/10 bg-black/20 px-5 py-5">
                  <div className="flex items-end justify-between gap-4">
                    <div>
                      <p className="text-[11px] uppercase tracking-[0.18em] text-white/45">Points</p>
                      <p className="mt-3 text-6xl font-semibold tracking-[-0.06em] text-esc-pink">
                        {points}
                      </p>
                    </div>

                    <div className="flex gap-2">
                      <button
                        type="button"
                        className={`${stepButtonClass} border-esc-pink/40 bg-esc-pink/12 text-esc-pink hover:bg-esc-pink hover:text-white`}
                        onClick={() => setPoints((value) => Math.min(20, value + 1))}
                      >
                        +
                      </button>
                      <button
                        type="button"
                        className={`${stepButtonClass} border-white/12 bg-white/[0.05] text-white/76 hover:border-white/24 hover:bg-white/[0.1]`}
                        onClick={() => setPoints((value) => Math.max(1, value - 1))}
                      >
                        -
                      </button>
                    </div>
                  </div>

                  <p className="mt-4 text-sm text-white/62">Choose a value between 1 and 20 points.</p>
                </div>

                <button
                  type="submit"
                  className="w-full rounded-full bg-esc-pink px-5 py-3 text-sm font-semibold uppercase tracking-[0.18em] text-white transition-transform duration-300 hover:-translate-y-0.5 hover:bg-esc-pink-dim"
                >
                  Submit vote
                </button>
              </form>
            </aside>
          </div>
        </div>
      </div>
    </section>
  );
};
