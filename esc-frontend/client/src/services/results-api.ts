import { fetchJson } from "./http-client";
import type { ContestState, Country, VoteResult } from "../types";
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

export const getResults = async (): Promise<VoteResult[]> => {
  const data = await fetchJson<VoteResult[] | { payload?: VoteResult[] }>("/api/results");
  if (Array.isArray(data)) {
    return data;
  }
  if (Array.isArray(data.payload)) {
    return data.payload;
  }
  return [];
};

export const getCountries = async (): Promise<Country[]> => {
  const data = await fetchJson<{ payload: Country[] }>("/api/countries");
  return data.payload;
};

export const getContestCurrent = async (): Promise<ContestState | null> => {
  try {
    const data = await fetchJson<ContestState | { payload?: ContestState; error?: string }>("/api/contest/current");
    if (isContestState(data)) {
      return data;
    }
    if (data && typeof data === "object" && "payload" in data && isContestState(data.payload)) {
      return data.payload;
    }
    return null;
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) {
      return null;
    }
    throw error;
  }
};

export const getStats = (): Promise<{
  totalPublic: number;
  totalJury: number;
  byCountry: Array<{ countryId: string; country: string; total: number }>;
}> => fetchJson("/api/stats");
