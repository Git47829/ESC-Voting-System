import { useState } from "react";

import { api } from "../../api/client";
import { normalizeYoutubeUrl } from "../../utils/normalizeYoutubeUrl";

export const AddEntryForms = ({ onDone }: { onDone: () => void }) => {
  const [countryId, setCountryId] = useState("");
  const [countryName, setCountryName] = useState("");
  const [artistFirstName, setArtistFirstName] = useState("");
  const [artistLastName, setArtistLastName] = useState("");
  const [songName, setSongName] = useState("");
  const [youtubeUrl, setYoutubeUrl] = useState("");

  return (
    <section className="grid gap-4 md:grid-cols-3">
      <form
        className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_28px_rgba(0,0,0,0.04)]"
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddCountry(countryId, countryName).then(onDone);
        }}
      >
        <h3 className="mb-3 text-base font-bold text-esc-black">Add Country</h3>
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="ID" />
        <input className="mb-3 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryName} onChange={(e) => setCountryName(e.target.value)} placeholder="Name" />
        <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim">Save</button>
      </form>

      <form
        className="rounded-[1.5rem] border border-esc-border bg-white/92 p-4 shadow-[0_12px_28px_rgba(0,0,0,0.04)]"
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddArtist(artistFirstName, artistLastName, countryId).then(onDone);
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
          void api
            .adminAddSong({
              songName,
              countryId,
              artistFirstName,
              artistLastName,
              youtubeUrl: normalizeYoutubeUrl(youtubeUrl)
            })
            .then(onDone);
        }}
      >
        <h3 className="mb-3 text-base font-bold text-esc-black">Add Song</h3>
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={songName} onChange={(e) => setSongName(e.target.value)} placeholder="Song name" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="Country ID" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={artistFirstName} onChange={(e) => setArtistFirstName(e.target.value)} placeholder="Artist first name" />
        <input className="mb-2 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={artistLastName} onChange={(e) => setArtistLastName(e.target.value)} placeholder="Artist last name" />
        <input className="mb-3 w-full rounded-xl border border-esc-border bg-white px-3 py-2 focus:border-esc-pink" value={youtubeUrl} onChange={(e) => setYoutubeUrl(e.target.value)} placeholder="YouTube URL" />
        <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 text-sm font-semibold text-white hover:bg-esc-pink-dim">Save</button>
      </form>
    </section>
  );
};
