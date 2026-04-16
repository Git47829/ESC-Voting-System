import cookieParser from "cookie-parser";
import cors from "cors";
import express from "express";
import session from "express-session";
import lusca from "lusca";

import { config, isMockMode } from "./config.js";
import { apiRouter } from "./routes/api.js";

const app = express();

app.use(
  cors({
    origin: "http://localhost:5173",
    credentials: true
  })
);
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
      secure: false,
      maxAge: 1000 * 60 * 60 * 24
    }
  })
);
app.use(lusca.csrf());

app.get("/health", (_req, res) => {
  res.status(200).json({ status: "healthy", service: "esc-frontend-server", mock: isMockMode() });
});

app.get("/api/csrf-token", (req, res) => {
  res.status(200).json({ csrfToken: req.csrfToken() });
});

app.use("/api", apiRouter);

app.listen(config.port, () => {
  // eslint-disable-next-line no-console
  console.log(`ESC server listening on http://localhost:${config.port} (mock=${isMockMode()})`);
});

