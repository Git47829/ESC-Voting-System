import type { Song } from "../types.js";
import { mockSongs } from "./songs.js";

const clone = <T>(value: T): T => JSON.parse(JSON.stringify(value)) as T;

export class MockSongDataService {
  private songs: Song[] = clone(mockSongs);

  public getSongs(): Song[] {
    return this.songs.map((song) => ({ ...song, totalVotes: song.publicVotes + song.juryVotes }));
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

  public castPublicVote(
    songId: number,
    points: number,
    sessionVoteState?: { votesRemaining: number; votesCast: Record<number, number> }
  ): { song: Song; voteState: { votesRemaining: number; votesCast: Record<number, number> } } {
    const state = sessionVoteState ?? { votesRemaining: 10, votesCast: {} };
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

  public setVotingOpen(open: boolean): void {
    this.songs = this.songs.map((song) => ({ ...song, votingIsOpen: open }));
  }

  public resetVotes(): void {
    this.songs = this.songs.map((song) => ({ ...song, publicVotes: 0, juryVotes: 0, totalVotes: 0 }));
  }
}

export const mockSongDataService = new MockSongDataService();
