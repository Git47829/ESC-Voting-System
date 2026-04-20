import { useEffect, useState } from "react";

import { api } from "../api/client";
import { JuryCard } from "../components/jury/JuryCard";
import { PointsLegend } from "../components/jury/PointsLegend";
import { useFlash } from "../context/FlashContext";
import type { Song } from "../types";

export const JuryPage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [selected, setSelected] = useState<Record<number, number>>({});
  const { addFlash } = useFlash();
  const allowedPoints = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12];

  useEffect(() => {
    void api.getSongs().then(setSongs);
  }, []);

  return (
    <section className="space-y-8 py-8 sm:space-y-10 sm:py-10">
      <header className="relative overflow-hidden rounded-[2.25rem] border border-white/10 bg-[#121212] px-6 py-7 text-white shadow-[0_28px_90px_rgba(0,0,0,0.2)] sm:px-8 sm:py-8">
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(135deg,rgba(255,4,144,0.14),rgba(255,4,144,0.03)_38%,rgba(255,255,255,0.02)_100%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,4,144,0.18),transparent_32%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:7rem_7rem] opacity-20" />

        <div className="relative z-10 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)] xl:items-end">
          <div className="max-w-3xl">
            <p className="text-[11px] uppercase tracking-[0.24em] text-white/55">Jury panel</p>
            <h1 className="mt-3 text-4xl font-bold tracking-[-0.04em] text-white sm:text-5xl">
              Jury voting
            </h1>
            <p className="mt-4 max-w-2xl text-sm leading-7 text-white/68 sm:text-base">
              Assign one of the existing jury point values to each song and submit each vote
              individually, with the same data flow as before.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Songs loaded</p>
              <p className="mt-3 text-3xl font-bold text-white">{songs.length}</p>
            </div>
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Allowed values</p>
              <p className="mt-3 text-lg font-semibold text-white">{allowedPoints.join(" • ")}</p>
            </div>
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Default pick</p>
              <p className="mt-3 text-3xl font-bold text-white">12</p>
            </div>
          </div>
        </div>
      </header>

      <PointsLegend />

      <section className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Voting grid</p>
            <h2 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-esc-black">
              Score each entry
            </h2>
          </div>
          <p className="max-w-lg text-sm leading-6 text-esc-muted">
            Each card keeps the existing select-and-submit flow, just with stronger hierarchy and
            easier scanning.
          </p>
        </div>

        <div className="grid gap-4 xl:grid-cols-2">
          {songs.map((song) => {
            const points = selected[song.songId] ?? 12;

            return (
              <article
                key={song.songId}
                className="rounded-[2rem] border border-white/10 bg-[#161616] p-5 text-white shadow-[0_20px_54px_rgba(0,0,0,0.16)] sm:p-6"
              >
                <div className="flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">
                      {song.countryId}
                    </p>
                    <h3 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-white">
                      {song.countryName}
                    </h3>
                    <p className="mt-2 text-lg text-white/82">{song.songName}</p>
                    <p className="mt-1 text-sm text-white/50">
                      {song.artistFirstName} {song.artistLastName}
                    </p>
                  </div>

                  <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] px-5 py-4 text-center">
                    <p className="text-[11px] uppercase tracking-[0.18em] text-white/45">
                      Selected
                    </p>
                    <p className="mt-2 text-3xl font-bold text-white">{points}</p>
                  </div>
                </div>

                <div className="mt-6 rounded-[1.5rem] border border-white/10 bg-black/20 p-4">
                  <JuryCard
                    song={song}
                    selected={points}
                    onSelect={(value) =>
                      setSelected((current) => ({ ...current, [song.songId]: value }))
                    }
                  />
                </div>

                <button
                  className="mt-5 inline-flex w-full items-center justify-center rounded-2xl border border-esc-pink bg-esc-pink px-4 py-3 text-sm font-semibold text-white transition-colors duration-200 hover:bg-esc-pink-dim"
                  onClick={() => {
                    void api
                      .juryVote(song.songId, points)
                      .then(() => addFlash(`Jury vote sent for ${song.countryName}`, "success"))
                      .catch((error: unknown) =>
                        addFlash(error instanceof Error ? error.message : "Vote failed", "error")
                      );
                  }}
                >
                  Submit jury vote
                </button>
              </article>
            );
          })}
        </div>
      </section>
    </section>
  );
};
