import type { Song } from "../../types";

export const EntryTable = ({ songs }: { songs: Song[] }) => {
  return (
    <div className="overflow-hidden rounded-[2rem] border border-esc-border bg-white/96 shadow-[0_20px_48px_rgba(0,0,0,0.06)]">
      <div className="overflow-x-auto">
        <table className="w-full min-w-[760px] text-sm">
          <thead className="bg-[#141414] text-white/55">
            <tr>
              <th className="px-5 py-4 text-left text-[11px] font-semibold uppercase tracking-[0.18em]">
                Country
              </th>
              <th className="px-5 py-4 text-left text-[11px] font-semibold uppercase tracking-[0.18em]">
                Song
              </th>
              <th className="px-5 py-4 text-left text-[11px] font-semibold uppercase tracking-[0.18em]">
                Public
              </th>
              <th className="px-5 py-4 text-left text-[11px] font-semibold uppercase tracking-[0.18em]">
                Jury
              </th>
              <th className="px-5 py-4 text-right text-[11px] font-semibold uppercase tracking-[0.18em]">
                Total
              </th>
            </tr>
          </thead>
          <tbody>
            {songs.map((song) => (
              <tr
                key={song.songId}
                className="border-t border-esc-border/80 text-esc-black transition-colors duration-200 hover:bg-esc-pink-soft"
              >
                <td className="px-5 py-4 align-top">
                  <div>
                    <p className="font-semibold text-esc-black">{song.countryName}</p>
                    <p className="mt-1 text-[11px] uppercase tracking-[0.18em] text-esc-muted">
                      {song.countryId}
                    </p>
                  </div>
                </td>
                <td className="px-5 py-4 align-top">
                  <div>
                    <p className="font-semibold text-esc-black">{song.songName}</p>
                    <p className="mt-1 text-sm text-esc-muted">
                      {song.artistFirstName} {song.artistLastName}
                    </p>
                  </div>
                </td>
                <td className="px-5 py-4 align-top text-esc-black-soft">{song.publicVotes}</td>
                <td className="px-5 py-4 align-top text-esc-black-soft">{song.juryVotes}</td>
                <td className="px-5 py-4 text-right align-top font-semibold text-esc-pink">
                  {song.totalVotes}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
