export type Role = "admin" | "jury";

export interface Song {
  songId: number;
  countryId: string;
  countryName: string;
  songName: string;
  artistFirstName: string;
  artistLastName: string;
  publicVotes: number;
  juryVotes: number;
  totalVotes: number;
  votingIsOpen: boolean;
  youtubeUrl?: string;
}

export interface Country {
  id: string;
  name: string;
}

export interface ContestState {
  runId: string;
  currentIndex: number;
  totalSongs: number;
  contestActive: boolean;
  currentSong: Song | null;
}

export interface VoteResult {
  id: number;
  country: string;
  countryId: string;
  name: string;
  escPublicPts: number;
  juryPts: number;
  totalPts: number;
  rank?: number;
}

export interface SessionState {
  authenticated: boolean;
  role: Role | null;
  token: string | null;
}

export interface FlashMessage {
  id: string;
  category: "success" | "error" | "info";
  message: string;
}

