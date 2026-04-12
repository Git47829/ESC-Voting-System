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
    <section className="space-y-4">
      <h1 className="text-3xl font-bold">Jury Voting</h1>
      <PointsLegend />
      <div className="grid gap-3 md:grid-cols-2">
        {songs.map((song) => (
          <div key={song.songId} className="space-y-2">
            <JuryCard song={song} selected={selected[song.songId] ?? 12} onSelect={(value) => setSelected((current) => ({ ...current, [song.songId]: value }))} />
            <button
              className="border border-esc-yellow px-3 py-1 text-esc-yellow"
              onClick={() => {
                const points = selected[song.songId] ?? 12;
                void api
                  .juryVote(song.songId, points)
                  .then(() => addFlash(`Jury vote sent for ${song.countryName}`, "success"))
                  .catch((error: unknown) => addFlash(error instanceof Error ? error.message : "Vote failed", "error"));
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

