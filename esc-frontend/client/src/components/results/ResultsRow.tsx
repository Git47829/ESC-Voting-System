import type { VoteResult } from "../../types";

export const ResultsRow = ({ item }: { item: VoteResult }) => {
  return (
    <div className="grid grid-cols-[3rem_1fr_6rem_6rem_6rem] items-center border-b border-esc-muted/40 px-2 py-2 text-sm">
      <span>#{item.rank}</span>
      <span>{item.country} - {item.name}</span>
      <span>{item.escPublicPts}</span>
      <span>{item.juryPts}</span>
      <span className="font-bold text-esc-yellow">{item.totalPts}</span>
    </div>
  );
};

