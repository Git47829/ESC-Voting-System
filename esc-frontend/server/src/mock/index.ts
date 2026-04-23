import { config } from "../config.js";
import type { ContestState, Country, Song, VoteResult } from "../types.js";
import { mockContestDataService } from "./contest-data-service.js";
import { mockSongDataService } from "./song-data-service.js";
import { mockVoteDataService } from "./vote-data-service.js";

/**
 * Unified MockDataService that coordinates all mock data services.
 * Separates concerns into domain-specific services while maintaining a single entry point.
 */
export class MockDataService {
  public getSongs(): Song[] {
    return mockSongDataService.getSongs();
  }

  public getCountries(): Country[] {
    return mockContestDataService.getCountries();
  }

  public getVotes(): VoteResult[] {
    return mockVoteDataService.getVotes();
  }

  public getContestCurrent(): ContestState {
    const songs = this.getSongs();
    return mockContestDataService.getContestCurrent(songs);
  }

  public setVotingOpen(open: boolean): void {
    mockSongDataService.setVotingOpen(open);
  }

  public resetVotes(): void {
    mockSongDataService.resetVotes();
  }

  public addCountry(countryId: string, countryName: string, pot: number): void {
    mockContestDataService.addCountry(countryId, countryName, pot);
  }

  public addArtist(): void {
    // In mock mode artists are represented inside songs; endpoint remains successful.
  }

  public addSong(input: {
    countryId: string;
    countryName: string;
    songName: string;
    artistFirstName: string;
    artistLastName: string;
    youtubeUrl?: string;
  }): Song {
    return mockSongDataService.addSong(input);
  }

  public startContest(): ContestState {
    const songs = this.getSongs();
    return mockContestDataService.startContest(songs.length);
  }

  public advanceContest(): ContestState {
    const songs = this.getSongs();
    return mockContestDataService.advanceContest(songs.length);
  }

  public castPublicVote(
    songId: number,
    points: number,
    sessionVoteState?: { votesRemaining: number; votesCast: Record<number, number> }
  ): { song: Song; voteState: { votesRemaining: number; votesCast: Record<number, number> } } {
    const state = sessionVoteState ?? { votesRemaining: config.totalVotePoints, votesCast: {} };
    return mockSongDataService.castPublicVote(songId, points, state);
  }

  public castJuryVote(songId: number, points: number): Song {
    return mockSongDataService.castJuryVote(songId, points);
  }
}

export const mockDataService = new MockDataService();
