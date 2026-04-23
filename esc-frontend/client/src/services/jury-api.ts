import { fetchJson } from "./http-client";

export const juryVote = (songID: number, points: number): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/jury/vote", {
    method: "POST",
    body: JSON.stringify({ songID, points })
  });

export const getJuryVoteState = (): Promise<{ votesCast: Record<number, number> }> =>
  fetchJson<{ payload: { votesCast: Record<number, number> } }>("/api/jury/vote/state")
    .then((data) => data.payload);
