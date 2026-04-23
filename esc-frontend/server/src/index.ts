import path from "node:path";
import { fileURLToPath } from "node:url";

import express, { type Request, type Response, type NextFunction } from "express";
import cookieParser from "cookie-parser";
import cors from "cors";
import rateLimit from "express-rate-limit";
import session from "express-session";

import { config, isMockMode } from "./config.js";
import { apiRouter } from "./routes/api.js";
import { healthCheck } from "./routes/health.js";
import { csrfProtection, csrfTokenEndpoint } from "./middleware/csrf.js";
import { errorHandler } from "./middleware/error-handler.js";

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

// CORS middleware (development only)
if (!isProduction) {
  app.use(
    cors({
      origin: "http://localhost:5173",
      credentials: true
    })
  );
}

// Body parsing middleware
app.use(express.json());
app.use(express.urlencoded({ extended: true }));
app.use(cookieParser());

// Session middleware
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

app.use(csrfProtection);

app.get("/api/csrf-token", csrfTokenEndpoint);

app.get("/health", healthCheck);

app.use("/api", apiRouter);

// SPA fallback (production only)
if (isProduction) {
  const clientDist = path.resolve(__dirname, "../../client/dist");
  app.use(express.static(clientDist));
  app.get("*", spaFallbackRateLimiter, (_req, res) => {
    res.sendFile(path.join(clientDist, "index.html"));
  });
}

app.use(errorHandler);

app.listen(config.port, () => {
  // eslint-disable-next-line no-console
  console.log(`ESC server listening on http://localhost:${config.port} (mock=${isMockMode()})`);
});
