import { useEffect, useState } from "react";

import { getSongs } from "../services/songs-api";
import {
  adminAdvanceContest,
  adminClose,
  adminDeleteVotes,
  adminOpen,
  adminStartContest,
} from "../services/admin-api";
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
      const nextSongs = await getSongs();
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

  const votingOpen = songs[0]?.votingIsOpen ?? false;
  const totalVotes = songs.reduce((sum, song) => sum + song.totalVotes, 0);

  useEffect(() => {
    void load();
  }, []);

  return (
    <section className="space-y-8 py-8 sm:space-y-10 sm:py-10">
      <header className="relative overflow-hidden rounded-[2.25rem] border border-white/10 bg-[#121212] px-6 py-7 text-white shadow-[0_28px_90px_rgba(0,0,0,0.2)] sm:px-8 sm:py-8">
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(135deg,rgba(255,4,144,0.16),rgba(255,4,144,0.02)_36%,rgba(255,255,255,0.02)_100%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(255,4,144,0.22),transparent_34%)]" />
        <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(255,255,255,0.04)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.04)_1px,transparent_1px)] bg-[size:7rem_7rem] opacity-20" />

        <div className="relative z-10 grid gap-6 xl:grid-cols-[minmax(0,1.2fr)_minmax(320px,0.8fr)] xl:items-end">
          <div className="max-w-3xl">
            <p className="text-[11px] uppercase tracking-[0.24em] text-white/55">Admin control</p>
            <h1 className="mt-3 text-4xl font-bold tracking-[-0.04em] text-white sm:text-5xl">
              Backstage dashboard
            </h1>
            <p className="mt-4 max-w-2xl text-sm leading-7 text-white/68 sm:text-base">
              Manage the live flow, update the lineup and keep the contest data tidy from one
              focused control surface.
            </p>
          </div>

          <div className="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Entries loaded</p>
              <p className="mt-3 text-3xl font-bold text-white">{songs.length}</p>
            </div>
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Voting status</p>
              <p className="mt-3 text-lg font-semibold text-white">
                {votingOpen ? "Open for votes" : "Currently closed"}
              </p>
            </div>
            <div className="rounded-[1.5rem] border border-white/10 bg-white/[0.06] p-4 backdrop-blur-sm">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Votes stored</p>
              <p className="mt-3 text-3xl font-bold text-white">{totalVotes}</p>
            </div>
          </div>
        </div>
      </header>

      <section className="grid gap-4 xl:grid-cols-[minmax(0,1.2fr)_minmax(280px,0.8fr)]">
        <div className="rounded-[2rem] border border-esc-border bg-white/95 p-6 shadow-[0_20px_48px_rgba(0,0,0,0.06)] sm:p-7">
          <div className="flex flex-col gap-3 border-b border-esc-border pb-5 sm:flex-row sm:items-end sm:justify-between">
            <div>
              <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Show controls</p>
              <h2 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-esc-black">
                Contest actions
              </h2>
            </div>
            <p className="max-w-md text-sm leading-6 text-esc-muted">
              The existing controls are grouped by live-show responsibility for faster scanning.
            </p>
          </div>

          <div className="mt-6 grid gap-3 sm:grid-cols-2">
            <button
              className="flex min-h-[6.5rem] flex-col items-start justify-between rounded-[1.5rem] border border-esc-pink bg-esc-pink px-5 py-4 text-left text-white shadow-[0_20px_36px_rgba(255,4,144,0.22)] transition-colors duration-200 hover:bg-esc-pink-dim"
              onClick={() => void run(() => adminOpen(), "Voting opened")}
            >
              <span className="text-[11px] uppercase tracking-[0.18em] text-white/70">Voting</span>
              <span>
                <span className="block text-lg font-semibold">Open voting</span>
                <span className="mt-1 block text-sm text-white/80">
                  Allow public voting to begin.
                </span>
              </span>
            </button>

            <button
              className="flex min-h-[6.5rem] flex-col items-start justify-between rounded-[1.5rem] border border-esc-border bg-white px-5 py-4 text-left text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
              onClick={() => void run(() => adminClose(), "Voting closed")}
            >
              <span className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Voting</span>
              <span>
                <span className="block text-lg font-semibold">Close voting</span>
                <span className="mt-1 block text-sm text-esc-muted">
                  Stop incoming public votes.
                </span>
              </span>
            </button>

            <button
              className="flex min-h-[6.5rem] flex-col items-start justify-between rounded-[1.5rem] border border-esc-border bg-white px-5 py-4 text-left text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
              onClick={() => void run(() => adminStartContest(), "Contest started")}
            >
              <span className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Contest</span>
              <span>
                <span className="block text-lg font-semibold">Start contest</span>
                <span className="mt-1 block text-sm text-esc-muted">
                  Initialize the running order.
                </span>
              </span>
            </button>

            <button
              className="flex min-h-[6.5rem] flex-col items-start justify-between rounded-[1.5rem] border border-esc-border bg-white px-5 py-4 text-left text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
              onClick={() => void run(() => adminAdvanceContest(), "Advanced to next song")}
            >
              <span className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Contest</span>
              <span>
                <span className="block text-lg font-semibold">Next song</span>
                <span className="mt-1 block text-sm text-esc-muted">
                  Advance to the next entry.
                </span>
              </span>
            </button>

            <button
              className="flex min-h-[6.5rem] flex-col items-start justify-between rounded-[1.5rem] border border-red-300 bg-red-50 px-5 py-4 text-left text-red-600 transition-colors duration-200 hover:bg-red-100 sm:col-span-2"
              onClick={() => setConfirmReset(true)}
            >
              <span className="text-[11px] uppercase tracking-[0.18em] text-red-400">Maintenance</span>
              <span>
                <span className="block text-lg font-semibold">Reset votes</span>
                <span className="mt-1 block text-sm text-red-500/90">
                  Clear public and jury vote totals.
                </span>
              </span>
            </button>
          </div>
        </div>

        <aside className="rounded-[2rem] border border-white/10 bg-[#161616] p-6 text-white shadow-[0_20px_54px_rgba(0,0,0,0.16)] sm:p-7">
          <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Control flow</p>
          <h2 className="mt-2 text-2xl font-bold tracking-[-0.04em] text-white">
            During the live run
          </h2>
          <div className="mt-6 space-y-5">
            {[
              ["01", "Prepare entries", "Add countries, artists and songs before the show begins."],
              ["02", "Start the contest", "Launch the running order once the lineup is ready."],
              ["03", "Manage voting", "Open or close public voting as the broadcast requires."],
              ["04", "Advance the stage", "Move the contest forward song by song when needed."]
            ].map(([step, title, text]) => (
              <div key={step} className="flex gap-4">
                <span className="mt-0.5 inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-white/10 bg-white/[0.06] text-[11px] font-semibold tracking-[0.18em] text-white/70">
                  {step}
                </span>
                <div>
                  <p className="text-sm font-semibold text-white">{title}</p>
                  <p className="mt-1 text-sm leading-6 text-white/58">{text}</p>
                </div>
              </div>
            ))}
          </div>
        </aside>
      </section>

      <section className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">
              Lineup management
            </p>
            <h2 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-esc-black">
              Add countries, artists and songs
            </h2>
          </div>
          <p className="max-w-lg text-sm leading-6 text-esc-muted">
            The forms keep the current add-entry flow intact, but present each step more clearly.
          </p>
        </div>
        <AddEntryForms onDone={load} onFlash={addFlash} />
      </section>

      <section className="space-y-4">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
          <div>
            <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Current lineup</p>
            <h2 className="mt-2 text-3xl font-bold tracking-[-0.04em] text-esc-black">
              Contest entries
            </h2>
          </div>
          <p className="text-sm text-esc-muted">
            {songs.length} {songs.length === 1 ? "entry" : "entries"} currently available in the
            table.
          </p>
        </div>
        <EntryTable songs={songs} />
      </section>

      <ConfirmModal
        open={confirmReset}
        title="Reset all votes"
        text="This action clears public and jury votes."
        onClose={() => setConfirmReset(false)}
        onConfirm={() => {
          void run(() => adminDeleteVotes(), "All votes reset").then(() =>
            setConfirmReset(false)
          );
        }}
      />
    </section>
  );
};
