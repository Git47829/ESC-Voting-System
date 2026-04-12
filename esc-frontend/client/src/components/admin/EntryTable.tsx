import type { Song } from "../../types";

export const EntryTable = ({ songs }: { songs: Song[] }) => {
  return (
    <div className="overflow-x-auto border border-esc-muted">
      <table className="w-full text-sm">
        <thead className="bg-esc-surface">
          <tr>
            <th className="px-2 py-2 text-left">Country</th>
            <th className="px-2 py-2 text-left">Song</th>
            <th className="px-2 py-2 text-left">Public</th>
            <th className="px-2 py-2 text-left">Jury</th>
            <th className="px-2 py-2 text-left">Total</th>
          </tr>
        </thead>
        <tbody>
          {songs.map((song) => (
            <tr key={song.songId} className="border-t border-esc-muted/40">
              <td className="px-2 py-2">{song.countryName}</td>
              <td className="px-2 py-2">{song.songName}</td>
              <td className="px-2 py-2">{song.publicVotes}</td>
              <td className="px-2 py-2">{song.juryVotes}</td>
              <td className="px-2 py-2">{song.totalVotes}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
};

