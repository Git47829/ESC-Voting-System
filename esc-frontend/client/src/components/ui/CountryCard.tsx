import { flagUrl } from "../../utils/flagUrl";
import type { Song } from "../../types";

interface CountryCardProps {
  song: Song;
  points: number;
  onChange: (songId: number, delta: number) => void;
}

export const CountryCard = ({ song, points, onChange }: CountryCardProps) => {
  const isActive = points > 0;

  return (
    <article
      className={`group rounded-2xl border p-5 transition-all duration-300 ${
        isActive
          ? "border-esc-pink/30 bg-[linear-gradient(180deg,rgba(255,247,251,0.98),rgba(255,228,241,0.93))] shadow-[0_12px_30px_rgba(255,4,144,0.08)]"
          : "border-esc-pink/12 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(255,246,251,0.95))] shadow-[0_8px_20px_rgba(255,4,144,0.03)] hover:border-esc-pink/22 hover:shadow-[0_12px_30px_rgba(255,4,144,0.06)]"
      }`}
    >
      <div className="mb-4 flex items-start justify-between gap-3">
        <div className="flex items-center gap-3">
          <img
            src={flagUrl(song.countryId)}
            alt={song.countryName}
            className="h-8 w-12 rounded-md border border-esc-pink/10 object-cover"
          />
          <div>
            <h3 className="text-lg font-semibold text-esc-black">{song.countryName}</h3>
            <p className="text-sm text-esc-muted">{song.songName}</p>
          </div>
        </div>
        <span
          className={`rounded-full border px-2.5 py-1 text-xs font-semibold tracking-wide ${
            isActive
              ? "border-esc-pink bg-esc-pink text-white"
              : "border-esc-pink/12 bg-white text-esc-black-soft"
          }`}
        >
          {points} pts
        </span>
      </div>
      <div className="flex items-center gap-2.5">
        <button
          aria-label={`decrease points for ${song.countryName}`}
          className="rounded-xl border border-esc-pink/12 px-3.5 py-1.5 text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
          onClick={() => onChange(song.songId, -1)}
        >
          -
        </button>
        <button
          aria-label={`increase points for ${song.countryName}`}
          className="rounded-xl border border-esc-pink/12 px-3.5 py-1.5 text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
          onClick={() => onChange(song.songId, 1)}
        >
          +
        </button>
      </div>
    </article>
  );
};