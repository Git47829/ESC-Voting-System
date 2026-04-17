import { useEffect, useMemo, useState } from "react";

import { api } from "../api/client";
import { useFlash } from "../context/FlashContext";
import { useContestPoll } from "../hooks/useContestPoll";
import { normalizeYoutubeUrl } from "../utils/normalizeYoutubeUrl";

export const NowPlayingPage = () => {
  const { contest, error, contestEnded } = useContestPoll();
  const [points, setPoints] = useState(1);
  const [phoneNum, setPhoneNum] = useState("");
  const [ownCountry, setOwnCountry] = useState("");
  const { addFlash } = useFlash();

  const progress = useMemo(() => {
    if (!contest || contest.totalSongs === 0) return 0;
    return ((contest.currentIndex + 1) / contest.totalSongs) * 100;
  }, [contest]);

  const currentSongEmbedUrl = useMemo(() => {
    if (!contest?.currentSong?.youtubeUrl) return "";
    const normalized = normalizeYoutubeUrl(contest.currentSong.youtubeUrl);
    return `${normalized}${normalized.includes("?") ? "&" : "?"}rel=0&modestbranding=1&playsinline=1`;
  }, [contest]);

  useEffect(() => {
    setPoints(1);
    setPhoneNum("");
    setOwnCountry("");
  }, [contest?.runId, contest?.currentIndex]);

  if (!contest?.currentSong) {
    return (
      <section className="-mx-4 sm:-mx-6 lg:-mx-8">
        <div className="relative isolate overflow-hidden px-4 pb-10 pt-8 sm:px-6 sm:pb-12 lg:px-8">
          <div className="pointer-events-none absolute inset-0 vote-section-wash" />
          <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
          <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
          <div className="pointer-events-none absolute left-[-24%] top-[2%] h-[56rem] w-[96rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.42)_0%,_rgba(255,4,144,0.24)_28%,_rgba(255,4,144,0.1)_50%,_rgba(255,4,144,0)_78%)] blur-[200px] opacity-96" />
          <div className="pointer-events-none absolute right-[-22%] top-[16%] h-[50rem] w-[84rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.3)_0%,_rgba(255,4,144,0.16)_32%,_rgba(255,4,144,0.06)_54%,_rgba(255,4,144,0)_80%)] blur-[190px] opacity-88" />
          <div className="relative z-10 mx-auto max-w-7xl">
            <section className="px-2 py-10 text-center sm:px-4">
              <h1 className="text-3xl font-bold text-white">Running Now</h1>
              <p className="mt-3 text-sm text-white/70">
                {error ?? "No active contest song."}
              </p>
              {contestEnded ? (
                <a
                  href="/results"
                  className="mt-5 inline-flex items-center rounded-xl border border-esc-pink/60 bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim"
                >
                  View live results
                </a>
              ) : null}
            </section>
          </div>
        </div>
      </section>
    );
  }

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate overflow-hidden px-4 pb-16 pt-8 sm:px-6 sm:pb-20 lg:px-8">
        {/* Background layers */}
        <div className="pointer-events-none absolute inset-0 vote-section-wash" />
        <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
        <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
        <div className="pointer-events-none absolute left-[-24%] top-[2%] h-[56rem] w-[96rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.42)_0%,_rgba(255,4,144,0.24)_28%,_rgba(255,4,144,0.1)_50%,_rgba(255,4,144,0)_78%)] blur-[200px] opacity-96" />
        <div className="pointer-events-none absolute right-[-22%] top-[16%] h-[50rem] w-[84rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.3)_0%,_rgba(255,4,144,0.16)_32%,_rgba(255,4,144,0.06)_54%,_rgba(255,4,144,0)_80%)] blur-[190px] opacity-88" />

        <div className="relative z-10 mx-auto max-w-7xl space-y-0">

          {/* ── HEADER ── */}
          <header className="px-2 pt-2 pb-8 sm:px-4 sm:pt-4">
            <div className="flex flex-wrap items-end justify-between gap-4">
              <div>
                <p className="text-xs uppercase tracking-[0.16em] text-white/50">Live contest</p>
                <h1 className="now-running-reveal mt-2 text-4xl font-bold sm:text-7xl">
                  <span className="now-running-reveal-bar" aria-hidden="true" />
                  <span className="now-running-reveal-text">Running Now</span>
                </h1>
              </div>
              <span className="inline-flex items-center gap-2 rounded-full border border-esc-pink/35 bg-black/20 px-4 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-esc-pink backdrop-blur-sm">
                <span className="h-2 w-2 rounded-full bg-esc-pink animate-stage-glow motion-reduce:animate-none" />
                Live
              </span>
            </div>
          </header>

          {/* ── NEON DIVIDER + PROGRESS ── */}
          <div
            className="px-2 pt-4 pb-8 sm:px-4"
          >
            <div className="flex items-center justify-between text-xs uppercase tracking-[0.14em] text-white/50 mb-2">
              <span>Song progress</span>
              <span style={{ color: "rgba(255,4,144,0.9)" }}>
                {contest.currentIndex + 1} / {contest.totalSongs}
              </span>
            </div>
            <div className="h-[3px] w-full overflow-hidden" style={{ background: "rgba(255,255,255,0.08)" }}>
              <div
                className="progress-shine h-full"
                style={{
                  width: `${progress}%`,
                  background: "linear-gradient(90deg, rgba(255,4,144,0.6), rgba(255,4,144,1), rgba(255,4,144,0.6))",
                  boxShadow: "0 0 10px rgba(255,4,144,0.8)",
                  transition: "width 0.6s ease",
                }}
              />
            </div>
          </div>

          {/* ── MAIN STAGE LAYOUT ── */}
          <div className="grid gap-0 pt-0 xl:grid-cols-[minmax(0,1.1fr)_minmax(320px,0.9fr)]">

            {/* LEFT: Video + Song info stacked flat */}
            <div
              style={{ borderRight: "1px solid rgba(255,255,255,0.07)" }}
              className="pr-0 xl:pr-8 space-y-0"
            >
              {/* Video — full bleed, no container */}
              {contest.currentSong.youtubeUrl ? (
                <div className="overflow-hidden" style={{ borderRadius: "0.5rem" }}>
                  <iframe
                    title="Current performance"
                    className="aspect-video w-full block"
                    src={currentSongEmbedUrl}
                    loading="eager"
                    referrerPolicy="strict-origin-when-cross-origin"
                    allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
                    allowFullScreen
                  />
                </div>
              ) : null}

              {/* Song info — flat text block below video */}
              <div
                className="pt-6 pb-8"
                style={{ borderBottom: "1px solid rgba(255,255,255,0.07)" }}
              >
                <p
                  className="text-xs uppercase tracking-[0.18em] mb-2"
                  style={{ color: "rgba(255,4,144,0.8)" }}
                >
                  Current performance
                </p>
                <h2 className="text-3xl font-bold text-white leading-tight">
                  {contest.currentSong.countryName}
                  <span style={{ color: "rgba(255,4,144,0.9)" }}> — </span>
                  {contest.currentSong.songName}
                </h2>
                <p
                  className="mt-2 text-base tracking-wide"
                  style={{ color: "rgba(255,255,255,0.5)" }}
                >
                  {contest.currentSong.artistFirstName} {contest.currentSong.artistLastName}
                </p>
              </div>
            </div>

            {/* RIGHT: Vote section — no card, pure layout */}
            <div className="xl:pl-8 pt-8 xl:pt-6 flex flex-col justify-start">
              <p
                className="text-xs uppercase tracking-[0.18em] mb-1"
                style={{ color: "rgba(255,4,144,0.8)" }}
              >
                Quick vote
              </p>
              <h3 className="text-3xl font-bold text-white mb-8">Cast points now</h3>

              <form
                className="space-y-4"
                onSubmit={async (e) => {
                  e.preventDefault();
                  try {
                    await api.submitVote({
                      songID: contest.currentSong!.songId,
                      phoneNum,
                      ownCountry,
                      points
                    });
                    addFlash("Vote submitted", "success");
                    setPoints(1);
                  } catch (error: unknown) {
                    addFlash(error instanceof Error ? error.message : "Vote failed", "error");
                  }
                }}
              >
                {/* Phone input */}
                <div>
                  <label
                    className="block text-xs uppercase tracking-[0.12em] mb-2"
                    style={{ color: "rgba(255,255,255,0.4)" }}
                  >
                    Phone
                  </label>
                  <input
                    style={{
                      width: "100%",
                      background: "transparent",
                      border: "none",
                      borderBottom: "1px solid rgba(255,255,255,0.2)",
                      color: "white",
                      padding: "0.5rem 0",
                      fontSize: "1rem",
                      outline: "none",
                    }}
                    placeholder="—"
                    value={phoneNum}
                    onChange={(e) => setPhoneNum(e.target.value)}
                    onFocus={(e) => (e.currentTarget.style.borderBottomColor = "rgba(255,4,144,0.9)")}
                    onBlur={(e) => (e.currentTarget.style.borderBottomColor = "rgba(255,255,255,0.2)")}
                  />
                </div>

                {/* Country input */}
                <div>
                  <label
                    className="block text-xs uppercase tracking-[0.12em] mb-2"
                    style={{ color: "rgba(255,255,255,0.4)" }}
                  >
                    Own Country
                  </label>
                  <input
                    style={{
                      width: "100%",
                      background: "transparent",
                      border: "none",
                      borderBottom: "1px solid rgba(255,255,255,0.2)",
                      color: "white",
                      padding: "0.5rem 0",
                      fontSize: "1rem",
                      outline: "none",
                    }}
                    placeholder="—"
                    value={ownCountry}
                    onChange={(e) => setOwnCountry(e.target.value.toUpperCase())}
                    onFocus={(e) => (e.currentTarget.style.borderBottomColor = "rgba(255,4,144,0.9)")}
                    onBlur={(e) => (e.currentTarget.style.borderBottomColor = "rgba(255,255,255,0.2)")}
                  />
                </div>

                {/* Points — big display number */}
                <div>
                  <label
                    className="block text-xs uppercase tracking-[0.12em] mb-2"
                    style={{ color: "rgba(255,255,255,0.4)" }}
                  >
                    Points
                  </label>
                  <div className="flex items-center gap-4">
                    <span
                      style={{
                        fontSize: "3.5rem",
                        fontWeight: 700,
                        lineHeight: 1,
                        color: "rgba(255,4,144,1)",
                        textShadow: "0 0 20px rgba(255,4,144,0.5)",
                        minWidth: "3rem",
                        display: "inline-block",
                      }}
                    >
                      {points}
                    </span>
                    <div className="flex flex-col gap-2">
                      <button
                        type="button"
                        onClick={() => setPoints((p) => Math.min(20, p + 1))}
                        style={{
                          background: "rgba(255,4,144,0.15)",
                          border: "1px solid rgba(255,4,144,0.4)",
                          color: "rgba(255,4,144,1)",
                          borderRadius: "0.25rem",
                          width: "2rem",
                          height: "1.75rem",
                          cursor: "pointer",
                          fontSize: "1rem",
                          lineHeight: 1,
                        }}
                      >
                        +
                      </button>
                      <button
                        type="button"
                        onClick={() => setPoints((p) => Math.max(1, p - 1))}
                        style={{
                          background: "rgba(255,255,255,0.06)",
                          border: "1px solid rgba(255,255,255,0.15)",
                          color: "rgba(255,255,255,0.6)",
                          borderRadius: "0.25rem",
                          width: "2rem",
                          height: "1.75rem",
                          cursor: "pointer",
                          fontSize: "1rem",
                          lineHeight: 1,
                        }}
                      >
                        −
                      </button>
                    </div>
                    <span
                      style={{ fontSize: "0.75rem", color: "rgba(255,255,255,0.3)", alignSelf: "flex-end", paddingBottom: "0.25rem" }}
                    >
                      max 20
                    </span>
                  </div>
                </div>

                {/* Vote button — full neon */}
                <div className="pt-4">
                  <button
                    type="submit"
                    style={{
                      width: "100%",
                      padding: "0.85rem 1.5rem",
                      background: "rgba(255,4,144,1)",
                      border: "none",
                      color: "white",
                      fontWeight: 700,
                      fontSize: "0.875rem",
                      letterSpacing: "0.12em",
                      textTransform: "uppercase",
                      cursor: "pointer",
                      borderRadius: "0.25rem",
                      boxShadow: "0 0 24px rgba(255,4,144,0.5), 0 0 4px rgba(255,4,144,0.8)",
                      transition: "box-shadow 0.2s ease",
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.boxShadow = "0 0 40px rgba(255,4,144,0.8), 0 0 8px rgba(255,4,144,1)";
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.boxShadow = "0 0 24px rgba(255,4,144,0.5), 0 0 4px rgba(255,4,144,0.8)";
                    }}
                  >
                    Vote now
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
};
