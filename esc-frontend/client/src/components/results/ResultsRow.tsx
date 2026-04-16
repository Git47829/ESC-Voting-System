import type { VoteResult } from "../../types";

export const ResultsRow = ({ item }: { item: VoteResult }) => {
  const rank = item.rank ?? 0;
  const isTop3 = rank > 0 && rank <= 3;

  return (
    <div
      className={`grid grid-cols-[3.5rem_1fr_6rem_6rem_6rem] items-center border-b border-esc-border/70 px-3 py-3 text-sm transition-colors ${
        isTop3 ? "bg-esc-pink-soft/40" : "bg-white"
      }`}
    >
      <span className="font-semibold text-esc-black">{rank > 0 ? `#${rank}` : "-"}</span>
      <span className="truncate pr-3 text-esc-black">
        <span className="font-semibold">{item.country}</span> - {item.name}
      </span>
      <span className="text-esc-black-soft">{item.escPublicPts}</span>
      <span className="text-esc-black-soft">{item.juryPts}</span>
      <span className="font-bold text-esc-pink">{item.totalPts}</span>
    </div>
  );
};
