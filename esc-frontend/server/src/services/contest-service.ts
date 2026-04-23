import type { ContestService } from "./interfaces.js";
import type { Song } from "../types.js";
import { upstream } from "../upstream.js";
import { authHeaders, normalizeYoutubeUrl, toInt } from "../utils/helpers.js";

export class ProductionContestService implements ContestService {
  async getSongs() {
    const response = await upstream.get("/songs/");
    if (response.status === 200 && Array.isArray((response.data as Record<string, unknown>)?.payload)) {
      const raw = (response.data as { payload: Song[] }).payload;
      const seen = new Map<number, Song>();
      for (const song of raw) {
        if (!seen.has(song.songId)) {
          seen.set(song.songId, song);
        }
      }
      return [...seen.values()];
    }
    throw new Error("Failed to fetch songs");
  }

  async getCountries() {
    const response = await upstream.get("/countries/");
    if (response.status === 200) {
      return response.data?.payload ?? [];
    }
    throw new Error("Failed to fetch countries");
  }

  async getContestCurrent() {
    const response = await upstream.get("/contest/current");
    if (response.status !== 200) {
      throw new Error("Failed to fetch contest");
    }

    const raw = response.data?.payload ?? response.data;

    // Backend already sends nested currentSong — pass through, coerce runId to string
    if (raw && typeof raw === "object" && "currentSong" in raw) {
      const payload = raw as Record<string, unknown>;
      payload.runId = String(payload.runId ?? "");
      return payload as any;
    }

    // Legacy flat shape — reshape into { currentSong: Song, ... }
    if (raw && typeof raw === "object" && "songId" in raw) {
      const { runId, currentIndex, totalSongs, songId, songName, youtubeUrl,
        countryId, countryName, artistId, artistFirstName, artistLastName,
        artistType, publicVotes, juryVotes, totalVotes, votingIsOpen,
        ...rest } = raw as Record<string, unknown>;
      return {
        runId: String(runId ?? ""),
        currentIndex,
        totalSongs,
        contestActive: true,
        currentSong: {
          songId,
          songName,
          youtubeUrl,
          countryId,
          countryName,
          artistFirstName,
          artistLastName,
          publicVotes,
          juryVotes,
          totalVotes,
          votingIsOpen
        },
        ...rest
      };
    }

    return raw;
  }

  async startContest() {
    const response = await upstream.post("/admin/startContest", null);
    if (response.status >= 200 && response.status < 300) {
      return response.data?.payload;
    }
    throw new Error("Failed to start contest");
  }

  async advanceContest() {
    const response = await upstream.post("/admin/advanceContest", null);
    if (response.status >= 200 && response.status < 300) {
      return response.data?.payload;
    }
    throw new Error("Failed to advance contest");
  }

  async addCountry(countryId: string, countryName: string, pot: number) {
    const response = await upstream.post("/admin/addCountry/", null, {
      params: {
        ID: countryId,
        Name: countryName,
        Pot: pot
      }
    });
    if (response.status >= 200 && response.status < 300) {
      return { message: "Country added" };
    }
    throw new Error(response.data?.error ?? "Failed to add country");
  }

  async addArtist(firstName: string, lastName: string, countryId: string) {
    const response = await upstream.post("/admin/addArtist/", null, {
      params: {
        vorName: firstName,
        Name: lastName,
        typ: "solo",
        Land: countryId
      }
    });
    if (response.status >= 200 && response.status < 300) {
      return { message: "Artist added" };
    }
    throw new Error(response.data?.error ?? "Failed to add artist");
  }

  async addSong(input: {
    countryId: string;
    countryName?: string;
    songName: string;
    artistFirstName?: string;
    artistLastName?: string;
    youtubeUrl?: string;
  }) {
    const response = await upstream.post("/admin/addSong/", null, {
      params: {
        SongName: input.songName,
        CountryID: input.countryId,
        YoutubeURL: input.youtubeUrl
      }
    });
    if (response.status >= 200 && response.status < 300) {
      return response.data?.payload;
    }
    throw new Error(response.data?.error ?? "Failed to add song");
  }
}
