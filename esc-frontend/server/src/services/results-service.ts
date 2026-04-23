import axios from "axios";
import type { ResultsService } from "./interfaces.js";
import type { Song, VoteResult } from "../types.js";
import { config } from "../config.js";
import { mockDataService } from "../mock/index.js";
import { upstream } from "../upstream.js";

export class ProductionResultsService implements ResultsService {
  async getResults() {
    const escResponse = await axios.get(`${config.escConverterUrl}/api/esc-points`, {
      timeout: config.apiTimeout,
      validateStatus: () => true
    });
    if (escResponse.status >= 400) {
      throw new Error("ESC converter unavailable");
    }

    const juryResponse = await upstream.get("/votes/");
    const juryMap = new Map<number, number>();
    const juryPayload = (juryResponse.data?.payload ?? []) as Array<{ id: number; juryVotes: number }>;
    juryPayload.forEach((entry) => juryMap.set(entry.id, entry.juryVotes ?? 0));

    const results = ((escResponse.data?.payload ?? []) as Array<{ songId: number; songName: string; country: string; countryId: string; escPoints: number }>).map((song) => {
      const juryPts = juryMap.get(song.songId) ?? 0;
      return {
        id: song.songId,
        name: song.songName,
        country: song.country,
        countryId: song.countryId,
        escPublicPts: song.escPoints,
        juryPts,
        totalPts: song.escPoints + juryPts
      };
    });

    results.sort((a, b) => b.totalPts - a.totalPts || a.id - b.id);
    return results.map((entry, index) => ({ ...entry, rank: index + 1 }));
  }

  async getStats() {
    return { totalPublic: 0, totalJury: 0, byCountry: [] };
  }
}

export class MockResultsService implements ResultsService {
  async getResults() {
    return mockDataService.getVotes();
  }

  async getStats() {
    const songs = mockDataService.getSongs();
    const totalPublic = songs.reduce((sum, song) => sum + song.publicVotes, 0);
    const totalJury = songs.reduce((sum, song) => sum + song.juryVotes, 0);
    const byCountry = songs
      .map((song: Song) => ({
        countryId: song.countryId,
        country: song.countryName,
        total: song.publicVotes + song.juryVotes
      }))
      .sort((a, b) => b.total - a.total || a.country.localeCompare(b.country));

    return {
      totalPublic,
      totalJury,
      byCountry
    };
  }
}
