import { Router } from "express";

import { config, isMockMode } from "../config.js";
import { requireRole } from "../middleware/auth.js";
import { mockDataService } from "../mock/index.js";
import { parseConsentCookie, upstream } from "../upstream.js";
import type { Song } from "../types.js";

export const apiRouter = Router();

const toInt = (value: unknown, fallback = 0): number => {
  const parsed = Number.parseInt(String(value ?? ""), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const normalizeYoutubeUrl = (url: string): string => {
  if (!url) return url;
  const embedMatch = /youtube\.com\/embed\/([A-Za-z0-9_-]{11})/.exec(url);
  if (embedMatch) return `https://www.youtube.com/embed/${embedMatch[1]}`;
  const shortMatch = /youtu\.be\/([A-Za-z0-9_-]{11})/.exec(url);
  if (shortMatch) return `https://www.youtube.com/embed/${shortMatch[1]}`;
  const watchMatch = /youtube\.com\/(?:watch\?(?:.*&)?v=|shorts\/)([A-Za-z0-9_-]{11})/.exec(url);
  if (watchMatch) return `https://www.youtube.com/embed/${watchMatch[1]}`;
  return url;
};

apiRouter.get("/session", (req, res) => {
  res.json({
    role: req.session.role ?? null,
    token: req.session.token ?? null,
    authenticated: Boolean(req.session.role && req.session.token)
  });
});

apiRouter.post("/login", async (req, res) => {
  const role = String(req.body?.role ?? "") as "admin" | "jury";
  const token = String(req.body?.token ?? "").trim();
  if (!token || (role !== "admin" && role !== "jury")) {
    res.status(422).json({ error: "Token and valid role are required" });
    return;
  }

  if (isMockMode()) {
    const isValid = (role === "admin" && token === "admin-token") || (role === "jury" && token === "jury-token");
    if (!isValid) {
      res.status(403).json({ error: "Invalid mock credentials" });
      return;
    }
    req.session.role = role;
    req.session.token = token;
    res.json({ ok: true, role });
    return;
  }

  const endpoint = role === "admin" ? "/admin/authenticate" : "/jury/authenticate";
  const response = await upstream.get(endpoint, { params: { Token: token } });
  if (response.status !== 202) {
    res.status(response.status).json(response.data);
    return;
  }
  req.session.role = role;
  req.session.token = token;
  res.json({ ok: true, role });
});

apiRouter.post("/logout", (req, res) => {
  req.session.destroy(() => {
    res.json({ ok: true });
  });
});

apiRouter.get("/songs", async (_req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.getSongs() });
    return;
  }
  const response = await upstream.get("/songs/");
  res.status(response.status).json(response.data);
});

apiRouter.get("/votes", async (_req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.getVotes() });
    return;
  }
  const response = await upstream.get("/votes/");
  res.status(response.status).json(response.data);
});

apiRouter.get("/countries", async (_req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.getCountries() });
    return;
  }
  const response = await upstream.get("/countries/");
  res.status(response.status).json(response.data);
});

apiRouter.get("/contest/current", async (_req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.getContestCurrent() });
    return;
  }
  const response = await upstream.get("/contest/current/");
  res.status(response.status).json(response.data);
});

apiRouter.post("/vote", async (req, res) => {
  const essentialConsent = parseConsentCookie(req.headers.cookie);
  if (!essentialConsent) {
    res.status(403).json({ error: "Please accept required vote cookies before submitting votes." });
    return;
  }

  if (isMockMode()) {
    const songID = toInt(req.body?.songID);
    const points = toInt(req.body?.points, 1);
    try {
      const state = req.session.voteState ?? {
        votesRemaining: config.totalVotePoints,
        votesCast: {}
      };
      const { voteState } = mockDataService.castPublicVote(songID, points, state);
      req.session.voteState = voteState;
      res.cookie("vote_state", JSON.stringify(voteState), {
        maxAge: 365 * 24 * 60 * 60 * 1000,
        httpOnly: true,
        sameSite: "strict"
      });
      res.json({
        message: "Vote submitted",
        payload: voteState
      });
    } catch (error) {
      res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
    }
    return;
  }

  const response = await upstream.post("/vote/", null, {
    params: {
      songID: req.body?.songID,
      phoneNum: req.body?.phoneNum,
      ownCountry: req.body?.ownCountry,
      points: req.body?.points
    },
    headers: {
      cookie: req.headers.cookie ?? ""
    }
  });
  const setCookie = response.headers["set-cookie"];
  if (setCookie) {
    res.setHeader("set-cookie", setCookie);
  }
  res.status(response.status).json(response.data);
});

