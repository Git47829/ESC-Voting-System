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

  useEffect(() => {
    void api.getSongs().then(setSongs);
  }, []);

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Jury panel</p>
        <h1 className="mt-2 text-4xl font-bold text-esc-black">Jury Voting</h1>
      </div>

      <PointsLegend />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {songs.map((song) => (
          <div key={song.songId} className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_30px_rgba(0,0,0,0.04)]">
            <JuryCard
              song={song}
              selected={selected[song.songId] ?? 12}
              onSelect={(value) =>
                setSelected((current) => ({ ...current, [song.songId]: value }))
              }
            />
            <button
              className="mt-3 rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim"
              onClick={() => {
                const points = selected[song.songId] ?? 12;
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
          </div>
        ))}
      </div>
    </section>
  );
};
