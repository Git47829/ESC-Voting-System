import { randomUUID } from "node:crypto";
import type { ContestState, Country } from "../types.js";
import { mockContest } from "./contest.js";
import { mockCountries } from "./countries.js";

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

export class MockContestDataService {
  private contest = { ...mockContest };
  private countries: Country[] = clone(mockCountries);

  public getCountries(): Country[] {
    return clone(this.countries);
  }

  public getContestCurrent(songs: { songId: number; countryName: string; countryId: string; songName: string; artistFirstName: string; artistLastName: string; publicVotes: number; juryVotes: number; totalVotes: number; votingIsOpen: boolean; youtubeUrl?: string }[]): ContestState {
    const currentSong = songs[this.contest.currentIndex] ?? null;
    return {
      runId: this.contest.runId,
      currentIndex: this.contest.currentIndex,
      totalSongs: songs.length,
      contestActive: this.contest.contestActive,
      currentSong
    };
  }

  public addCountry(countryId: string, countryName: string, pot: number): void {
    this.countries.push({ id: countryId, name: countryName, pot });
  }

  public startContest(songsLength: number): ContestState {
    this.contest = {
      runId: randomUUID(),
      currentIndex: 0,
      totalSongs: songsLength,
      contestActive: true
    };
    return this.contest as any;
  }

  public advanceContest(songsLength: number): ContestState {
    if (this.contest.currentIndex < Math.max(songsLength - 1, 0)) {
      this.contest.currentIndex += 1;
    }
    return { ...this.contest, totalSongs: songsLength, currentSong: null };
  }
}

export const mockContestDataService = new MockContestDataService();
