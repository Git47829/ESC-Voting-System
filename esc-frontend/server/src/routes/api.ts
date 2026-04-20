import axios from "axios";
import { Router } from "express";
import rateLimit from "express-rate-limit";

import { config, isMockMode } from "../config.js";
import { requireRole } from "../middleware/auth.js";
import { mockDataService } from "../mock/index.js";
import { parseConsentCookie, upstream } from "../upstream.js";
import type { Song } from "../types.js";

const authHeaders = (session: { token?: string; email?: string }) => ({
  authorization: `Bearer ${session.token ?? ""}`,
  "X-Email": session.email ?? ""
});

export const apiRouter = Router();

const authLimiter = rateLimit({
  windowMs: 15 * 60 * 1000,
  max: 15,
  standardHeaders: true,
  legacyHeaders: false,
  message: { error: "Too many authentication attempts, please try again later" }
});

const authorizedLimiter = rateLimit({
  windowMs: 15 * 60 * 1000,
  max: 100,
  standardHeaders: true,
  legacyHeaders: false,
  message: { error: "Too many requests, please try again later" }
});

const decodeVoteStateCookie = (
  cookieValue: string
): { votes_remaining: number; votes_cast: Record<string, number> } | null => {
  try {
    const raw = Buffer.from(cookieValue, "hex");
    const sep = raw.lastIndexOf(0x2e); // '.'
    if (sep === -1) return null;
    const payload = raw.subarray(0, sep).toString("utf-8");
    return JSON.parse(payload) as { votes_remaining: number; votes_cast: Record<string, number> };
  } catch {
    return null;
  }
};

