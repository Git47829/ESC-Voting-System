import "express-session";

declare module "express-session" {
  interface SessionData {
    role?: "admin" | "jury";
    token?: string;
    pendingEmail?: string;
    pendingRole?: string;
    voteState?: {
      votesRemaining: number;
      votesCast: Record<number, number>;
    };
    juryVotes?: Record<string, boolean>;
  }
}

