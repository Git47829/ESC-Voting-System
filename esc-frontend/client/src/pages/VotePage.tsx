import { useEffect, useMemo, useState } from "react";

import EurovisionHeart from "../../img/EurovisionHeart.png";
import { api } from "../api/client";
import { CountryCard } from "../components/ui/CountryCard";
import { BudgetBar } from "../components/vote/BudgetBar";
import { SubmitModal } from "../components/vote/SubmitModal";
import { VoteBasket } from "../components/vote/VoteBasket";
import { useCookieConsent } from "../context/CookieConsentContext";
import { useFlash } from "../context/FlashContext";
import type { Song } from "../types";
import { flagUrl } from "../utils/flagUrl";

const TOTAL = 20;
const HERO_WORDS = ["Cast", "Your", "Votes"];
const HERO_BAR_WIDTHS = ["62%", "54%", "72%"];
const HERO_CHIPS = ["Live voting", "20 points to spend", "Choose your favorites", "Submit when ready"];

export const VotePage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [selection, setSelection] = useState<Record<number, number>>({});
  const [serverRemaining, setServerRemaining] = useState(TOTAL);
  const [hasSubmitted, setHasSubmitted] = useState(false);
  const [openSubmit, setOpenSubmit] = useState(false);
  const [heroAccentActive, setHeroAccentActive] = useState(false);
  const [heroSweepX, setHeroSweepX] = useState(-20);
  const { addFlash } = useFlash();
  const { consent } = useCookieConsent();

  const votingLocked = hasSubmitted || serverRemaining === 0;

  useEffect(() => {
    void Promise.all([api.getSongs(), api.getVoteState()])
      .then(([songs, state]) => {
        setSongs(songs);
        setServerRemaining(state.votesRemaining);
        if (Object.keys(state.votesCast).length > 0) {
          setSelection(state.votesCast);
        }
        if (state.votesRemaining === 0 && Object.keys(state.votesCast).length > 0) {
          setHasSubmitted(true);
        }
      })
      .catch((error: unknown) => {
        addFlash(error instanceof Error ? error.message : "Failed to load voting data", "error");
      });
  }, [addFlash]);

  useEffect(() => {
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");

    const onScroll = () => {
      if (media.matches) {
        setHeroSweepX(-20);
        return;
      }

      const maxScroll = 700;
      const progress = Math.max(0, Math.min(window.scrollY / maxScroll, 1));
      const nextX = -20 + progress * 48;
      setHeroSweepX(nextX);
    };

    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });

    return () => {
      window.removeEventListener("scroll", onScroll);
    };
  }, []);

  useEffect(() => {
    const onScroll = () => {
      if (window.scrollY > 24) {
        setHeroAccentActive(true);
        window.removeEventListener("scroll", onScroll);
      }
    };

    if (window.scrollY > 24) {
      setHeroAccentActive(true);
      return;
    }

    window.addEventListener("scroll", onScroll, { passive: true });

    return () => {
      window.removeEventListener("scroll", onScroll);
    };
  }, []);

  const used = useMemo(
    () => Object.values(selection).reduce((sum, value) => sum + value, 0),
    [selection]
  );
  const remaining = TOTAL - used;

  const selectedSongs = useMemo(
    () => songs.filter((song) => (selection[song.songId] ?? 0) > 0),
    [selection, songs]
  );

  const topSelection = useMemo(() => {
    return (
      selectedSongs
        .map((song) => ({ ...song, points: selection[song.songId] ?? 0 }))
        .sort((a, b) => b.points - a.points)[0] ?? null
    );
  }, [selectedSongs, selection]);

  const changePoints = (songId: number, delta: number) => {
    if (votingLocked) return;
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
    const entries = Object.entries(selection).filter(([, points]) => points > 0);
    let failed = false;
    for (const [songID, points] of entries) {
      try {
        await api.submitVote({ songID: Number(songID), phoneNum: phone, ownCountry, points });
      } catch (error: unknown) {
        failed = true;
        const msg = error instanceof Error ? error.message : "Vote submission failed";
        addFlash(msg, "error");
        break;
      }
    }
    const state = await api.getVoteState();
    setSelection(state.votesCast);
    setServerRemaining(state.votesRemaining);
    setOpenSubmit(false);
    if (state.votesRemaining === 0) {
      setHasSubmitted(true);
    }
    if (!failed) {
      setHasSubmitted(true);
      addFlash("Votes submitted", "success");
    }
  };

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate bg-white">
        <div className="pointer-events-none absolute inset-0 vote-page-surface" />
        <div className="relative isolate overflow-hidden px-6 pb-12 pt-6 sm:px-10 sm:pb-16 lg:px-14 lg:pb-20 lg:pt-8">
          <div className="pointer-events-none absolute inset-0 hero-pink-wash" />

          <div className="pointer-events-none absolute inset-0 z-[2] overflow-hidden">
            <div className="absolute left-1/2 top-[7%] w-[min(88vw,30rem)] -translate-x-1/2 sm:top-[6%] sm:w-[min(78vw,36rem)] md:top-[5%] md:w-[min(70vw,44rem)] lg:top-[4%] lg:w-[min(64vw,52rem)] xl:top-[3%] xl:w-[58rem]">
              <img
                src={EurovisionHeart}
                alt=""
                aria-hidden="true"
                className="block w-full object-contain opacity-[0.34]"
              />

              <div className="absolute inset-0 flex flex-col items-center justify-center -translate-y-3 text-center">
                <span className="heart-copy animate-heart-copy-1 font-heading text-[clamp(1rem,2vw,1.6rem)] font-semibold uppercase tracking-[0.18em] text-white/70">
                  United.
                </span>
                <span className="heart-copy animate-heart-copy-2 mt-1 font-heading text-[clamp(1rem,2vw,1.6rem)] font-semibold uppercase tracking-[0.18em] text-white/70">
                  By.
                </span>
                <span className="heart-copy animate-heart-copy-3 mt-1 font-heading text-[clamp(1rem,2vw,1.6rem)] font-semibold uppercase tracking-[0.18em] text-white/70">
                  Music.
                </span>
              </div>
            </div>
          </div>

          <div
            className="pointer-events-none absolute top-[6%] z-[1] hidden h-[28rem] w-[62rem] -rotate-[9deg] rounded-full opacity-75 blur-[95px] md:block"
            style={{
              left: `${heroSweepX}%`,
              background:
                "linear-gradient(90deg, rgba(255,4,144,0) 0%, rgba(255,4,144,0.12) 16%, rgba(255,4,144,0.3) 50%, rgba(255,4,144,0.12) 84%, rgba(255,4,144,0) 100%)"
            }}
          />
          <div
            className="pointer-events-none absolute top-[10%] z-[1] hidden h-[24rem] w-[50rem] -rotate-[9deg] rounded-full opacity-60 blur-[125px] md:block"
            style={{
              left: `${heroSweepX + 8}%`,
              background:
                "linear-gradient(90deg, rgba(255,4,144,0) 0%, rgba(255,4,144,0.08) 18%, rgba(255,4,144,0.2) 50%, rgba(255,4,144,0.08) 82%, rgba(255,4,144,0) 100%)"
            }}
          />
          <div className="pointer-events-none absolute inset-y-0 left-0 hidden w-[clamp(7rem,14vw,16rem)] bg-gradient-to-r from-esc-pink/16 via-esc-pink/8 to-transparent blur-2xl opacity-55 sm:block md:w-[clamp(9rem,16vw,20rem)] lg:w-[clamp(11rem,18vw,24rem)] lg:opacity-70" />
          <div className="pointer-events-none absolute inset-y-0 right-0 hidden w-[clamp(7rem,14vw,16rem)] bg-gradient-to-l from-esc-pink/16 via-esc-pink/8 to-transparent blur-2xl opacity-55 sm:block md:w-[clamp(9rem,16vw,20rem)] lg:w-[clamp(11rem,18vw,24rem)] lg:opacity-70" />
          <div className="pointer-events-none absolute inset-y-[14%] left-0 hidden w-[clamp(6rem,11vw,14rem)] bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.14)_0%,_rgba(255,4,144,0.08)_38%,_rgba(255,4,144,0)_74%)] opacity-55 sm:block lg:inset-y-[10%] lg:w-[clamp(8rem,13vw,18rem)] lg:opacity-75" />
          <div className="pointer-events-none absolute inset-y-[14%] right-0 hidden w-[clamp(6rem,11vw,14rem)] bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.14)_0%,_rgba(255,4,144,0.08)_38%,_rgba(255,4,144,0)_74%)] opacity-55 sm:block lg:inset-y-[10%] lg:w-[clamp(8rem,13vw,18rem)] lg:opacity-75" />
          <div className="pointer-events-none absolute inset-0 hero-grid-lines opacity-70" />
          <div className="pointer-events-none absolute inset-0 hero-noise opacity-[0.16] mix-blend-soft-light" />
          <div className="pointer-events-none absolute -left-20 top-20 h-72 w-72 rounded-full bg-esc-pink/12 blur-3xl animate-hero-aurora" />
          <div className="pointer-events-none absolute right-[-8rem] top-[8%] h-[28rem] w-[28rem] rounded-full bg-esc-pink/10 blur-3xl animate-hero-aurora-delayed" />
          <div className="pointer-events-none absolute bottom-[-8rem] left-[20%] h-[22rem] w-[22rem] rounded-full bg-black/5 blur-3xl animate-hero-aurora" />
          <div className="hero-spotlight pointer-events-none absolute inset-y-[-20%] left-1/2 hidden w-[36rem] -translate-x-1/2 opacity-80 md:block" />

          <div className="relative z-10 mx-auto flex min-h-[100svh] w-full max-w-7xl flex-col justify-between gap-14 lg:min-h-[960px]">
            <div className="flex flex-wrap items-center justify-between gap-4 border-b border-esc-border/80 pb-4 text-[11px] uppercase tracking-[0.24em] text-esc-muted">
              <div
                className="opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "60ms" }}
              >
                Live voting experience
              </div>
              <div
                className="opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "120ms" }}
              >
                Choose your favorites • Spend 20 points • Submit when ready
              </div>
            </div>

            <div className="grid items-center gap-14 lg:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)] lg:gap-10">
              <div className="relative z-10 max-w-4xl">
                <div
                  className="mb-6 inline-flex items-center gap-3 rounded-full border border-esc-border bg-white/80 px-4 py-2 text-[11px] uppercase tracking-[0.2em] text-esc-muted shadow-[0_16px_40px_rgba(17,17,17,0.06)] backdrop-blur-sm opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "140ms" }}
                >
                  <span className="h-2 w-2 rounded-full bg-esc-pink animate-stage-glow motion-reduce:animate-none" />
                  Tonight&apos;s viewer vote
                </div>

                <div className="space-y-1 sm:space-y-2">
                  {HERO_WORDS.map((word, index) => {
                    const barDelay = `${220 + index * 140}ms`;
                    const textDelay = `${300 + index * 140}ms`;

                    return (
                      <div key={word} className="hero-word-line">
                        <span
                          className="hero-word-bar motion-reduce:animate-none"
                          style={
                            heroAccentActive
                              ? {
                                  width: HERO_BAR_WIDTHS[index],
                                  animationDelay: barDelay
                                }
                              : {
                                  width: HERO_BAR_WIDTHS[index],
                                  animation: "none",
                                  opacity: 0
                                }
                          }
                        />

                        <h1 className="hero-display text-balance">
                          <span
                            className="hero-word-copy motion-reduce:animate-none"
                            style={{ animationDelay: textDelay }}
                          >
                            {word}
                          </span>

                          <span
                            aria-hidden="true"
                            className="hero-word-copy-overlay motion-reduce:animate-none"
                            style={
                              heroAccentActive
                                ? {
                                    width: HERO_BAR_WIDTHS[index],
                                    animationDelay: barDelay
                                  }
                                : {
                                    width: HERO_BAR_WIDTHS[index],
                                    animation: "none",
                                    opacity: 0
                                  }
                            }
                          >
                            <span
                              className="hero-word-copy-overlay-text motion-reduce:animate-none"
                              style={
                                heroAccentActive
                                  ? {
                                      animationDelay: textDelay
                                    }
                                  : {
                                      animation: "none",
                                      opacity: 0
                                    }
                              }
                            >
                              {word}
                            </span>
                          </span>
                        </h1>
                      </div>
                    );
                  })}
                </div>

                <p
                  className="mt-8 max-w-2xl text-base leading-7 text-esc-black-soft/80 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100 sm:text-lg"
                  style={{ animationDelay: "680ms" }}
                >
                  Pick the performances you love most, spread your 20 points however you want and
                  keep an eye on your current favorite before you submit.
                </p>

                <div
                  className="mt-8 flex flex-wrap gap-3 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "820ms" }}
                >
                  {HERO_CHIPS.map((chip) => (
                    <span key={chip} className="hero-chip">
                      {chip}
                    </span>
                  ))}
                </div>

                <div
                  className="mt-10 flex flex-wrap items-center gap-5 opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "900ms" }}
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

              <div className="relative mx-auto flex w-full max-w-[34rem] items-center justify-center">
                <div className="pointer-events-none absolute inset-10 rounded-full border border-esc-border/70" />
                <div className="pointer-events-none absolute inset-[18%] rounded-full border border-esc-border/60" />
                <div className="pointer-events-none absolute inset-[30%] rounded-full border border-esc-border/50" />

                <div
                  className="relative aspect-[0.94] w-full max-w-[31rem] opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                  style={{ animationDelay: "420ms" }}
                >
                  <div
                    className="absolute left-[4%] top-[10%] w-[62%] opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                    style={{ animationDelay: "520ms" }}
                  >
                    <div className="hero-card animate-hero-card-float motion-reduce:animate-none">
                      <div className="flex items-start justify-between gap-4">
                        <div>
                          <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Your budget</p>
                          <p className="mt-2 text-3xl font-bold text-esc-black">{TOTAL} pts</p>
                        </div>
                        <div className="rounded-full bg-esc-pink-soft px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.18em] text-esc-pink">
                          Live
                        </div>
                      </div>
                      <div className="mt-5 h-2 overflow-hidden rounded-full bg-esc-pink-soft">
                        <div
                          className="progress-shine h-full rounded-full bg-gradient-to-r from-esc-pink-dim via-esc-pink to-esc-pink-dim"
                          style={{ width: `${Math.max(16, (used / TOTAL) * 100)}%` }}
                        />
                      </div>
                      <div className="mt-4 flex items-center justify-between text-sm text-esc-muted">
                        <span>{remaining} points left</span>
                        <span>{used} used</span>
                      </div>
                    </div>
                  </div>

                  <div
                    className="absolute right-[2%] top-[29%] w-[54%] opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                    style={{ animationDelay: "680ms" }}
                  >
                    <div className="hero-card animate-hero-card-float-delayed motion-reduce:animate-none">
                      <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Live tonight</p>
                      <div className="mt-4">
                        <p className="text-base font-semibold text-esc-black">Voting is open</p>
                        <p className="mt-2 text-sm leading-6 text-esc-muted">
                          Choose your favorites below and adjust your points anytime.
                        </p>
                      </div>
                      <div className="mt-5 flex gap-2">
                        <span className="hero-mini-pill">All songs</span>
                        <span className="hero-mini-pill">Easy to edit</span>
                      </div>
                    </div>
                  </div>

                  <div
                    className="absolute bottom-[11%] left-[10%] w-[58%] opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                    style={{ animationDelay: "840ms" }}
                  >
                    <div className="hero-card animate-hero-card-float motion-reduce:animate-none">
                      <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Your current favorite</p>
                      {topSelection ? (
                        <div className="mt-4">
                          <p className="text-base font-semibold text-esc-black">{topSelection.countryName}</p>
                          <p className="mt-1 text-sm text-esc-muted">{topSelection.songName}</p>
                          <p className="mt-3 text-sm leading-6 text-esc-black-soft/75">
                            This entry currently leads your personal ranking with {topSelection.points}{" "}
                            {topSelection.points === 1 ? "point" : "points"}.
                          </p>
                        </div>
                      ) : (
                        <div className="mt-4 flex items-center gap-3">
                          <div className="flex h-11 w-11 items-center justify-center rounded-full border border-esc-border bg-white text-lg">
                            ✓
                          </div>
                          <div>
                            <p className="text-base font-semibold text-esc-black">Nothing selected yet</p>
                            <p className="text-sm text-esc-muted">
                              Start assigning points and your leader appears here.
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>

                  <div className="absolute right-[13%] top-[14%] h-28 w-28 overflow-hidden rounded-full border border-esc-border/70 bg-white/40 backdrop-blur-md animate-hero-aurora motion-reduce:animate-none">
                    {topSelection ? (
                      <img
                        src={flagUrl(topSelection.countryId)}
                        alt={topSelection.countryName}
                        className="h-full w-full object-cover scale-[10] -rotate-[8deg]"                      />
                    ) : (
                      <div className="h-full w-full bg-white/30" />
                    )}
                  </div>
                </div>
              </div>
            </div>

            <div className="flex flex-wrap items-end justify-between gap-6 border-t border-esc-border/80 pt-6">
              <div
                className="max-w-md opacity-0 animate-hero-fade-up motion-reduce:animate-none motion-reduce:opacity-100"
                style={{ animationDelay: "980ms" }}
              >
                <p className="text-[11px] uppercase tracking-[0.22em] text-esc-muted">Before you vote</p>
                <p className="mt-3 text-sm leading-6 text-esc-black-soft/75">
                  Review the songs, spread your points and keep track of the entry that currently leads your personal scoreboard.
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
        </div>

        <div
          id="voting-grid"
          className="relative px-4 pb-8 pt-4 sm:px-6 sm:pb-10 sm:pt-5 lg:px-8 lg:pb-12 lg:pt-6"
        >
          <div className="pointer-events-none absolute inset-x-0 top-[-5rem] h-40 bg-gradient-to-b from-esc-pink/16 via-esc-pink/8 to-white/0" />
          <div className="pointer-events-none absolute inset-0 vote-section-wash" />
          <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
          <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
          <div className="pointer-events-none absolute left-0 right-0 top-0 h-32 bg-gradient-to-b from-black/30 via-black/12 to-transparent" />
          <div className="pointer-events-none absolute left-[-22%] top-[4%] h-[64rem] w-[110rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.4)_0%,_rgba(255,4,144,0.24)_26%,_rgba(255,4,144,0.11)_48%,_rgba(255,4,144,0)_78%)] blur-[210px] opacity-95" />
          <div className="pointer-events-none absolute right-[-26%] top-[10%] h-[58rem] w-[96rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.31)_0%,_rgba(255,4,144,0.17)_28%,_rgba(255,4,144,0.075)_52%,_rgba(255,4,144,0)_80%)] blur-[200px] opacity-86" />
          <div className="pointer-events-none absolute left-[-18%] top-[26%] h-[72rem] w-[132rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.29)_0%,_rgba(255,4,144,0.145)_32%,_rgba(255,4,144,0.06)_56%,_rgba(255,4,144,0)_82%)] blur-[230px] opacity-84" />

          <div className="relative z-10 mx-auto max-w-7xl space-y-6">
            <div className="vote-panel rounded-[2rem] p-6 sm:p-8">
              <div className="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
                <div className="max-w-2xl space-y-3">
                  <p className="text-xs uppercase tracking-[0.16em] text-esc-pink/72">Live voting</p>
                  <h3 className="inline-flex rounded-full border border-esc-pink/16 bg-white/95 px-4 py-2 text-4xl font-bold leading-none text-esc-pink shadow-[0_12px_30px_rgba(255,4,144,0.08)]">
                    Cast Your Votes
                  </h3>
                  <p className="text-sm leading-7 text-esc-black-soft/75 sm:text-base">
                    Assign your points, keep an eye on your current leader and adjust everything until
                    your personal scoreboard feels right.
                  </p>
                </div>

                <div className="grid gap-3 sm:grid-cols-3">
                  <div className="vote-panel-soft rounded-2xl px-4 py-4">
                    <p className="text-[11px] uppercase tracking-[0.14em] text-esc-pink/72">Countries</p>
                    <p className="mt-2 text-2xl font-bold text-esc-black">{selectedSongs.length}</p>
                    <p className="mt-1 text-sm text-esc-muted">selected</p>
                  </div>

                  <div className="vote-panel-soft rounded-2xl px-4 py-4">
                    <p className="text-[11px] uppercase tracking-[0.14em] text-esc-pink/72">Points used</p>
                    <p className="mt-2 text-2xl font-bold text-esc-black">{used}</p>
                    <p className="mt-1 text-sm text-esc-muted">of {TOTAL}</p>
                  </div>

                  <div className="vote-panel-accent rounded-2xl px-4 py-4">
                    <p className="text-[11px] uppercase tracking-[0.14em] text-esc-pink/78">Top pick</p>
                    <p className="mt-2 truncate text-2xl font-bold text-esc-black">
                      {topSelection ? topSelection.countryName : "—"}
                    </p>
                    <p className="mt-1 text-sm text-esc-muted">
                      {topSelection ? `${topSelection.points} pts` : "No leader yet"}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div className="grid gap-6 xl:grid-cols-[320px_minmax(0,1fr)]">
              <aside className="space-y-4">
                <BudgetBar remaining={remaining} total={TOTAL} />

                <div className="vote-panel-soft rounded-[1.75rem] p-5">
                  <p className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">Current leader</p>
                  {topSelection ? (
                    <div className="mt-3 space-y-1">
                      <p className="text-2xl font-bold text-esc-black">{topSelection.countryName}</p>
                      <p className="text-sm text-esc-muted">{topSelection.songName}</p>
                      <p className="pt-2 text-sm leading-6 text-esc-black-soft/75">
                        This entry currently leads your personal ranking with {topSelection.points}{" "}
                        {topSelection.points === 1 ? "point" : "points"}.
                      </p>
                    </div>
                  ) : (
                    <p className="mt-3 text-sm leading-6 text-esc-black-soft/75">
                      As soon as you assign points, your current favorite will appear here.
                    </p>
                  )}
                </div>

                <div className="vote-panel-soft rounded-[1.75rem] p-5">
                  <p className="text-xs uppercase tracking-[0.14em] text-esc-pink/72">Quick note</p>
                  <p className="mt-3 text-sm leading-6 text-esc-black-soft/75">
                    You can change, remove and rebalance points anytime before submitting.
                  </p>
                </div>
              </aside>

              <div className="vote-panel rounded-[2rem] p-4 sm:p-5 lg:p-6">
                <div className="mb-5 flex flex-col gap-4 border-b border-esc-pink/12 pb-5 lg:flex-row lg:items-end lg:justify-between">
                  <div>
                    <p className="text-xs uppercase tracking-[0.16em] text-esc-pink/72">All entries</p>
                    <h4 className="mt-2 text-2xl font-bold text-esc-black sm:text-3xl">
                      Choose your countries
                    </h4>
                  </div>
                  <p className="max-w-md text-sm leading-6 text-esc-black-soft/75">
                    Use the buttons on each card to add or remove points.
                  </p>
                </div>

                <div className="grid gap-4 md:grid-cols-2 xl:gap-5">
                  {songs.map((song) => (
                    <CountryCard
                      key={song.songId}
                      song={song}
                      points={selection[song.songId] ?? 0}
                      onChange={changePoints}
                      disabled={votingLocked}
                    />
                  ))}
                </div>
              </div>
            </div>

            {votingLocked ? (
              <div className="sticky bottom-4 z-30 w-full sm:bottom-5">
                <div className="flex items-center justify-center rounded-xl border border-green-400/20 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(240,253,244,0.94))] px-4 py-3 shadow-[0_8px_28px_rgba(34,197,94,0.06)] backdrop-blur-md">
                  <span className="text-sm font-semibold text-green-700">
                    Votes submitted — {used} points spent
                  </span>
                </div>
              </div>
            ) : (
              <VoteBasket total={used} onSubmit={() => setOpenSubmit(true)} disabled={used === 0} />
            )}

            <SubmitModal
              open={openSubmit}
              totalPoints={used}
              onClose={() => setOpenSubmit(false)}
              onSubmit={submit}
            />
          </div>
        </div>
      </div>
    </section>
  );
};
