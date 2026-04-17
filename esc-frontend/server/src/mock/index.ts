import { randomUUID } from "node:crypto";

import { config } from "../config.js";
import type { ContestState, Country, Song, VoteResult } from "../types.js";
import { mockContest } from "./contest.js";
import { mockCountries } from "./countries.js";
import { mockSongs } from "./songs.js";

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

export class MockDataService {
  private songs: Song[] = clone(mockSongs);
  private countries: Country[] = clone(mockCountries);
  private contest = { ...mockContest };

  public getSongs(): Song[] {
    return this.songs.map((song) => ({ ...song, totalVotes: song.publicVotes + song.juryVotes }));
  }

  public getCountries(): Country[] {
    return clone(this.countries);
  }

  public getVotes(): VoteResult[] {
    return this.getSongs()
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

  public getContestCurrent(): ContestState {
    const songs = this.getSongs();
    const currentSong = songs[this.contest.currentIndex] ?? null;
    return {
      runId: this.contest.runId,
      currentIndex: this.contest.currentIndex,
      totalSongs: songs.length,
      contestActive: this.contest.contestActive,
      currentSong
    };
  }

  public setVotingOpen(open: boolean): void {
    this.songs = this.songs.map((song) => ({ ...song, votingIsOpen: open }));
  }

  public resetVotes(): void {
    this.songs = this.songs.map((song) => ({ ...song, publicVotes: 0, juryVotes: 0, totalVotes: 0 }));
  }

  public addCountry(countryId: string, countryName: string, pot: number): void {
    this.countries.push({ id: countryId, name: countryName, pot });
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
    const nextId = this.songs.reduce((max, s) => Math.max(max, s.songId), 0) + 1;
    const song: Song = {
      songId: nextId,
      countryId: input.countryId,
      countryName: input.countryName,
      songName: input.songName,
      artistFirstName: input.artistFirstName,
      artistLastName: input.artistLastName,
      publicVotes: 0,
      juryVotes: 0,
      totalVotes: 0,
      votingIsOpen: this.songs[0]?.votingIsOpen ?? true,
      youtubeUrl: input.youtubeUrl
    };
    this.songs.push(song);
    return song;
  }

  public startContest(): ContestState {
    this.contest = {
      runId: randomUUID(),
      currentIndex: 0,
      totalSongs: this.songs.length,
      contestActive: true
    };
    return this.getContestCurrent();
  }

  public advanceContest(): ContestState {
    if (this.contest.currentIndex < Math.max(this.songs.length - 1, 0)) {
      this.contest.currentIndex += 1;
    }
    return this.getContestCurrent();
  }

  public castPublicVote(songId: number, points: number, sessionVoteState?: { votesRemaining: number; votesCast: Record<number, number> }): { song: Song; voteState: { votesRemaining: number; votesCast: Record<number, number> } } {
    const state = sessionVoteState ?? { votesRemaining: config.totalVotePoints, votesCast: {} };
    const song = this.songs.find((item) => item.songId === songId);
    if (!song) {
      throw new Error("Song not found");
    }
    if (points > state.votesRemaining) {
      throw new Error("Not enough remaining points");
    }

    song.publicVotes += points;
    state.votesRemaining -= points;
    state.votesCast[songId] = (state.votesCast[songId] ?? 0) + points;
    return { song, voteState: state };
  }

  public castJuryVote(songId: number, points: number): Song {
    const song = this.songs.find((item) => item.songId === songId);
    if (!song) {
      throw new Error("Song not found");
    }
    song.juryVotes += points;
    return song;
  }
}

export const mockDataService = new MockDataService();
