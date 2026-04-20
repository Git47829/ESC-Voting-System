import { useState } from "react";

import { api } from "../../api/client";
import { normalizeYoutubeUrl } from "../../utils/normalizeYoutubeUrl";

const formClass =
  "rounded-[1.9rem] border border-white/10 bg-[#161616] p-5 text-white shadow-[0_20px_48px_rgba(0,0,0,0.16)] sm:p-6";
const labelClass = "text-[11px] font-semibold uppercase tracking-[0.18em] text-white/48";
const inputClass =
  "mt-2 h-11 w-full rounded-2xl border border-white/10 bg-white/[0.06] px-4 text-sm text-white placeholder:text-white/32 focus:border-esc-pink focus:bg-white/[0.08]";
const buttonClass =
  "mt-2 inline-flex w-full items-center justify-center rounded-2xl border border-esc-pink bg-esc-pink px-4 py-3 text-sm font-semibold text-white transition-colors duration-200 hover:bg-esc-pink-dim";

export const AddEntryForms = ({ onDone }: { onDone: () => void }) => {
  const [countryId, setCountryId] = useState("");
  const [countryName, setCountryName] = useState("");
  const [artistFirstName, setArtistFirstName] = useState("");
  const [artistLastName, setArtistLastName] = useState("");
  const [songName, setSongName] = useState("");
  const [youtubeUrl, setYoutubeUrl] = useState("");

  return (
    <section className="grid gap-4 xl:grid-cols-3">
      <form
        className={formClass}
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddCountry(countryId, countryName).then(onDone);
        }}
      >
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step one</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add country</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Create the country record used by the artist and song entries.
        </p>

        <div className="mt-6 space-y-4">
          <label className="block">
            <span className={labelClass}>Country ID</span>
            <input
              className={inputClass}
              value={countryId}
              onChange={(e) => setCountryId(e.target.value.toUpperCase())}
              placeholder="ID"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Country name</span>
            <input
              className={inputClass}
              value={countryName}
              onChange={(e) => setCountryName(e.target.value)}
              placeholder="Name"
            />
          </label>
        </div>

        <button className={buttonClass}>Save country</button>
      </form>

      <form
        className={formClass}
        onSubmit={(e) => {
          e.preventDefault();
          void api.adminAddArtist(artistFirstName, artistLastName, countryId).then(onDone);
        }}
      >
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step two</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add artist</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Link an artist to the existing country entry before creating the song.
        </p>

        <div className="mt-6 space-y-4">
          <label className="block">
            <span className={labelClass}>First name</span>
            <input
              className={inputClass}
              value={artistFirstName}
              onChange={(e) => setArtistFirstName(e.target.value)}
              placeholder="First name"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Last name</span>
            <input
              className={inputClass}
              value={artistLastName}
              onChange={(e) => setArtistLastName(e.target.value)}
              placeholder="Last name"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Country ID</span>
            <input
              className={inputClass}
              value={countryId}
              onChange={(e) => setCountryId(e.target.value.toUpperCase())}
              placeholder="Country ID"
            />
          </label>
        </div>

        <button className={buttonClass}>Save artist</button>
      </form>

      <form
        className={formClass}
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
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step three</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add song</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Finish the lineup entry with title, artist and the optional performance video.
        </p>

        <div className="mt-6 space-y-4">
          <label className="block">
            <span className={labelClass}>Song name</span>
            <input
              className={inputClass}
              value={songName}
              onChange={(e) => setSongName(e.target.value)}
              placeholder="Song name"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Country ID</span>
            <input
              className={inputClass}
              value={countryId}
              onChange={(e) => setCountryId(e.target.value.toUpperCase())}
              placeholder="Country ID"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Artist first name</span>
            <input
              className={inputClass}
              value={artistFirstName}
              onChange={(e) => setArtistFirstName(e.target.value)}
              placeholder="Artist first name"
            />
          </label>

          <label className="block">
            <span className={labelClass}>Artist last name</span>
            <input
              className={inputClass}
              value={artistLastName}
              onChange={(e) => setArtistLastName(e.target.value)}
              placeholder="Artist last name"
            />
          </label>

          <label className="block">
            <span className={labelClass}>YouTube URL</span>
            <input
              className={inputClass}
              value={youtubeUrl}
              onChange={(e) => setYoutubeUrl(e.target.value)}
              placeholder="YouTube URL"
            />
          </label>
        </div>

        <button className={buttonClass}>Save song</button>
      </form>
    </section>
  );
};
