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
      <h3 className="text-base font-bold text-esc-black">{song.countryName} - {song.songName}</h3>
      <div className="mt-3 flex flex-wrap gap-2">
        {[1, 2, 3, 4, 5, 6, 7, 8, 10, 12].map((points) => (
          <button
            key={points}
            className={`rounded-lg border px-2.5 py-1.5 text-sm transition-colors ${selected === points ? "border-esc-pink bg-esc-pink text-white" : "border-esc-border bg-white text-esc-black hover:border-esc-pink hover:text-esc-pink"}`}
            onClick={() => onSelect(points)}
          >
            {points}
          </button>
        ))}
      </div>
    </article>
  );
};
