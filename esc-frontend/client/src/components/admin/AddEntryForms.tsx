import { useState } from "react";

import { adminAddArtist, adminAddCountry, adminAddSong } from "../../services/admin-api";
import type { FlashMessage } from "../../types";
import { normalizeYoutubeUrl } from "../../utils/normalizeYoutubeUrl";

const formClass =
  "rounded-[1.9rem] border border-white/10 bg-[#161616] p-5 text-white shadow-[0_20px_48px_rgba(0,0,0,0.16)] sm:p-6";
const labelClass = "text-[11px] font-semibold uppercase tracking-[0.18em] text-white/48";
const inputClass =
  "mt-2 h-11 w-full rounded-2xl border border-white/10 bg-white/[0.06] px-4 text-sm text-white placeholder:text-white/32 focus:border-esc-pink focus:bg-white/[0.08]";
const buttonClass =
  "mt-2 inline-flex w-full items-center justify-center rounded-2xl border border-esc-pink bg-esc-pink px-4 py-3 text-sm font-semibold text-white transition-colors duration-200 hover:bg-esc-pink-dim";

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
    <section className="grid gap-4 xl:grid-cols-3">
      <form
        className={formClass}
        onSubmit={(e) => {
          e.preventDefault();
          void run(
            () => adminAddCountry(countryId, countryName, Number.parseInt(countryPot, 10) || 1),
            `Country '${countryName}' added`,
            () => { setCountryId(""); setCountryName(""); setCountryPot("1"); }
          );
        }}
      >
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step one</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add country</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Create the country record and assign its draw pot.
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

          <label className="block">
            <span className={labelClass}>Pot</span>
            <input
              className={inputClass}
              type="number"
              min={1}
              max={99}
              value={countryPot}
              onChange={(e) => setCountryPot(e.target.value)}
              placeholder="Pot"
            />
          </label>
        </div>

        <button className={buttonClass}>Save country</button>
      </form>

      <form
        className={formClass}
        onSubmit={(e) => {
          e.preventDefault();
          void run(
            () => adminAddArtist(artistFirstName, artistLastName, countryId),
            `Artist '${artistFirstName} ${artistLastName}' added`,
            () => { setArtistFirstName(""); setArtistLastName(""); }
          );
        }}
      >
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step two</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add artist</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Link the artist to the country that will perform the entry.
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
          void run(
            () => adminAddSong({
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
        <p className="text-[11px] uppercase tracking-[0.2em] text-white/45">Step three</p>
        <h3 className="mt-2 text-2xl font-bold tracking-[-0.03em] text-white">Add song</h3>
        <p className="mt-2 text-sm leading-6 text-white/58">
          Finish the lineup entry with song, artist ID and video source.
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
            <span className={labelClass}>Artist ID</span>
            <input
              className={inputClass}
              type="number"
              min={1}
              value={songArtistId}
              onChange={(e) => setSongArtistId(e.target.value)}
              placeholder="Artist ID"
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
