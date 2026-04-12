import { useEffect, useState } from "react";

import { api } from "../api/client";
import { AddEntryForms } from "../components/admin/AddEntryForms";
import { EntryTable } from "../components/admin/EntryTable";
import { ConfirmModal } from "../components/ui/ConfirmModal";
import type { Song } from "../types";

export const AdminPage = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [confirmReset, setConfirmReset] = useState(false);

  const load = () => void api.getSongs().then(setSongs);

  useEffect(() => {
    load();
  }, []);

  return (
    <section className="space-y-4">
      <h1 className="text-3xl font-bold">Admin Dashboard</h1>
      <div className="flex flex-wrap gap-2">
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow" onClick={() => void api.adminOpen().then(load)}>Open voting</button>
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow" onClick={() => void api.adminClose().then(load)}>Close voting</button>
        <button className="border border-red-500 px-3 py-1 text-red-400" onClick={() => setConfirmReset(true)}>Reset votes</button>
        <button className="border border-esc-muted px-3 py-1" onClick={() => void api.adminStartContest().then(load)}>Start contest</button>
        <button className="border border-esc-muted px-3 py-1" onClick={() => void api.adminAdvanceContest().then(load)}>Next song</button>
      </div>
      <AddEntryForms onDone={load} />
      <EntryTable songs={songs} />
      <ConfirmModal
        open={confirmReset}
        title="Reset all votes"
        text="This action clears public and jury votes."
        onClose={() => setConfirmReset(false)}
        onConfirm={() => {
          void api.adminDeleteVotes().then(() => {
            setConfirmReset(false);
            load();
          });
        }}
      />
    </section>
  );
};

