import "express-session";

declare module "express-session" {
  interface SessionData {
    role?: "admin" | "jury";
    token?: string;
    email?: string;
    pendingEmail?: string;
    pendingRole?: string;
    pendingPassword?: string;
    voteState?: {
      votesRemaining: number;
      votesCast: Record<number, number>;
    };
    juryVotes?: Record<string, boolean>;
    juryVoteState?: {
      token: string;
      votesCast: Record<number, number>;
    };
    csrfToken?: string;
  }
}
