import type { Request, Response, NextFunction } from "express";

export const requireRole = (role: "admin" | "jury") => {
  return (req: Request, res: Response, next: NextFunction): void => {
    const currentRole = req.session.role;
    if (!currentRole) {
      res.status(401).json({ error: "Authentication required" });
      return;
    }
    if (currentRole !== role) {
      res.status(403).json({ error: "Forbidden" });
      return;
    }
    next();
  };
};

