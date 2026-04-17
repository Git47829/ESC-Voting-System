import { useEffect, useState } from "react";

import { api } from "../api/client";
import { AddEntryForms } from "../components/admin/AddEntryForms";
import { EntryTable } from "../components/admin/EntryTable";
import { ConfirmModal } from "../components/ui/ConfirmModal";
import { useFlash } from "../context/FlashContext";
import type { Song } from "../types";

export const AdminPage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [confirmReset, setConfirmReset] = useState(false);
  const { addFlash } = useFlash();

  const load = async () => {
    try {
      const nextSongs = await api.getSongs();
      setSongs(nextSongs);
    } catch (err) {
      addFlash(err instanceof Error ? err.message : "Failed to load songs", "error");
    }
  };

  const run = async (action: () => Promise<unknown>, successMsg: string) => {
    try {
      await action();
      addFlash(successMsg, "success");
      void load();
    } catch (err) {
      addFlash(err instanceof Error ? err.message : "Action failed", "error");
    }
  };

  useEffect(() => {
    void load();
  }, []);

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Admin control</p>
        <h1 className="mt-2 text-4xl font-bold text-esc-black">Admin Dashboard</h1>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <div className="flex flex-wrap gap-2.5">
          <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim" onClick={() => void run(() => api.adminOpen(), "Voting opened")}>Open voting</button>
          <button className="rounded-xl border border-esc-border px-3 py-2 text-sm text-esc-black hover:border-esc-pink hover:text-esc-pink" onClick={() => void run(() => api.adminClose(), "Voting closed")}>Close voting</button>
          <button className="rounded-xl border border-red-500 px-3 py-2 text-sm text-red-500 hover:bg-red-50" onClick={() => setConfirmReset(true)}>Reset votes</button>
          <button className="rounded-xl border border-esc-border px-3 py-2 text-sm text-esc-black hover:border-esc-pink hover:text-esc-pink" onClick={() => void run(() => api.adminStartContest(), "Contest started")}>Start contest</button>
          <button className="rounded-xl border border-esc-border px-3 py-2 text-sm text-esc-black hover:border-esc-pink hover:text-esc-pink" onClick={() => void run(() => api.adminAdvanceContest(), "Advanced to next song")}>Next song</button>
        </div>
      </div>

      <AddEntryForms onDone={load} onFlash={addFlash} />
      <EntryTable songs={songs} />
      <ConfirmModal
        open={confirmReset}
        title="Reset all votes"
        text="This action clears public and jury votes."
        onClose={() => setConfirmReset(false)}
        onConfirm={() => {
          void run(() => api.adminDeleteVotes(), "All votes reset").then(() =>
            setConfirmReset(false)
          );
        }}
      />
    </section>
  );
};
