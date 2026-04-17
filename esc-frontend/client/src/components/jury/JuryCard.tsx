import type { Song } from "../../types";

export const JuryCard = ({
  song,
  selected,
  onSelect,
  disabled = false,
  usedPointValues
}: {
  song: Song;
  selected?: number;
  onSelect: (value: number) => void;
  disabled?: boolean;
  usedPointValues: Set<number>;
}) => {
  return (
    <article>
      <h3 className="text-base font-bold text-esc-black">{song.countryName} - {song.songName}</h3>
      <div className="mt-3 flex flex-wrap gap-2">
        {[1, 2, 3, 4, 5, 6, 7, 8, 10, 12].map((points) => {
          const isUsedElsewhere = usedPointValues.has(points) && selected !== points;
          const isDisabled = disabled || isUsedElsewhere;
          return (
          <button
            key={points}
            disabled={isDisabled}
            title={isUsedElsewhere ? `${points} pts already awarded to another entry` : undefined}
            className={`rounded-lg border px-2.5 py-1.5 text-sm transition-colors ${selected === points ? "border-esc-pink bg-esc-pink text-white" : "border-esc-border bg-white text-esc-black"} ${isDisabled ? "cursor-not-allowed opacity-40" : "hover:border-esc-pink hover:text-esc-pink"}`}
            onClick={() => {
              if (!isDisabled) {
                onSelect(points);
              }
            }}
          >
            {points}
          </button>
          );
        })}
      </div>
    </article>
  );
};
