import { randomUUID } from "node:crypto";
import { config } from "../config.js";
import { mockContest } from "./contest.js";
import { mockCountries } from "./countries.js";
import { mockSongs } from "./songs.js";
const clone = (value) => JSON.parse(JSON.stringify(value));
export class MockDataService {
    songs = clone(mockSongs);
    countries = clone(mockCountries);
    contest = { ...mockContest };
    getSongs() {
        return this.songs.map((song) => ({ ...song, totalVotes: song.publicVotes + song.juryVotes }));
    }
    getCountries() {
        return clone(this.countries);
    }
    getVotes() {
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
    getContestCurrent() {
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
    setVotingOpen(open) {
        this.songs = this.songs.map((song) => ({ ...song, votingIsOpen: open }));
    }
    resetVotes() {
        this.songs = this.songs.map((song) => ({ ...song, publicVotes: 0, juryVotes: 0, totalVotes: 0 }));
    }
    addCountry(countryId, countryName, pot) {
        this.countries.push({ id: countryId, name: countryName, pot });
    }
    addArtist() {
        // In mock mode artists are represented inside songs; endpoint remains successful.
    }
    addSong(input) {
        const nextId = this.songs.reduce((max, s) => Math.max(max, s.songId), 0) + 1;
        const song = {
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
    startContest() {
        this.contest = {
            runId: randomUUID(),
            currentIndex: 0,
            totalSongs: this.songs.length,
            contestActive: true
        };
        return this.getContestCurrent();
    }
    advanceContest() {
        if (this.contest.currentIndex < Math.max(this.songs.length - 1, 0)) {
            this.contest.currentIndex += 1;
        }
        return this.getContestCurrent();
    }
    castPublicVote(songId, points, sessionVoteState) {
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
    castJuryVote(songId, points) {
        const song = this.songs.find((item) => item.songId === songId);
        if (!song) {
            throw new Error("Song not found");
        }
        song.juryVotes += points;
        return song;
    }
}
export const mockDataService = new MockDataService();
