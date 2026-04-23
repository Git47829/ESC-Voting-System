import { Router, type Request, type Response, type NextFunction } from "express";
import rateLimit from "express-rate-limit";
import { trace } from "@opentelemetry/api";

import { config } from "../config.js";
import { requireRole } from "../middleware/auth.js";
import { getAuthService, getContestService, getVotingService, getResultsService } from "../services/factory.js";
import { parseConsentCookie } from "../upstream.js";
import {
  authHeaders,
  decodeVoteStateCookie,
  toInt,
  getJuryVoteState,
  juryPointValues
} from "../utils/helpers.js";

const authService = getAuthService();
const contestService = getContestService();
const votingService = getVotingService();
const resultsService = getResultsService();

export const apiRouter = Router();

const asyncHandler = (fn: (req: Request, res: Response, next: NextFunction) => Promise<void>) =>
  (req: Request, res: Response, next: NextFunction) => { fn(req, res, next).catch(next); };

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

apiRouter.get("/session", (req, res) => {
  res.json({
    role: req.session.role ?? null,
    token: req.session.token ?? null,
    authenticated: Boolean(req.session.role && req.session.token)
  });
});

apiRouter.post("/login", authLimiter, asyncHandler(async (req, res) => {
  const tracer = trace.getTracer("esc-frontend-login");
  const span = tracer.startSpan("login_with_token");

  const role = String(req.body?.role ?? "") as "admin" | "jury";
  const token = String(req.body?.token ?? "").trim();

  span.setAttributes({
    "user.role": role
  });

  if (!token || (role !== "admin" && role !== "jury")) {
    span.setAttributes({
      "user.login_success": false,
      "user.login_error": "invalid_payload"
    });
    span.end();
    res.status(422).json({ error: "Token and valid role are required" });
    return;
  }

  try {
    const result = await authService.loginWithToken(role, token);
    if (!("role" in result)) {
      span.setAttributes({
        "user.login_success": false,
        "user.login_error": "invalid_credentials"
      });
      span.end();
      res.status(403).json({ error: "Invalid credentials" });
      return;
    }
    if (!result.ok) {
      span.setAttributes({
        "user.login_success": false,
        "user.login_error": "invalid_credentials"
      });
      span.end();
      res.status(403).json({ error: "Invalid credentials" });
      return;
    }
    req.session.role = role;
    req.session.token = token;
    span.setAttributes({
      "user.login_success": true,
      "user.role": role
    });
    span.end();
    req.session.save((err) => {
      if (err) {
        res.status(500).json({ error: "Session save failed" });
        return;
      }
      res.json({ ok: true, role });
    });
  } catch (error) {
    span.setAttributes({
      "user.login_success": false,
      "user.login_error": "exception",
      "error.message": error instanceof Error ? error.message : "unknown"
    });
    span.end();
    res.status(500).json({ error: "Authentication failed" });
  }
}));

apiRouter.post("/auth/login", authLimiter, asyncHandler(async (req, res) => {
  const email = String(req.body?.email ?? "").trim();
  const password = String(req.body?.password ?? "").trim();
  const role = String(req.body?.role ?? "") as "admin" | "jury";
  if (!email || !password || (role !== "admin" && role !== "jury")) {
    res.status(422).json({ error: "Email, password, and valid role are required" });
    return;
  }

  try {
    const result = await authService.loginWithEmail(email, password, role);
    if ("error" in result) {
      res.status(403).json({ error: result.error });
      return;
    }
    req.session.pendingEmail = email;
    req.session.pendingRole = role;
    req.session.pendingPassword = password;
    res.json({ message: "Verification code sent" });
  } catch (error) {
    res.status(500).json({ error: "Login failed" });
  }
}));

apiRouter.post("/auth/verify", authLimiter, asyncHandler(async (req, res) => {
  const email = String(req.body?.email ?? "").trim();
  const code = String(req.body?.code ?? "").trim();
  const role = req.session.pendingRole as "admin" | "jury" | undefined;
  if (!email || !code) {
    res.status(422).json({ error: "Email and code are required" });
    return;
  }
  if (!role) {
    res.status(409).json({ error: "No pending login — call /auth/login first" });
    return;
  }

  try {
    const result = await authService.verifyEmail(email, code, role);
    if ("error" in result) {
      res.status(401).json({ error: result.error });
      return;
    }
    req.session.role = role;
    req.session.token = req.session.pendingPassword ?? "mock-2fa-token";
    req.session.email = email;
    delete req.session.pendingEmail;
    delete req.session.pendingRole;
    delete req.session.pendingPassword;
    req.session.save((err) => {
      if (err) {
        res.status(500).json({ error: "Session save failed" });
        return;
      }
      res.json({ ok: true, role });
    });
  } catch (error) {
    res.status(401).json({ error: "Verification failed" });
  }
}));