apiRouter.post("/jury/vote", requireRole("jury"), async (req, res) => {
  if (isMockMode()) {
    const token = req.session.token ?? "jury-token";
    const songID = toInt(req.body?.songID);
    const points = toInt(req.body?.points, 12);
    const cookieKey = `jury_votes_${token}`;
    const already = req.session.juryVotes?.[`${cookieKey}_${songID}`];
    if (already) {
      res.status(409).json({ error: "Duplicate jury vote detected" });
      return;
    }
    try {
      mockDataService.castJuryVote(songID, points);
      req.session.juryVotes = {
        ...(req.session.juryVotes ?? {}),
        [`${cookieKey}_${songID}`]: true
      };
      res.json({ message: "Jury vote submitted" });
    } catch (error) {
      res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
    }
    return;
  }

  const response = await upstream.post("/jury/vote/", null, {
    params: {
      songID: req.body?.songID,
      points: req.body?.points
    },
    headers: {
      authorization: `Bearer ${req.session.token ?? ""}`
    }
  });
  res.status(response.status).json(response.data);
});

apiRouter.get("/admin/authenticate", async (req, res) => {
  const token = String(req.query.Token ?? "");
  if (isMockMode()) {
    res.status(token === "admin-token" ? 202 : 403).json(
      token === "admin-token" ? { message: "ok" } : { error: "Invalid admin token" }
    );
    return;
  }
  const response = await upstream.get("/admin/authenticate", { params: { Token: token } });
  res.status(response.status).json(response.data);
});

apiRouter.get("/jury/authenticate", async (req, res) => {
  const token = String(req.query.Token ?? "");
  if (isMockMode()) {
    res.status(token === "jury-token" ? 202 : 403).json(
      token === "jury-token" ? { message: "ok" } : { error: "Invalid jury token" }
    );
    return;
  }
  const response = await upstream.get("/jury/authenticate", { params: { Token: token } });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/open", requireRole("admin"), async (_req, res) => {
  if (isMockMode()) {
    mockDataService.setVotingOpen(true);
    res.json({ message: "Voting opened" });
    return;
  }
  const response = await upstream.post("/admin/open/", null, { params: { Token: _req.session.token } });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/close", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.setVotingOpen(false);
    res.json({ message: "Voting closed" });
    return;
  }
  const response = await upstream.post("/admin/close", null, { params: { Token: req.session.token } });
  res.status(response.status).json(response.data);
});

apiRouter.delete("/admin/deleteVotes", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.resetVotes();
    req.session.voteState = { votesRemaining: config.totalVotePoints, votesCast: {} };
    res.json({ message: "Votes reset" });
    return;
  }
  const response = await upstream.delete("/admin/deleteVotes/", { params: { Token: req.session.token } });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addCountry", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.addCountry(String(req.body?.countryId ?? ""), String(req.body?.countryName ?? ""));
    res.json({ message: "Country added" });
    return;
  }
  const response = await upstream.post("/admin/addCountry/", null, {
    params: {
      Token: req.session.token,
      ID: req.body?.countryId,
      Name: req.body?.countryName
    }
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addArtist", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.addArtist();
    res.json({ message: "Artist added" });
    return;
  }
  const response = await upstream.post("/admin/addArtist/", null, {
    params: {
      Token: req.session.token,
      FirstName: req.body?.firstName,
      LastName: req.body?.lastName,
      CountryID: req.body?.countryId
    }
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addSong", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    const songs = mockDataService.getSongs();
    const country = songs.find((entry) => entry.countryId === String(req.body?.countryId)) ?? songs[0];
    const song = mockDataService.addSong({
      countryId: String(req.body?.countryId),
      countryName: country?.countryName ?? String(req.body?.countryId),
      songName: String(req.body?.songName),
      artistFirstName: String(req.body?.artistFirstName),
      artistLastName: String(req.body?.artistLastName),
      youtubeUrl: normalizeYoutubeUrl(String(req.body?.youtubeUrl ?? ""))
    });
    res.json({ message: "Song added", payload: song });
    return;
  }
  const response = await upstream.post("/admin/addSong/", null, {
    params: {
      Token: req.session.token,
      SongName: req.body?.songName,
      CountryID: req.body?.countryId,
      KuenstlerID: req.body?.artistId,
      YoutubeURL: req.body?.youtubeUrl
    }
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/startContest", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.startContest() });
    return;
  }
  const response = await upstream.post("/admin/startContest/", null, {
    params: { Token: req.session.token }
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/advanceContest", requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.advanceContest() });
    return;
  }
  const response = await upstream.post("/admin/advanceContest/", null, {
    params: { Token: req.session.token }
  });
  res.status(response.status).json(response.data);
});

apiRouter.get("/results", async (_req, res) => {
  if (isMockMode()) {
    res.json(mockDataService.getVotes());
    return;
  }

  const escResponse = await upstream.get(`${config.escConverterUrl}/api/esc-points`);
  if (escResponse.status >= 400) {
    res.status(escResponse.status).json({ error: "ESC converter unavailable" });
    return;
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
  res.json(results.map((entry, index) => ({ ...entry, rank: index + 1 })));
});

apiRouter.get("/stats", async (_req, res) => {
  if (isMockMode()) {
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

    res.json({
      totalPublic,
      totalJury,
      byCountry
    });
    return;
  }
  const response = await upstream.get(`${config.eurostatsUrl}/votes/subscribe`);
  res.status(response.status).json(response.data);
});

