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
        className="border border-esc-muted p-3"
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddCountry(countryId, countryName).then(onDone);
        }}
      >
        <h3 className="mb-2 font-bold">Add Country</h3>
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="ID" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={countryName} onChange={(e) => setCountryName(e.target.value)} placeholder="Name" />
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow">Save</button>
      </form>

      <form
        className="border border-esc-muted p-3"
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddArtist(artistFirstName, artistLastName, countryId).then(onDone);
        }}
      >
        <h3 className="mb-2 font-bold">Add Artist</h3>
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={artistFirstName} onChange={(e) => setArtistFirstName(e.target.value)} placeholder="First name" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={artistLastName} onChange={(e) => setArtistLastName(e.target.value)} placeholder="Last name" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="Country ID" />
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow">Save</button>
      </form>

      <form
        className="border border-esc-muted p-3"
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
        <h3 className="mb-2 font-bold">Add Song</h3>
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={songName} onChange={(e) => setSongName(e.target.value)} placeholder="Song name" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={countryId} onChange={(e) => setCountryId(e.target.value.toUpperCase())} placeholder="Country ID" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={artistFirstName} onChange={(e) => setArtistFirstName(e.target.value)} placeholder="Artist first name" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={artistLastName} onChange={(e) => setArtistLastName(e.target.value)} placeholder="Artist last name" />
        <input className="mb-2 w-full border border-esc-muted bg-transparent px-2 py-1" value={youtubeUrl} onChange={(e) => setYoutubeUrl(e.target.value)} placeholder="YouTube URL" />
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow">Save</button>
      </form>
    </section>
  );
};

