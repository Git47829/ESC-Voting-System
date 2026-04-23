import type { Song, Country, ContestState, VoteResult } from "../types.js";

export type AuthCredentials = { token?: string; email?: string };

export interface AuthService {
  loginWithToken(role: "admin" | "jury", token: string): Promise<{ ok: boolean; role: string }>;
  loginWithEmail(email: string, password: string, role: "admin" | "jury"): Promise<{ ok: boolean } | { error: string }>;
  verifyEmail(email: string, code: string, role: "admin" | "jury"): Promise<{ ok: boolean; role: string } | { error: string }>;
  authenticateToken(token: string, role: "admin" | "jury"): Promise<{ ok: boolean } | { error: string }>;
}

export interface ContestService {
  getSongs(): Promise<Song[]>;
  getCountries(): Promise<Country[]>;
  getContestCurrent(): Promise<ContestState>;
  startContest(credentials?: AuthCredentials): Promise<ContestState>;
  advanceContest(credentials?: AuthCredentials): Promise<ContestState>;
  addCountry(countryId: string, countryName: string, pot: number, credentials?: AuthCredentials): Promise<{ message: string }>;
  addArtist(firstName: string, lastName: string, countryId: string, credentials?: AuthCredentials): Promise<{ message: string }>;
  addSong(input: {
    countryId: string;
    countryName?: string;
    songName: string;
    artistFirstName?: string;
    artistLastName?: string;
    youtubeUrl?: string;
  }, credentials?: AuthCredentials): Promise<Song>;
}

export interface VotingService {
  castPublicVote(
    songId: number,
    points: number,
    sessionCookie?: string,
    additionalParams?: Record<string, unknown>
  ): Promise<{ message: string; payload: { votesRemaining: number; votesCast: Record<number, number> } }>;
  castJuryVote(songId: number, points: number, credentials?: AuthCredentials): Promise<{ message: string }>;
  getVotes(): Promise<VoteResult[]>;
  setVotingOpen(open: boolean, credentials?: AuthCredentials): Promise<{ message: string }>;
  resetVotes(credentials?: AuthCredentials): Promise<{ message: string }>;
}

export interface ResultsService {
  getResults(): Promise<VoteResult[]>;
  getStats(): Promise<{ totalPublic: number; totalJury: number; byCountry: Array<{ countryId: string; country: string; total: number }> }>;
}
