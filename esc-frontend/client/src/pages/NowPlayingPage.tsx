import { useMemo, useState } from "react";

import { api } from "../api/client";
import { useContestPoll } from "../hooks/useContestPoll";

export const NowPlayingPage = () => {
  const contest = useContestPoll();
  const [points, setPoints] = useState(1);
  const [phoneNum, setPhoneNum] = useState("");
  const [ownCountry, setOwnCountry] = useState("");

  const progress = useMemo(() => {
    if (!contest || contest.totalSongs === 0) return 0;
    return ((contest.currentIndex + 1) / contest.totalSongs) * 100;
  }, [contest]);

  if (!contest?.currentSong) {
    return <p>No active contest song.</p>;
  }

  return (
    <section className="space-y-4">
      <h1 className="text-3xl font-bold">Running Now</h1>
      <div className="h-2 bg-esc-surface">
        <div className="progress-shine h-2 bg-esc-yellow" style={{ width: `${progress}%` }} />
      </div>
      {contest.currentSong.youtubeUrl ? (
        <iframe
          title="Current performance"
          className="aspect-video w-full border border-esc-muted"
          src={contest.currentSong.youtubeUrl}
          allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
          allowFullScreen
        />
      ) : null}
      <div className="border border-esc-muted p-4">
        <h2 className="text-xl font-bold">{contest.currentSong.countryName} - {contest.currentSong.songName}</h2>
        <p className="text-sm text-esc-muted">{contest.currentSong.artistFirstName} {contest.currentSong.artistLastName}</p>
      </div>
      <form
        className="grid gap-3 border border-esc-muted p-4 md:grid-cols-4"
        onSubmit={(e) => {
          e.preventDefault();
          void api.submitVote({
            songID: contest.currentSong!.songId,
            phoneNum,
            ownCountry,
            points
          });
        }}
      >
        <input className="border border-esc-muted bg-transparent px-2 py-1" placeholder="Phone" value={phoneNum} onChange={(e) => setPhoneNum(e.target.value)} />
        <input className="border border-esc-muted bg-transparent px-2 py-1" placeholder="Own Country" value={ownCountry} onChange={(e) => setOwnCountry(e.target.value.toUpperCase())} />
        <input className="border border-esc-muted bg-transparent px-2 py-1" type="number" min={1} max={20} value={points} onChange={(e) => setPoints(Number(e.target.value))} />
        <button className="border border-esc-yellow px-3 py-1 text-esc-yellow">Vote now</button>
      </form>
    </section>
  );
};

