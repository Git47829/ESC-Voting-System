import { fetchJson } from "./http-client";
import type { ContestState } from "../types";
import { ApiError } from "./error-handler";

const isContestState = (value: unknown): value is ContestState => {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    (typeof record.runId === "string" || typeof record.runId === "number") &&
    typeof record.currentIndex === "number" &&
    typeof record.totalSongs === "number" &&
    typeof record.contestActive === "boolean" &&
    ("currentSong" in record)
  );
};

export const adminOpen = (): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/open", { method: "POST" });

export const adminClose = (): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/close", { method: "POST" });

export const adminDeleteVotes = (): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/deleteVotes", { method: "DELETE" });

export const adminAddCountry = (countryId: string, countryName: string, pot: number): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/addCountry", {
    method: "POST",
    body: JSON.stringify({ countryId, countryName, pot })
  });

export const adminAddArtist = (firstName: string, lastName: string, countryId: string): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/addArtist", {
    method: "POST",
    body: JSON.stringify({ firstName, lastName, countryId })
  });

export const adminAddSong = (payload: {
  songName: string;
  countryId: string;
  artistId: number;
  youtubeUrl: string;
}): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/admin/addSong", {
    method: "POST",
    body: JSON.stringify(payload)
  });

export const adminStartContest = (): Promise<{ payload: ContestState }> =>
  fetchJson<{ payload: ContestState }>("/api/admin/startContest", { method: "POST" });

export const adminAdvanceContest = (): Promise<{ payload: ContestState }> =>
  fetchJson<{ payload: ContestState }>("/api/admin/advanceContest", { method: "POST" });
