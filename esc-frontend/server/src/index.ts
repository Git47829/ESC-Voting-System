import path from "node:path";
import { fileURLToPath } from "node:url";

import "./observability.js";

import express, { type Request, type Response, type NextFunction } from "express";
import cookieParser from "cookie-parser";
import cors from "cors";
import rateLimit from "express-rate-limit";
import session from "express-session";
import { trace } from "@opentelemetry/api";

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

// Request logging middleware for observability
app.use((req: Request, res: Response, next: NextFunction) => {
  const tracer = trace.getTracer("esc-frontend-request-logger");
  const span = tracer.startSpan(`${req.method} ${req.path}`);

  const startTime = Date.now();

  res.on("finish", () => {
    const duration = Date.now() - startTime;
    span.setAttributes({
      "http.method": req.method,
      "http.url": req.originalUrl,
      "http.target": req.path,
      "http.status_code": res.statusCode,
      "http.response_content_length": res.get("content-length") ?? 0
    });
    span.addEvent("http.request_complete", {
      "http.duration_ms": duration
    });
    span.end();

    if (!isProduction) {
      // eslint-disable-next-line no-console
      console.log(`[${req.method}] ${req.path} - ${res.statusCode} (${duration}ms)`);
    }
  });

  next();
});

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
