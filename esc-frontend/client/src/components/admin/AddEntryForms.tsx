import { useState } from "react";

import { api } from "../../api/client";
import type { FlashMessage } from "../../types";
import { normalizeYoutubeUrl } from "../../utils/normalizeYoutubeUrl";

export const AddEntryForms = ({
  onDone,
  onFlash
}: {
  onDone: () => void;
  onFlash: (message: string, category: FlashMessage["category"]) => void;
}) => {
  const [countryId, setCountryId] = useState("");
  const [countryName, setCountryName] = useState("");
  const [countryPot, setCountryPot] = useState("1");
  const [artistFirstName, setArtistFirstName] = useState("");
  const [artistLastName, setArtistLastName] = useState("");
  const [songName, setSongName] = useState("");
  const [songArtistId, setSongArtistId] = useState("");
  const [youtubeUrl, setYoutubeUrl] = useState("");

  const run = async (action: () => Promise<unknown>, successMsg: string, resetFields?: () => void) => {
    try {
      await action();
      onFlash(successMsg, "success");
      resetFields?.();
      onDone();
    } catch (err) {
      onFlash(err instanceof Error ? err.message : "Action failed", "error");
    }
  };

  return (
    <section className="grid gap-4 md:grid-cols-3">
      <form
        className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_28px_rgba(0,0,0,0.04)]"
        onSubmit={(e) => {
          e.preventDefault();
          void run(
            () => api.adminAddCountry(countryId, countryName, Number.parseInt(countryPot, 10) || 1),
            `Country '${countryName}' added`,
            () => { setCountryId(""); setCountryName(""); setCountryPot("1"); }
          );
        }}
      >
        <h3 className="mb-3 text-base font-bold text-esc-black">Add Country</h3>
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="ID" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryName} onChange={(e) => setCountryName(e.target.value)} placeholder="Name" />
        <input className="mb-3 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" type="number" min={1} max={99} value={countryPot} onChange={(e) => setCountryPot(e.target.value)} placeholder="Pot" />
        <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim">Save</button>
      </form>

      <form
        className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_28px_rgba(0,0,0,0.04)]"
        onSubmit={(e) => {
          e.preventDefault();
          void run(
            () => api.adminAddArtist(artistFirstName, artistLastName, countryId),
            `Artist '${artistFirstName} ${artistLastName}' added`,
            () => { setArtistFirstName(""); setArtistLastName(""); }
          );
        }}
      >
        <h3 className="mb-3 text-base font-bold text-esc-black">Add Artist</h3>
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={artistFirstName} onChange={(e) => setArtistFirstName(e.target.value)} placeholder="First name" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={artistLastName} onChange={(e) => setArtistLastName(e.target.value)} placeholder="Last name" />
        <input className="mb-3 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="Country ID" />
        <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim">Save</button>
      </form>

      <form
        className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_28px_rgba(0,0,0,0.04)]"
        onSubmit={(e) => {
          e.preventDefault();
          void run(
            () => api.adminAddSong({
              songName,
              countryId,
              artistId: Number.parseInt(songArtistId, 10),
              youtubeUrl: normalizeYoutubeUrl(youtubeUrl)
            }),
            `Song '${songName}' added`,
            () => { setSongName(""); setSongArtistId(""); setYoutubeUrl(""); }
          );
        }}
      >
        <h3 className="mb-3 text-base font-bold text-esc-black">Add Song</h3>
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={songName} onChange={(e) => setSongName(e.target.value)} placeholder="Song name" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="Country ID" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" type="number" min={1} value={songArtistId} onChange={(e) => setSongArtistId(e.target.value)} placeholder="Artist ID" />
        <input className="mb-3 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={youtubeUrl} onChange={(e) => setYoutubeUrl(e.target.value)} placeholder="YouTube URL" />
        <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim">Save</button>
      </form>
    </section>
  );
};
