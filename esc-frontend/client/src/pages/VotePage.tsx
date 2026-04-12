import { useEffect, useMemo, useState } from "react";

import { api } from "../api/client";
import { CountryCard } from "../components/ui/CountryCard";
import { BudgetBar } from "../components/vote/BudgetBar";
import { SubmitModal } from "../components/vote/SubmitModal";
import { VoteBasket } from "../components/vote/VoteBasket";
import { useCookieConsent } from "../context/CookieConsentContext";
import { useFlash } from "../context/FlashContext";
import type { Song } from "../types";

const TOTAL = 20;
const HERO_WORDS = ["Cast", "Your", "Votes"];
const HERO_CHIPS = ["Live televote", "20 points to spend", "Fast, secure, editorial"];

export const VotePage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [selection, setSelection] = useState<Record<number, number>>({});
  const [openSubmit, setOpenSubmit] = useState(false);
  const { addFlash } = useFlash();
  const { consent } = useCookieConsent();

  useEffect(() => {
    void api.getSongs().then(setSongs);
  }, []);

  const used = useMemo(
    () => Object.values(selection).reduce((sum, value) => sum + value, 0),
    [selection]
  );
  const remaining = TOTAL - used;

  const changePoints = (songId: number, delta: number) => {
    setSelection((current) => {
      const nextValue = Math.max(0, (current[songId] ?? 0) + delta);
      const next = { ...current, [songId]: nextValue };
      const nextUsed = Object.values(next).reduce((sum, value) => sum + value, 0);
      if (nextUsed > TOTAL) return current;
      return next;
    });
  };

  const submit = async (phone: string, ownCountry: string) => {
    if (!consent?.preferences.essential) {
      addFlash("Bitte zuerst erforderliche Vote-Cookies akzeptieren.", "error");
      return;
    }
    const tasks = Object.entries(selection)
      .filter(([, points]) => points > 0)
      .map(([songID, points]) =>
        api.submitVote({ songID: Number(songID), phoneNum: phone, ownCountry, points })
      );
    await Promise.all(tasks);
    setOpenSubmit(false);
    addFlash("Votes submitted", "success");
    setSelection({});
  };

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="bg-esc-white">
        <section className="relative isolate overflow-hidden border-b border-esc-border bg-esc-white px-6 pb-12 pt-6 sm:px-10 sm:pb-16 lg:px-14 lg:pb-20 lg:pt-8">
          <div className="pointer-events-none absolute inset-0 hero-grid-lines opacity-70" />
          <div className="pointer-events-none absolute inset-0 hero-noise opacity-[0.16] mix-blend-soft-light" />
          <div className="pointer-events-none absolute -left-20 top-20 h-72 w-72 rounded-full bg-esc-pink/12 blur-3xl animate-hero-aurora" />
          <div className="pointer-events-none absolute right-[-8rem] top-[8%] h-[28rem] w-[28rem] rounded-full bg-esc-pink/10 blur-3xl animate-hero-aurora-delayed" />
          <div className="pointer-events-none absolute bottom-[-8rem] left-[20%] h-[22rem] w-[22rem] rounded-full bg-black/5 blur-3xl animate-hero-aurora" />
          <div className="hero-spotlight pointer-events-none absolute inset-y-[-20%] left-1/2 hidden w-[36rem] -translate-x-1/2 opacity-80 md:block" />

          <div className="relative mx-auto flex min-h-[100svh] w-full max-w-7xl flex-col justify-between gap-14 lg:min-h-[960px]">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-esc-border/80 pb-4 text-[11px] uppercase tracking-[0.24em] text-esc-muted">
              <div
                className="opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "60ms" }}
              >
                Eurovision-style voting experience
              </div>
              <div
                className="opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "120ms" }}
              >
                Live • Premium motion • Interactive reveal
              </div>
            </div>

            <div className="grid items-center gap-14 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:gap-10">
              <div className="relative z-10 max-w-4xl">
                <div
                  className="mb-6 inline-flex items-center gap-3 rounded-full border border-esc-border bg-white/80 px-4 py-2 text-[11px] uppercase tracking-[0.2em] text-esc-muted shadow-[0_16px_40px_rgba(17,17,17,0.06)] backdrop-blur-sm opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "140ms" }}
                >
                  <span className="h-2 w-2 rounded-full bg-esc-pink animate-stage-glow motion-reduce:animate-none" />
                  Designed to feel like a launch sequence
                </div>

                <div className="space-y-1 sm:space-y-2">
                  {HERO_WORDS.map((word, index) => (
                    <div key={word} className="overflow-hidden">
                      <h1 className="hero-display text-balance">
                        <span
                          className="inline-block translate-y-[115%] opacity-0 animate-hero-word motion-reduce:animate-none motion-reduce:translate-y-0 motion-reduce:opacity-100"
                          style={{ animationDelay: `${220 + index * 140}ms` }}
                        >
                          {word}
                        </span>
                      </h1>
                    </div>
                  ))}
                </div>

                <p
                  className="mt-8 max-w-2xl text-base leading-7 text-esc-black-soft/80 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100 sm:text-lg"
                  style={{ animationDelay: "620ms" }}
                >
                  Allocate your 20 points with a smoother, more cinematic opening: strong typography,
                  soft light movement, floating status cards and a cleaner transition into the actual
                  voting flow.
                </p>

                <div
                  className="mt-8 flex flex-wrap gap-3 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "760ms" }}
                >
                  {HERO_CHIPS.map((chip) => (
                    <span key={chip} className="hero-chip">
                      {chip}
                    </span>
                  ))}
                </div>

                <div
                  className="mt-10 flex flex-wrap items-center gap-5 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "840ms" }}
                >
                  <a
                    href="#voting-grid"
                    className="inline-flex items-center gap-3 rounded-full bg-esc-black px-6 py-3 text-sm font-semibold text-white shadow-[0_20px_40px_rgba(17,17,17,0.18)] transition-transform duration-300 hover:-translate-y-0.5"
                  >
                    Start voting
                    <span aria-hidden="true">↓</span>
                  </a>
                  <p className="text-sm text-esc-muted">
                    Scroll into the grid and distribute your points in real time.
                  </p>
                </div>
              </div>

              <div
                className="relative mx-auto flex w-full max-w-[34rem] items-center justify-center opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "520ms" }}
              >
                <div className="pointer-events-none absolute inset-10 rounded-full border border-esc-border/70" />
                <div className="pointer-events-none absolute inset-[18%] rounded-full border border-esc-border/60" />
                <div className="pointer-events-none absolute inset-[30%] rounded-full border border-esc-border/50" />

                <div className="relative aspect-[0.94] w-full max-w-[31rem]">
                  <div className="hero-card absolute left-[4%] top-[10%] w-[62%] animate-hero-card-float motion-reduce:animate-none">
                    <div className="flex items-start justify-between gap-4">
                      <div>
                        <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Vote budget</p>
                        <p className="mt-2 text-3xl font-bold text-esc-black">20 pts</p>
                      </div>
                      <div className="rounded-full bg-esc-pink-soft px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-esc-pink">
                        Live
                      </div>
                    </div>
                    <div className="mt-5 h-2 overflow-hidden rounded-full bg-esc-pink-soft">
                      <div className="progress-shine h-full w-[72%] rounded-full bg-gradient-to-r from-esc-pink-dim via-esc-pink to-esc-pink-dim" />
                    </div>
                    <div className="mt-4 flex items-center justify-between text-sm text-esc-muted">
                      <span>Ready to allocate</span>
                      <span>12 / 20 used</span>
                    </div>
                  </div>

                  <div className="hero-card absolute right-[2%] top-[29%] w-[54%] animate-hero-card-float-delayed motion-reduce:animate-none">
                    <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Now playing</p>
                    <div className="mt-4 flex items-center gap-3">
                      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-esc-black text-sm font-semibold text-white shadow-[0_12px_26px_rgba(17,17,17,0.18)]">
                        ESC
                      </div>
                      <div>
                        <p className="text-base font-semibold text-esc-black">Stage is live</p>
                        <p className="text-sm text-esc-muted">Animated hero overlay</p>
                      </div>
                    </div>
                    <div className="mt-5 flex gap-2">
                      <span className="hero-mini-pill">Editorial type</span>
                      <span className="hero-mini-pill">Motion depth</span>
                    </div>
                  </div>

                  <div className="hero-card absolute bottom-[11%] left-[10%] w-[58%] animate-hero-card-float motion-reduce:animate-none">
                    <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Secure submit</p>
                    <div className="mt-4 flex items-center gap-3">
                      <div className="flex h-11 w-11 items-center justify-center rounded-full border border-esc-border bg-white text-lg">
                        ✓
                      </div>
                      <div>
                        <p className="text-base font-semibold text-esc-black">Clean transition</p>
                        <p className="text-sm text-esc-muted">From hero directly into voting grid</p>
                      </div>
                    </div>
                  </div>

                  <div className="absolute right-[13%] top-[14%] h-28 w-28 rounded-full border border-esc-border/70 bg-white/40 backdrop-blur-md animate-hero-aurora motion-reduce:animate-none" />
                </div>
              </div>
            </div>

            <div className="flex flex-wrap items-end justify-between gap-6 border-t border-esc-border/80 pt-6">
              <div
                className="max-w-md opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "920ms" }}
              >
                <p className="text-[11px] uppercase tracking-[0.22em] text-esc-muted">Motion direction</p>
                <p className="mt-3 text-sm leading-6 text-esc-black-soft/75">
                  Spotlight sweep, layered gradients, floating cards and stronger editorial typography
                  instead of three isolated full-screen fades.
                </p>
              </div>

              <a
                href="#voting-grid"
                className="inline-flex items-center gap-3 text-[11px] uppercase tracking-[0.22em] text-esc-muted transition-transform duration-300 hover:translate-y-0.5"
              >
                <span>Scroll to voting grid</span>
                <span className="flex h-10 w-10 items-center justify-center rounded-full border border-esc-border bg-white animate-scroll-nudge motion-reduce:animate-none">
                  ↓
                </span>
              </a>
            </div>
          </div>
        </section>
      </div>

      <div
        id="voting-grid"
        className="border-t border-esc-border bg-esc-surface2/60 px-4 pb-28 pt-16 sm:px-6 lg:px-8"
      >
        <div className="mx-auto max-w-7xl space-y-8">
          <div className="space-y-3">
            <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Live Experience</p>
            <h3 className="text-4xl font-bold text-esc-black">Cast Your Votes</h3>
          </div>

          <BudgetBar remaining={remaining} total={TOTAL} />

          <div className="grid gap-4 md:grid-cols-2 xl:gap-5">
            {songs.map((song) => (
              <CountryCard
                key={song.songId}
                song={song}
                points={selection[song.songId] ?? 0}
                onChange={changePoints}
              />
            ))}
          </div>

          <VoteBasket total={used} onSubmit={() => setOpenSubmit(true)} />
          <SubmitModal
            open={openSubmit}
            totalPoints={used}
            onClose={() => setOpenSubmit(false)}
            onSubmit={submit}
          />
        </div>
      </div>
    </section>
  );
};

