import type { VotingService } from "./interfaces.js";
import { config } from "../config.js";
import { upstream } from "../upstream.js";
import { authHeaders, toInt } from "../utils/helpers.js";

export class ProductionVotingService implements VotingService {
  async castPublicVote(
    songId: number,
    points: number,
    sessionCookie?: string,
    additionalParams?: Record<string, unknown>
  ) {
    const response = await upstream.post("/vote/", null, {
      params: {
        songID: songId,
        points,
        ...(additionalParams || {})
      },
      headers: {
        cookie: sessionCookie ?? ""
      }
    });
    if (response.status === 201) {
      return {
        message: "Vote submitted",
        payload: {
          votesRemaining: (response.data?.payload?.votesRemaining as number) ?? 0,
          votesCast: (response.data?.payload?.votesCast as Record<number, number>) ?? {}
        }
      };
    }
    throw new Error(response.data?.error ?? "Vote failed");
  }

  async castJuryVote(songId: number, points: number) {
    const response = await upstream.post("/jury/vote/", null, {
      params: { songID: songId, points }
    });
    if (response.status >= 200 && response.status < 300) {
      return { message: "Jury vote submitted" };
    }
    throw new Error(response.data?.error ?? "Jury vote failed");
  }

  async getVotes() {
    const response = await upstream.get("/votes/");
    if (response.status === 200) {
      return response.data?.payload ?? [];
    }
    throw new Error("Failed to fetch votes");
  }

  async setVotingOpen(open: boolean) {
    const endpoint = open ? "/admin/open" : "/admin/close";
    const response = await upstream.post(endpoint, null);
    if (response.status >= 200 && response.status < 300) {
      return { message: open ? "Voting opened" : "Voting closed" };
    }
    throw new Error(response.data?.error ?? "Failed to update voting status");
  }

  async resetVotes() {
    const response = await upstream.delete("/admin/deleteVotes/");
    if (response.status >= 200 && response.status < 300) {
      return { message: "Votes reset" };
    }
    throw new Error(response.data?.error ?? "Failed to reset votes");
  }
}
