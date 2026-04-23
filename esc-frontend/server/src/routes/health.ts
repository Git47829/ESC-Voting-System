import type { Request, Response } from "express";
import { isMockMode } from "../config.js";

export const healthCheck = (_req: Request, res: Response) => {
  res.status(200).json({ status: "healthy", service: "esc-frontend-server", mock: isMockMode() });
};