apiRouter.post("/logout", (req, res) => {
  req.session.destroy(() => {
    res.json({ ok: true });
  });
});

apiRouter.get("/songs", asyncHandler(async (_req, res) => {
  const tracer = trace.getTracer("esc-frontend-contest");
  const span = tracer.startSpan("fetch_songs");

  try {
    const songs = await contestService.getSongs();
    span.setAttributes({
      "contest.operation": "get_songs",
      "contest.songs_count": songs?.length ?? 0
    });
    span.end();
    res.json({ message: "Success", payload: songs });
  } catch (error) {
    span.setAttributes({
      "contest.operation": "get_songs",
      "contest.error": "fetch_failed",
      "error.message": error instanceof Error ? error.message : "unknown"
    });
    span.end();
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch songs" });
  }
}));

apiRouter.get("/votes", asyncHandler(async (_req, res) => {
  try {
    const votes = await votingService.getVotes();
    res.json({ payload: votes });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch votes" });
  }
}));

apiRouter.get("/countries", asyncHandler(async (_req, res) => {
  try {
    const countries = await contestService.getCountries();
    res.json({ payload: countries });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch countries" });
  }
}));

apiRouter.get("/contest/current", asyncHandler(async (_req, res) => {
  try {
    const contestState = await contestService.getContestCurrent();
    res.json({ payload: contestState });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch contest" });
  }
}));

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

apiRouter.post("/vote", asyncHandler(async (req, res) => {
  const tracer = trace.getTracer("esc-frontend-voting");
  const span = tracer.startSpan("cast_public_vote");

  const essentialConsent = parseConsentCookie(req.headers.cookie);
  if (!essentialConsent) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "missing_consent"
    });
    span.end();
    res.status(403).json({ error: "Please accept required vote cookies before submitting votes." });
    return;
  }

  const songID = toInt(req.body?.songID);
  const points = toInt(req.body?.points, 1);
  const state = req.session.voteState ?? {
    votesRemaining: config.totalVotePoints,
    votesCast: {}
  };

  span.setAttributes({
    "vote.song_id": songID,
    "vote.points": points,
    "vote.votes_remaining": state.votesRemaining
  });

  if (points > state.votesRemaining) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "insufficient_points"
    });
    span.end();
    res.status(403).json({
      error: `Not enough vote points remaining (have ${state.votesRemaining}, requested ${points})`,
      votes_remaining: state.votesRemaining
    });
    return;
  }

  try {
    const result = await votingService.castPublicVote(
      songID,
      points,
      req.headers.cookie,
      {
        phoneNum: req.body?.phoneNum,
        ownCountry: req.body?.ownCountry
      }
    );
    const newState = result.payload;
    req.session.voteState = newState;
    span.setAttributes({
      "vote.success": true,
      "vote.votes_remaining_after": newState.votesRemaining
    });
    span.end();
    res.status(201).json({
      message: result.message,
      payload: newState
    });
  } catch (error) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "exception",
      "error.message": error instanceof Error ? error.message : "unknown"
    });
    span.end();
    res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
  }
}));

apiRouter.post("/jury/vote", authorizedLimiter, requireRole("jury"), asyncHandler(async (req, res) => {
  const tracer = trace.getTracer("esc-frontend-voting");
  const span = tracer.startSpan("cast_jury_vote");

  const songID = toInt(req.body?.songID);
  const points = toInt(req.body?.points);
  const juryVoteState = getJuryVoteState(req.session);

  span.setAttributes({
    "vote.type": "jury",
    "vote.song_id": songID,
    "vote.points": points
  });

  if (!songID || !juryPointValues.includes(points as (typeof juryPointValues)[number])) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "invalid_payload"
    });
    span.end();
    res.status(422).json({ error: "Invalid jury vote payload" });
    return;
  }

  if (juryVoteState.votesCast[songID] !== undefined) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "song_already_voted"
    });
    span.end();
    res.status(409).json({ error: "This song already has a jury vote from this session." });
    return;
  }

  if (Object.values(juryVoteState.votesCast).includes(points)) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "points_already_used"
    });
    span.end();
    res.status(409).json({ error: `${points} points were already used for another song in this session.` });
    return;
  }

  const creds = { token: req.session.token, email: req.session.email };
  try {
    await votingService.castJuryVote(songID, points, creds);
    juryVoteState.votesCast[songID] = points;
    req.session.juryVoteState = juryVoteState;
    span.setAttributes({
      "vote.success": true
    });
    span.end();
    res.json({ message: "Jury vote submitted" });
  } catch (error) {
    span.setAttributes({
      "vote.success": false,
      "vote.error": "exception",
      "error.message": error instanceof Error ? error.message : "unknown"
    });
    span.end();
    res.status(422).json({ error: error instanceof Error ? error.message : "Vote failed" });
  }
}));