const toInt = (value: unknown, fallback = 0): number => {
  const parsed = Number.parseInt(String(value ?? ""), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

const juryPointValues = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12] as const;

const getJuryVoteState = (session: {
  token?: string;
  juryVoteState?: { token: string; votesCast: Record<number, number> };
}) => {
  const token = session.token ?? "";
  if (!session.juryVoteState || session.juryVoteState.token !== token) {
    session.juryVoteState = {
      token,
      votesCast: {}
    };
  }
  return session.juryVoteState;
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

apiRouter.post("/login", authLimiter, async (req, res) => {
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

apiRouter.post("/auth/login", authLimiter, async (req, res) => {
  const email = String(req.body?.email ?? "").trim();
  const password = String(req.body?.password ?? "").trim();
  const role = String(req.body?.role ?? "") as "admin" | "jury";
  if (!email || !password || (role !== "admin" && role !== "jury")) {
    res.status(422).json({ error: "Email, password, and valid role are required" });
    return;
  }

  if (isMockMode()) {
    const isValid =
      (role === "admin" && password === "admin-token") ||
      (role === "jury" && password === "jury-token");
    if (!isValid) {
      res.status(403).json({ error: "Invalid mock credentials" });
      return;
    }
    req.session.pendingEmail = email;
    req.session.pendingRole = role;
    req.session.pendingPassword = password;
    res.json({ message: "Verification code sent" });
    return;
  }

  const response = await upstream.post("/auth/login", { email, password, role });
  if (response.status === 202) {
    req.session.pendingEmail = email;
    req.session.pendingRole = role;
    req.session.pendingPassword = password;
  }
  res.status(response.status).json(response.data);
});

apiRouter.post("/auth/verify", authLimiter, async (req, res) => {
  const email = String(req.body?.email ?? "").trim();
  const code = String(req.body?.code ?? "").trim();
  const role = String(req.body?.role ?? "") as "admin" | "jury";
  if (!email || !code) {
    res.status(422).json({ error: "Email and code are required" });
    return;
  }

  if (isMockMode()) {
    if (req.session.pendingEmail === email) {
      req.session.role = req.session.pendingRole as "admin" | "jury";
      req.session.token = req.session.pendingPassword ?? "mock-2fa-token";
      req.session.email = email;
      delete req.session.pendingEmail;
      delete req.session.pendingRole;
      delete req.session.pendingPassword;
      res.json({ ok: true, role: req.session.role });
      return;
    }
    res.status(401).json({ error: "Invalid or expired verification code" });
    return;
  }

  const response = await upstream.post("/auth/verify", { email, code });
  if (response.status === 202) {
    req.session.role = role;
    req.session.token = req.session.pendingPassword ?? "";
    req.session.email = email;
    delete req.session.pendingEmail;
    delete req.session.pendingRole;
    delete req.session.pendingPassword;
    res.json({ ok: true, role });
    return;
  }
  res.status(response.status).json(response.data);
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
  if (response.status === 200 && Array.isArray((response.data as Record<string, unknown>)?.payload)) {
    const raw = (response.data as { payload: Song[] }).payload;
    const seen = new Map<number, Song>();
    for (const song of raw) {
      if (!seen.has(song.songId)) {
        seen.set(song.songId, song);
      }
    }
    res.json({ message: "Success", payload: [...seen.values()] });
    return;
  }
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
  const response = await upstream.get("/contest/current");
  if (response.status !== 200) {
    res.status(response.status).json(response.data);
    return;
  }

  const raw = response.data?.payload ?? response.data;

  // Backend already sends nested currentSong — pass through, coerce runId to string
  if (raw && typeof raw === "object" && "currentSong" in raw) {
    const payload = raw as Record<string, unknown>;
    payload.runId = String(payload.runId ?? "");
    res.json({ payload });
    return;
  }

  // Legacy flat shape — reshape into { currentSong: Song, ... }
  if (raw && typeof raw === "object" && "songId" in raw) {
    const { runId, currentIndex, totalSongs, songId, songName, youtubeUrl,
      countryId, countryName, artistId, artistFirstName, artistLastName,
      artistType, publicVotes, juryVotes, totalVotes, votingIsOpen,
      ...rest } = raw as Record<string, unknown>;
    res.json({
      payload: {
        runId: String(runId ?? ""), currentIndex, totalSongs,
        contestActive: true,
        currentSong: {
          songId, songName, youtubeUrl, countryId, countryName,
          artistFirstName, artistLastName,
          publicVotes, juryVotes, totalVotes, votingIsOpen
        },
        ...rest
      }
    });
    return;
  }

  res.json(response.data);
});

apiRouter.get("/vote/state", (req, res) => {
  let state = req.session.voteState;
  if (!state) {
    const raw = req.cookies?.vote_state as string | undefined;
    if (raw) {
      const decoded = decodeVoteStateCookie(raw);
      if (decoded) {
        state = {
          votesRemaining: decoded.votes_remaining,
          votesCast: decoded.votes_cast
        };
        req.session.voteState = state;
      }
    }
  }
  res.json(state ?? { votesRemaining: config.totalVotePoints, votesCast: {} });
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
      res.json({
        message: "Vote submitted",
        payload: voteState
      });
    } catch (error) {
      res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
    }
    return;
  }

  const songID = toInt(req.body?.songID);
  const points = toInt(req.body?.points, 1);
  const state = req.session.voteState ?? {
    votesRemaining: config.totalVotePoints,
    votesCast: {}
  };

  if (points > state.votesRemaining) {
    res.status(403).json({
      error: `Not enough vote points remaining (have ${state.votesRemaining}, requested ${points})`,
      votes_remaining: state.votesRemaining
    });
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

  if (response.status === 201) {
    state.votesRemaining -= points;
    state.votesCast[songID] = (state.votesCast[songID] ?? 0) + points;
    req.session.voteState = state;
  }

  res.status(response.status).json(
    response.status === 201
      ? { ...response.data, payload: state }
      : response.data
  );
});

apiRouter.post("/jury/vote", authorizedLimiter, requireRole("jury"), async (req, res) => {
  const songID = toInt(req.body?.songID);
  const points = toInt(req.body?.points);
  const juryVoteState = getJuryVoteState(req.session);

  if (!songID || !juryPointValues.includes(points as (typeof juryPointValues)[number])) {
    res.status(422).json({ error: "Invalid jury vote payload" });
    return;
  }

  if (juryVoteState.votesCast[songID] !== undefined) {
    res.status(409).json({ error: "This song already has a jury vote from this session." });
    return;
  }

  if (Object.values(juryVoteState.votesCast).includes(points)) {
    res.status(409).json({ error: `${points} points were already used for another song in this session.` });
    return;
  }

  if (isMockMode()) {
    try {
      mockDataService.castJuryVote(songID, points);
      juryVoteState.votesCast[songID] = points;
      req.session.juryVoteState = juryVoteState;
      res.json({ message: "Jury vote submitted" });
    } catch (error) {
      res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
    }
    return;
  }

  const response = await upstream.post("/jury/vote/", null, {
    params: {
      songID,
      points
    },
    headers: authHeaders(req.session)
  });
  if (response.status >= 200 && response.status < 300) {
    juryVoteState.votesCast[songID] = points;
    req.session.juryVoteState = juryVoteState;
  }
  res.status(response.status).json(response.data);
});

apiRouter.get("/jury/vote/state", authorizedLimiter, requireRole("jury"), (req, res) => {
  const state = getJuryVoteState(req.session);
  res.json({ payload: { votesCast: state.votesCast } });
});

apiRouter.get("/admin/authenticate", authLimiter, async (req, res) => {
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

apiRouter.get("/jury/authenticate", authLimiter, async (req, res) => {
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

apiRouter.post("/admin/open", authorizedLimiter, requireRole("admin"), async (_req, res) => {
  if (isMockMode()) {
    mockDataService.setVotingOpen(true);
    res.json({ message: "Voting opened" });
    return;
  }
  const response = await upstream.post("/admin/open", null, { headers: authHeaders(_req.session) });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/close", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.setVotingOpen(false);
    res.json({ message: "Voting closed" });
    return;
  }
  const response = await upstream.post("/admin/close", null, { headers: authHeaders(req.session) });
  res.status(response.status).json(response.data);
});

apiRouter.delete("/admin/deleteVotes", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.resetVotes();
    req.session.voteState = { votesRemaining: config.totalVotePoints, votesCast: {} };
    res.json({ message: "Votes reset" });
    return;
  }
  const response = await upstream.delete("/admin/deleteVotes/", { headers: authHeaders(req.session) });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addCountry", authorizedLimiter, requireRole("admin"), async (req, res) => {
  const pot = toInt(req.body?.pot, 1);
  if (isMockMode()) {
    mockDataService.addCountry(String(req.body?.countryId ?? ""), String(req.body?.countryName ?? ""), pot);
    res.json({ message: "Country added" });
    return;
  }
  const response = await upstream.post("/admin/addCountry/", null, {
    params: {
      ID: req.body?.countryId,
      Name: req.body?.countryName,
      Pot: pot
    },
    headers: authHeaders(req.session)
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addArtist", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    mockDataService.addArtist();
    res.json({ message: "Artist added" });
    return;
  }
  const response = await upstream.post("/admin/addArtist/", null, {
    params: {
      vorName: req.body?.firstName,
      Name: req.body?.lastName,
      typ: "solo",
      Land: req.body?.countryId
    },
    headers: authHeaders(req.session)
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/addSong", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    const songs = mockDataService.getSongs();
    const country = songs.find((entry) => entry.countryId === String(req.body?.countryId)) ?? songs[0];
    const artistId = String(req.body?.artistId ?? "").trim();
    const song = mockDataService.addSong({
      countryId: String(req.body?.countryId),
      countryName: country?.countryName ?? String(req.body?.countryId),
      songName: String(req.body?.songName),
      artistFirstName: "Artist",
      artistLastName: artistId !== "" ? `#${artistId}` : "Unknown",
      youtubeUrl: normalizeYoutubeUrl(String(req.body?.youtubeUrl ?? ""))
    });
    res.json({ message: "Song added", payload: song });
    return;
  }
  const response = await upstream.post("/admin/addSong/", null, {
    params: {
      SongName: req.body?.songName,
      CountryID: req.body?.countryId,
      KuenstlerID: req.body?.artistId,
      YoutubeURL: req.body?.youtubeUrl
    },
    headers: authHeaders(req.session)
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/startContest", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.startContest() });
    return;
  }
  const response = await upstream.post("/admin/startContest", null, {
    headers: authHeaders(req.session)
  });
  res.status(response.status).json(response.data);
});

apiRouter.post("/admin/advanceContest", authorizedLimiter, requireRole("admin"), async (req, res) => {
  if (isMockMode()) {
    res.json({ payload: mockDataService.advanceContest() });
    return;
  }
  const response = await upstream.post("/admin/advanceContest", null, {
    headers: authHeaders(req.session)
  });
  res.status(response.status).json(response.data);
});

apiRouter.get("/results", async (_req, res) => {
  if (isMockMode()) {
    res.json(mockDataService.getVotes());
    return;
  }

  const escResponse = await axios.get(`${config.escConverterUrl}/api/esc-points`, {
    timeout: config.apiTimeout,
    validateStatus: () => true
  });
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
  const response = await upstream.get("/votes/");
  if (response.status !== 200) {
    res.status(response.status).json(response.data);
    return;
  }
  const votes = ((response.data?.payload ?? []) as Array<{
    songId: number; countryId: string; countryName: string;
    publicVotes: number; juryVotes: number;
  }>);
  const totalPublic = votes.reduce((sum, v) => sum + (v.publicVotes ?? 0), 0);
  const totalJury = votes.reduce((sum, v) => sum + (v.juryVotes ?? 0), 0);
  const byCountry = votes
    .map((v) => ({
      countryId: v.countryId,
      country: v.countryName,
      total: (v.publicVotes ?? 0) + (v.juryVotes ?? 0)
    }))
    .sort((a, b) => b.total - a.total || a.country.localeCompare(b.country));
  res.json({ totalPublic, totalJury, byCountry });
});
