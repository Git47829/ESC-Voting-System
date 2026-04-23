import crypto from "node:crypto";
import type { Request, Response, NextFunction } from "express";

const csrfExempt = new Set(["/api/login", "/api/auth/login", "/api/auth/verify", "/api/csrf-token", "/health"]);
const csrfSafeMethods = new Set(["GET", "HEAD", "OPTIONS"]);

export const csrfProtection = (req: Request, res: Response, next: NextFunction): void => {
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
};

export const csrfTokenEndpoint = (req: Request, res: Response) => {
  if (!req.session.csrfToken) {
    req.session.csrfToken = crypto.randomUUID();
  }
  res.json({ csrfToken: req.session.csrfToken });
};
