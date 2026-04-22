import crypto from "node:crypto";
import path from "node:path";
import { fileURLToPath } from "node:url";

import axios from "axios";
import cookieParser from "cookie-parser";
import cors from "cors";
import express, { type Request, type Response, type NextFunction } from "express";
import rateLimit from "express-rate-limit";
import session from "express-session";

import { config, isMockMode } from "./config.js";
import { apiRouter } from "./routes/api.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isProduction = config.nodeEnv === "production";

if (isProduction && config.sessionSecret === "change-me-in-production") {
  throw new Error("SESSION_SECRET environment variable is required in production");
}

const app = express();

const spaFallbackRateLimiter = rateLimit({
  windowMs: 15 * 60 * 1000,
  max: 100,
  standardHeaders: true,
  legacyHeaders: false
});

if (isProduction) {
  app.set("trust proxy", 1);
}

if (!isProduction) {
  app.use(
    cors({
      origin: "http://localhost:5173",
      credentials: true
    })
  );
}

app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(cookieParser());
app.use(
  session({
    secret: config.sessionSecret,
    resave: false,
    saveUninitialized: false,
    cookie: {
      sameSite: "lax",
      secure: isProduction,
      maxAge: 1000 * 60 * 60 * 24
    }
  })
);

const csrfExempt = new Set(["/api/login", "/api/auth/login", "/api/auth/verify", "/api/csrf-token", "/health"]);
const csrfSafeMethods = new Set(["GET", "HEAD", "OPTIONS"]);

app.use((req, res, next) => {
  if (csrfSafeMethods.has(req.method) || csrfExempt.has(req.path)) {
    next();
    return;
  }
  const token = req.headers["x-csrf-token"] as string | undefined;
  if (!token || token !== req.session.csrfToken) {
    res.status(403).json({ error: "CSRF token invalid" });
    return;
  }
  next();
});

app.get("/api/csrf-token", (req, res) => {
  if (!req.session.csrfToken) {
    req.session.csrfToken = crypto.randomUUID();
  }
  res.json({ csrfToken: req.session.csrfToken });
});

app.get("/health", (_req, res) => {
  res.status(200).json({ status: "healthy", service: "esc-frontend-server", mock: isMockMode() });
});

app.use("/api", apiRouter);

if (isProduction) {
  const clientDist = path.resolve(__dirname, "../../client/dist");
  app.use(express.static(clientDist));
  app.get("*", spaFallbackRateLimiter, (_req, res) => {
    res.sendFile(path.join(clientDist, "index.html"));
  });
}

app.use((err: unknown, _req: Request, res: Response, _next: NextFunction) => {
  if (axios.isAxiosError(err)) {
    res.status(502).json({ error: "Upstream service unavailable" });
    return;
  }
  // eslint-disable-next-line no-console
  console.error("Unhandled error:", err);
  res.status(500).json({ error: "Internal server error" });
});

app.listen(config.port, () => {
  // eslint-disable-next-line no-console
  console.log(`ESC server listening on http://localhost:${config.port} (mock=${isMockMode()})`);
});