apiRouter.get("/jury/vote/state", authorizedLimiter, requireRole("jury"), (req, res) => {
  const state = getJuryVoteState(req.session);
  res.json({ payload: { votesCast: state.votesCast } });
});

apiRouter.get("/admin/authenticate", authLimiter, asyncHandler(async (req, res) => {
  const token = String(req.query.Token ?? "");
  try {
    const result = await authService.authenticateToken(token, "admin");
    if ("error" in result) {
      res.status(403).json(result);
      return;
    }
    res.status(202).json({ message: "ok" });
  } catch (error) {
    res.status(403).json({ error: error instanceof Error ? error.message : "Authentication failed" });
  }
}));

apiRouter.get("/jury/authenticate", authLimiter, asyncHandler(async (req, res) => {
  const token = String(req.query.Token ?? "");
  try {
    const result = await authService.authenticateToken(token, "jury");
    if ("error" in result) {
      res.status(403).json(result);
      return;
    }
    res.status(202).json({ message: "ok" });
  } catch (error) {
    res.status(403).json({ error: error instanceof Error ? error.message : "Authentication failed" });
  }
}));

apiRouter.post("/admin/open", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const result = await votingService.setVotingOpen(true, creds);
    res.json(result);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to open voting" });
  }
}));

apiRouter.post("/admin/close", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const result = await votingService.setVotingOpen(false, creds);
    res.json(result);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to close voting" });
  }
}));

apiRouter.delete("/admin/deleteVotes", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    await votingService.resetVotes(creds);
    req.session.voteState = { votesRemaining: config.totalVotePoints, votesCast: {} };
    res.json({ message: "Votes reset" });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to reset votes" });
  }
}));

apiRouter.post("/admin/addCountry", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  const pot = toInt(req.body?.pot, 1);
  try {
    const result = await contestService.addCountry(
      String(req.body?.countryId ?? ""),
      String(req.body?.countryName ?? ""),
      pot,
      creds
    );
    res.json(result);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to add country" });
  }
}));

apiRouter.post("/admin/addArtist", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const result = await contestService.addArtist(
      String(req.body?.firstName ?? ""),
      String(req.body?.lastName ?? ""),
      String(req.body?.countryId ?? ""),
      creds
    );
    res.json(result);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to add artist" });
  }
}));

apiRouter.post("/admin/addSong", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const song = await contestService.addSong({
      countryId: String(req.body?.countryId ?? ""),
      songName: String(req.body?.songName ?? ""),
      youtubeUrl: String(req.body?.youtubeUrl ?? "")
    }, creds);
    res.json({ message: "Song added", payload: song });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to add song" });
  }
}));

apiRouter.post("/admin/startContest", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const contestState = await contestService.startContest(creds);
    res.json({ payload: contestState });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to start contest" });
  }
}));

apiRouter.post("/admin/advanceContest", authorizedLimiter, requireRole("admin"), asyncHandler(async (req, res) => {
  const creds = { token: req.session.token, email: req.session.email };
  try {
    const contestState = await contestService.advanceContest(creds);
    res.json({ payload: contestState });
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to advance contest" });
  }
}));

apiRouter.get("/results", asyncHandler(async (_req, res) => {
  try {
    const results = await resultsService.getResults();
    res.json(results);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch results" });
  }
}));

apiRouter.get("/stats", asyncHandler(async (_req, res) => {
  try {
    const stats = await resultsService.getStats();
    res.json(stats);
  } catch (error) {
    res.status(502).json({ error: error instanceof Error ? error.message : "Failed to fetch stats" });
  }
}));
