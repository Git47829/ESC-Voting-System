import type { Song } from "../../types";

export const EntryTable = ({ songs }: { songs: Song[] }) => {
  return (
    <div className="overflow-x-auto rounded-[1.75rem] border border-esc-border bg-white/92 shadow-[0_16px_36px_rgba(0,0,0,0.05)]">
      <table className="w-full text-sm">
        <thead className="bg-esc-surface2 text-esc-muted">
          <tr>
            <th className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em]">
              Country
            </th>
            <th className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em]">
              Song
            </th>
            <th className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em]">
              Public
            </th>
            <th className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em]">
              Jury
            </th>
            <th className="px-3 py-3 text-left text-xs font-semibold uppercase tracking-[0.12em]">
              Total
            </th>
          </tr>
        </thead>
        <tbody>
          {songs.map((song) => (
            <tr
              key={song.songId}
              className="border-t border-esc-border/70 text-esc-black"
            >
              <td className="px-3 py-2.5 font-medium">{song.countryName}</td>
              <td className="px-3 py-2.5">{song.songName}</td>
              <td className="px-3 py-2.5 text-esc-black-soft">
                {song.publicVotes}
              </td>
              <td className="px-3 py-2.5 text-esc-black-soft">
                {song.juryVotes}
              </td>
              <td className="px-3 py-2.5 font-semibold text-esc-pink">
                {song.totalVotes}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};
