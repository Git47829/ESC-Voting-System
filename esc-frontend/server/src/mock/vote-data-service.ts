import type { VoteResult } from "../types.js";
import { mockSongDataService } from "./song-data-service.js";

export class MockVoteDataService {
  public getVotes(): VoteResult[] {
    return mockSongDataService
      .getSongs()
      .map((song) => ({
        id: song.songId,
        name: song.songName,
        country: song.countryName,
        countryId: song.countryId,
        escPublicPts: song.publicVotes,
        juryPts: song.juryVotes,
        totalPts: song.publicVotes + song.juryVotes
      }))
      .sort((a, b) => b.totalPts - a.totalPts || a.id - b.id)
      .map((entry, index) => ({ ...entry, rank: index + 1 }));
  }
}

export const mockVoteDataService = new MockVoteDataService();
