import { fetchJson } from "./http-client";
import type { Song, VoteResult } from "../types";

export const getSongs = async (): Promise<Song[]> => {
  const data = await fetchJson<{ payload: Song[] }>("/api/songs");
  return data.payload;
};

export const getVotes = async (): Promise<VoteResult[]> => {
  const data = await fetchJson<{ payload: VoteResult[] }>("/api/votes");
  return data.payload;
};

export const submitVote = (payload: {
  songID: number;
  phoneNum: string;
  ownCountry: string;
  points: number;
}): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/vote", {
    method: "POST",
    body: JSON.stringify(payload)
  });

export const getVoteState = (): Promise<{ votesRemaining: number; votesCast: Record<number, number> }> =>
  fetchJson("/api/vote/state");
