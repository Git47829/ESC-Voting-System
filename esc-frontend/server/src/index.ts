import path from "node:path";
import { fileURLToPath } from "node:url";

import cookieParser from "cookie-parser";
import { RedisStore } from "connect-redis";
import cors from "cors";
import express from "express";
import rateLimit from "express-rate-limit";
import session from "express-session";
import Redis from "ioredis";
import lusca from "lusca";

import { config, isMockMode } from "./config.js";
import { apiRouter } from "./routes/api.js";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const isProduction = config.nodeEnv === "production";

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
// Redis-backed session store for horizontal scaling
const redisUrl = process.env.REDIS_URL ?? "redis://redis:6379";
const redisClient = new Redis.default(redisUrl);
redisClient.on("error", (err: Error) => console.error("Redis session error:", err));
redisClient.on("connect", () => console.log("Redis session store connected"));

app.use(
  session({
    store: new RedisStore({ client: redisClient }),
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

if (!isProduction) {
  app.use(lusca.csrf());
  app.get("/api/csrf-token", (req, res) => {
    res.status(200).json({ csrfToken: (req as unknown as { csrfToken(): string }).csrfToken() });
  });
}

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

app.listen(config.port, () => {
  // eslint-disable-next-line no-console
  console.log(`ESC server listening on http://localhost:${config.port} (mock=${isMockMode()})`);
});

