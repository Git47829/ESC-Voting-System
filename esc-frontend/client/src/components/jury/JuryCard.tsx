import type { Song } from "../../types";

export const JuryCard = ({
  song,
  selected,
  onSelect
}: {
  song: Song;
  selected: number;
  onSelect: (value: number) => void;
}) => {
  return (
    <article>
      <div className="flex flex-col gap-1 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <p className="text-[11px] uppercase tracking-[0.18em] text-white/45">Available points</p>
          <h3 className="mt-2 text-base font-semibold text-white">{song.countryName} - {song.songName}</h3>
        </div>
        <p className="text-xs uppercase tracking-[0.16em] text-white/35">
          Tap one value and submit below
        </p>
      </div>

      <div className="mt-4 flex flex-wrap gap-2.5">
        {[1, 2, 3, 4, 5, 6, 7, 8, 10, 12].map((points) => (
          <button
            key={points}
            type="button"
            className={`inline-flex min-w-[3rem] items-center justify-center rounded-2xl border px-3.5 py-2.5 text-sm font-semibold transition-colors duration-200 ${
              selected === points
                ? "border-esc-pink bg-esc-pink text-white shadow-[0_10px_24px_rgba(255,4,144,0.18)]"
                : "border-white/10 bg-white/[0.06] text-white/82 hover:border-esc-pink hover:text-white"
            }`}
            onClick={() => onSelect(points)}
          >
            {points}
          </button>
        ))}
      </div>
    </article>
  );
};
