import axios from "axios";
import type { Request, Response, NextFunction } from "express";

export const errorHandler = (err: unknown, _req: Request, res: Response, _next: NextFunction) => {
  if (axios.isAxiosError(err)) {
    res.status(502).json({ error: "Upstream service unavailable" });
    return;
  }
  console.error("Unhandled error:", err);
  res.status(500).json({ error: "Internal server error" });
};
