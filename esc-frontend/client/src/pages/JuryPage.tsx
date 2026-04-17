import { useEffect, useState } from "react";

import { api } from "../api/client";
import { JuryCard } from "../components/jury/JuryCard";
import { PointsLegend } from "../components/jury/PointsLegend";
import { useFlash } from "../context/FlashContext";
import type { Song } from "../types";

export const JuryPage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [selected, setSelected] = useState<Record<number, number>>({});
  const [submittedVotes, setSubmittedVotes] = useState<Record<number, number>>({});
  const [submittingSongs, setSubmittingSongs] = useState<Record<number, boolean>>({});
  const { addFlash } = useFlash();

  useEffect(() => {
    void Promise.all([api.getSongs(), api.getJuryVoteState()])
      .then(([fetchedSongs, voteState]) => {
        setSongs(fetchedSongs);
        setSubmittedVotes(voteState.votesCast);
      })
      .catch((error: unknown) => {
        addFlash(error instanceof Error ? error.message : "Failed to load jury votes", "error");
      });
  }, [addFlash]);

  const usedPointValues = new Set(Object.values(submittedVotes));

  useEffect(() => {
    setSelected((current) => {
      const next = { ...current };
      let changed = false;
      for (const [songIdString, points] of Object.entries(next)) {
        const songId = Number(songIdString);
        if (submittedVotes[songId] === undefined && usedPointValues.has(points)) {
          delete next[songId];
          changed = true;
        }
      }
      return changed ? next : current;
    });
  }, [submittedVotes]);

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Jury panel</p>
        <h1 className="mt-2 text-4xl font-bold text-esc-black">Jury Voting</h1>
      </div>

      <PointsLegend />

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {songs.map((song) => (
          <div key={song.songId} className={`rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_30px_rgba(0,0,0,0.04)] ${submittedVotes[song.songId] !== undefined ? "opacity-80" : ""}`}>
            <p className="mb-2 text-xs font-semibold uppercase tracking-[0.12em] text-esc-muted">
              {submittedVotes[song.songId] !== undefined ? "Vote submitted" : "Ready to vote"}
            </p>
            <JuryCard
              song={song}
              selected={submittedVotes[song.songId] ?? selected[song.songId]}
              disabled={submittedVotes[song.songId] !== undefined}
              usedPointValues={usedPointValues}
              onSelect={(value) =>
                setSelected((current) => ({ ...current, [song.songId]: value }))
              }
            />
            <button
              className="mt-3 rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim disabled:cursor-not-allowed disabled:opacity-50"
              disabled={
                submittedVotes[song.songId] !== undefined ||
                submittingSongs[song.songId] ||
                selected[song.songId] === undefined ||
                (selected[song.songId] !== undefined && usedPointValues.has(selected[song.songId]))
              }
              onClick={() => {
                const points = selected[song.songId];
                if (points === undefined) {
                  addFlash("Select points before submitting.", "error");
                  return;
                }
                setSubmittingSongs((current) => ({ ...current, [song.songId]: true }));
                void api
                  .juryVote(song.songId, points)
                  .then(() => {
                    setSubmittedVotes((current) => ({ ...current, [song.songId]: points }));
                    addFlash(`Jury vote sent for ${song.countryName}`, "success");
                  })
                  .catch((error: unknown) =>
                    addFlash(error instanceof Error ? error.message : "Vote failed", "error")
                  )
                  .finally(() => {
                    setSubmittingSongs((current) => ({ ...current, [song.songId]: false }));
                  });
              }}
            >
              {submittedVotes[song.songId] !== undefined ? "Jury vote submitted" : submittingSongs[song.songId] ? "Submitting..." : "Submit jury vote"}
            </button>
          </div>
        ))}
      </div>
    </section>
  );
};
