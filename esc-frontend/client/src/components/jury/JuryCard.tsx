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
    <article className="border border-esc-muted p-3">
      <h3 className="font-bold">{song.countryName} - {song.songName}</h3>
      <div className="mt-2 flex flex-wrap gap-2">
        {[1, 2, 3, 4, 5, 6, 7, 8, 10, 12].map((points) => (
          <button
            key={points}
            className={`border px-2 py-1 ${selected === points ? "border-esc-yellow text-esc-yellow" : "border-esc-muted"}`}
            onClick={() => onSelect(points)}
          >
            {points}
          </button>
        ))}
      </div>
    </article>
  );
};

