import type { ContestState, Country, Role, SessionState, Song, VoteResult } from "../types";

export class ApiError extends Error {
  public readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const extractErrorMessage = (value: unknown): string | null => {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  }

  if (!value || typeof value !== "object") {
    return null;
  }

  const record = value as Record<string, unknown>;
  const direct =
    extractErrorMessage(record.error) ??
    extractErrorMessage(record.message) ??
    extractErrorMessage(record.detail) ??
    extractErrorMessage(record.title) ??
    extractErrorMessage(record.payload);

  if (direct) {
    return direct;
  }

  if (Array.isArray(record.errors)) {
    const first = record.errors.map(extractErrorMessage).find(Boolean);
    return first ?? null;
  }

  return null;
};

const fetchJson = async <T>(url: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(url, {
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {})
    },
    ...init
  });

  const rawBody = await response.text();
  let data = {} as T;
  if (rawBody.length > 0) {
      try {
        data = JSON.parse(rawBody) as T;
      } catch {
        if (!response.ok) {
          throw new ApiError(rawBody.trim() || `Request failed: ${response.status}`, response.status);
        }
        throw new Error("Received invalid JSON response");
      }
  }

  if (!response.ok) {
    const message =
      extractErrorMessage(data) ??
      (response.statusText || `Request failed: ${response.status}`);
    throw new ApiError(message, response.status);
  }

  return data;
};

const isContestState = (value: unknown): value is ContestState => {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.runId === "string" &&
    typeof record.currentIndex === "number" &&
    typeof record.totalSongs === "number" &&
    typeof record.contestActive === "boolean" &&
    ("currentSong" in record)
  );
};

export const api = {
  getSongs: async (): Promise<Song[]> => {
    const data = await fetchJson<{ payload: Song[] }>("/api/songs");
    return data.payload;
  },
  getVotes: async (): Promise<VoteResult[]> => {
    const data = await fetchJson<{ payload: VoteResult[] }>("/api/votes");
    return data.payload;
  },
  getVoteState: (): Promise<{ votesRemaining: number; votesCast: Record<number, number> }> =>
    fetchJson("/api/vote/state"),
  getResults: async (): Promise<VoteResult[]> => {
    const data = await fetchJson<VoteResult[] | { payload?: VoteResult[] }>("/api/results");
    if (Array.isArray(data)) {
      return data;
    }
    if (Array.isArray(data.payload)) {
      return data.payload;
    }
    return [];
  },
  getCountries: async (): Promise<Country[]> => {
    const data = await fetchJson<{ payload: Country[] }>("/api/countries");
    return data.payload;
  },
  getContestCurrent: async (): Promise<ContestState | null> => {
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
  },
  submitVote: (payload: {
    songID: number;
    phoneNum: string;
    ownCountry: string;
    points: number;
  }): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/vote", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  authLogin: (email: string, password: string, role: Role): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password, role })
    }),
  authVerify: (email: string, code: string, role: Role): Promise<{ ok: boolean; role: Role }> =>
    fetchJson<{ ok: boolean; role: Role }>("/api/auth/verify", {
      method: "POST",
      body: JSON.stringify({ email, code, role })
    }),
  login: (role: Role, token: string): Promise<{ ok: boolean; role: Role }> =>
    fetchJson<{ ok: boolean; role: Role }>("/api/login", {
      method: "POST",
      body: JSON.stringify({ role, token })
    }),
  logout: (): Promise<{ ok: boolean }> =>
    fetchJson<{ ok: boolean }>("/api/logout", { method: "POST" }),
  session: (): Promise<SessionState> => fetchJson<SessionState>("/api/session"),
  adminOpen: (): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/open", { method: "POST" }),
  adminClose: (): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/close", { method: "POST" }),
  adminDeleteVotes: (): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/deleteVotes", { method: "DELETE" }),
  adminAddCountry: (countryId: string, countryName: string, pot: number): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/addCountry", {
      method: "POST",
      body: JSON.stringify({ countryId, countryName, pot })
    }),
  adminAddArtist: (firstName: string, lastName: string, countryId: string): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/addArtist", {
      method: "POST",
      body: JSON.stringify({ firstName, lastName, countryId })
    }),
  adminAddSong: (payload: {
    songName: string;
    countryId: string;
    artistFirstName: string;
    artistLastName: string;
    youtubeUrl: string;
  }): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/admin/addSong", {
      method: "POST",
      body: JSON.stringify(payload)
    }),
  adminStartContest: (): Promise<{ payload: ContestState }> =>
    fetchJson<{ payload: ContestState }>("/api/admin/startContest", { method: "POST" }),
  adminAdvanceContest: (): Promise<{ payload: ContestState }> =>
    fetchJson<{ payload: ContestState }>("/api/admin/advanceContest", { method: "POST" }),
  juryVote: (songID: number, points: number): Promise<{ message: string }> =>
    fetchJson<{ message: string }>("/api/jury/vote", {
      method: "POST",
      body: JSON.stringify({ songID, points })
    }),
  getJuryVoteState: (): Promise<{ votesCast: Record<number, number> }> =>
    fetchJson<{ payload: { votesCast: Record<number, number> } }>("/api/jury/vote/state")
      .then((data) => data.payload),
  getStats: (): Promise<{
    totalPublic: number;
    totalJury: number;
    byCountry: Array<{ countryId: string; country: string; total: number }>;
  }> => fetchJson("/api/stats")
};
