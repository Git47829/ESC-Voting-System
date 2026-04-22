import type { ContestState, Country, Role, Song, VoteResult } from "../types";

export class ApiError extends Error {
  public readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

const DEFAULT_PUBLIC_VOTE_STATE = { votesRemaining: 20, votesCast: {} as Record<number, number> };
const PUBLIC_VOTE_STATE_KEY = "esc_public_vote_state";
const JURY_VOTE_STATE_KEY = "esc_jury_vote_state";

const trimTrailingSlash = (value: string) => value.replace(/\/+$/, "");
const crudApiBase = trimTrailingSlash(import.meta.env.VITE_CRUD_API_BASE_URL ?? "/crud-api");
const buildUrl = (path: string) => `${crudApiBase}${path.startsWith("/") ? path : `/${path}`}`;

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

const fetchJson = async <T>(path: string, init?: RequestInit): Promise<T> => {
  const response = await fetch(buildUrl(path), {
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

const fetchJsonWithRefresh = async <T>(path: string, init?: RequestInit): Promise<T> => {
  try {
    return await fetchJson<T>(path, init);
  } catch (error) {
    if (!(error instanceof ApiError) || error.status !== 403 || path === "/auth/refresh") {
      throw error;
    }

    await fetchJson<{ authenticated: boolean }>("/auth/refresh", { method: "POST" });
    return fetchJson<T>(path, init);
  }
};

const toNumericVoteMap = (value: unknown): Record<number, number> => {
  if (!value || typeof value !== "object") {
    return {};
  }

  const entries = Object.entries(value as Record<string, unknown>)
    .map(([songId, points]) => [Number(songId), Number(points)] as const)
    .filter(([songId, points]) => Number.isFinite(songId) && Number.isFinite(points));

  return Object.fromEntries(entries);
};

const loadLocalState = <T>(key: string, fallback: T): T => {
  if (typeof window === "undefined") {
    return fallback;
  }

  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return fallback;
    return { ...fallback, ...(JSON.parse(raw) as object) } as T;
  } catch {
    return fallback;
  }
};

const saveLocalState = (key: string, value: unknown) => {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    // ignore storage errors
  }
};

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

interface AuthResponse {
  authenticated?: boolean;
  user?: {
    role?: Role;
  };
}

export const api = {
  getSongs: async (): Promise<Song[]> => {
    const data = await fetchJson<{ payload: Song[] }>("/songs/");
    const seen = new Map<number, Song>();
    for (const song of data.payload ?? []) {
      if (!seen.has(song.songId)) {
        seen.set(song.songId, song);
      }
    }
    return [...seen.values()];
  },
  getVotes: async (): Promise<VoteResult[]> => {
    const data = await fetchJson<{ payload: VoteResult[] }>("/votes/");
    return data.payload;
  },
  getVoteState: async (): Promise<{ votesRemaining: number; votesCast: Record<number, number> }> =>
    loadLocalState(PUBLIC_VOTE_STATE_KEY, DEFAULT_PUBLIC_VOTE_STATE),
  getResults: (): Promise<VoteResult[]> => fetchJson<VoteResult[]>("/results"),
  getCountries: async (): Promise<Country[]> => {
    const data = await fetchJson<{ payload: Country[] }>("/countries/");
    return data.payload;
  },
  getContestCurrent: async (): Promise<ContestState | null> => {
    try {
      const data = await fetchJson<ContestState | { payload?: ContestState; error?: string }>("/contest/current");
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
  submitVote: async (payload: {
    songID: number;
    phoneNum: string;
    ownCountry: string;
    points: number;
  }): Promise<{ message: string }> => {
    const params = new URLSearchParams({
      songID: String(payload.songID),
      phoneNum: payload.phoneNum,
      ownCountry: payload.ownCountry,
      points: String(payload.points)
    });

    const response = await fetchJson<{
      message?: string;
      votes_remaining?: number;
      votes_cast?: Record<string, number>;
    }>(`/vote/?${params.toString()}`, {
      method: "POST"
    });

    if (typeof response.votes_remaining === "number") {
      saveLocalState(PUBLIC_VOTE_STATE_KEY, {
        votesRemaining: response.votes_remaining,
        votesCast: toNumericVoteMap(response.votes_cast)
      });
    }

    return { message: response.message ?? "Vote submitted" };
  },
  authLogin: async (email: string, password: string, role: Role): Promise<{ message: string }> => {
    return fetchJson<{ message: string }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password, role })
    });
  },
  authVerifyCode: async (email: string, code: string, role: Role): Promise<AuthResponse> => {
    return fetchJson<AuthResponse>("/auth/verify-code", {
      method: "POST",
      body: JSON.stringify({ email, code, role })
    });
  },
  authVerify: async (): Promise<{ ok: boolean; role: Role | null }> => {
    const data = await fetchJsonWithRefresh<AuthResponse>("/auth/verify", { method: "POST" });
    return {
      ok: Boolean(data.authenticated),
      role: data.user?.role ?? null
    };
  },
  authMe: async (): Promise<{ authenticated: boolean; role: Role | null }> => {
    const data = await fetchJsonWithRefresh<AuthResponse>("/auth/me");
    return {
      authenticated: Boolean(data.authenticated),
      role: data.user?.role ?? null
    };
  },
  logout: (): Promise<{ ok: boolean }> =>
    fetchJson<{ ok: boolean }>("/auth/logout", { method: "POST" }),
  adminOpen: (): Promise<{ message: string }> =>
    fetchJsonWithRefresh<{ message?: string; payload?: string }>("/admin/open", { method: "POST" }).then((data) => ({
      message: data.message ?? data.payload ?? "Voting opened"
    })),
  adminClose: (): Promise<{ message: string }> =>
    fetchJsonWithRefresh<{ message?: string; payload?: string }>("/admin/close", { method: "POST" }).then((data) => ({
      message: data.message ?? data.payload ?? "Voting closed"
    })),
  adminDeleteVotes: (): Promise<{ message: string }> =>
    fetchJsonWithRefresh<{ message?: string; payload?: string }>("/admin/deleteVotes/", { method: "DELETE" }).then((data) => ({
      message: data.message ?? data.payload ?? "Votes deleted"
    })),
  adminAddCountry: (countryId: string, countryName: string, pot: number): Promise<{ message: string }> => {
    const params = new URLSearchParams({ ID: countryId, Name: countryName, Pot: String(pot) });
    return fetchJsonWithRefresh(`/admin/addCountry/?${params.toString()}`, { method: "POST" }).then(() => ({
      message: "Country added"
    }));
  },
  adminAddArtist: (firstName: string, lastName: string, countryId: string): Promise<{ message: string }> => {
    const params = new URLSearchParams({ FirstName: firstName, LastName: lastName, CountryID: countryId });
    return fetchJsonWithRefresh(`/admin/addArtist/?${params.toString()}`, { method: "POST" }).then(() => ({
      message: "Artist added"
    }));
  },
  adminAddSong: (payload: {
    songName: string;
    countryId: string;
    artistId: number;
    youtubeUrl: string;
  }): Promise<{ message: string }> => {
    const params = new URLSearchParams({
      SongName: payload.songName,
      CountryID: payload.countryId,
      KuenstlerID: String(payload.artistId),
      YoutubeURL: payload.youtubeUrl
    });
    return fetchJsonWithRefresh(`/admin/addSong/?${params.toString()}`, { method: "POST" }).then(() => ({
      message: "Song added"
    }));
  },
  adminStartContest: (): Promise<{ payload: ContestState }> =>
    fetchJsonWithRefresh<{ payload: ContestState }>("/admin/startContest", { method: "POST" }),
  adminAdvanceContest: (): Promise<{ payload: ContestState }> =>
    fetchJsonWithRefresh<{ payload: ContestState }>("/admin/advanceContest", { method: "POST" }),
  juryVote: async (songID: number, points: number): Promise<{ message: string }> => {
    const params = new URLSearchParams({ songID: String(songID), points: String(points) });
    await fetchJsonWithRefresh(`/jury/vote/?${params.toString()}`, {
      method: "POST"
    });

    const current = loadLocalState<{ votesCast: Record<number, number> }>(JURY_VOTE_STATE_KEY, { votesCast: {} });
    const next = { votesCast: { ...current.votesCast, [songID]: points } };
    saveLocalState(JURY_VOTE_STATE_KEY, next);

    return { message: "Jury vote submitted" };
  },
  getJuryVoteState: async (): Promise<{ votesCast: Record<number, number> }> =>
    loadLocalState(JURY_VOTE_STATE_KEY, { votesCast: {} })
};
