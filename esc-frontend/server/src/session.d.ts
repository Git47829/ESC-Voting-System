import "express-session";

interface AuthSession {
  role?: "admin" | "jury";
  token?: string;
  email?: string;
  pendingEmail?: string;
  pendingRole?: "admin" | "jury";
  pendingPassword?: string;
}

interface VoteSession {
  voteState?: {
    votesRemaining: number;
    votesCast: Record<number, number>;
  };
  juryVoteState?: {
    token: string;
    votesCast: Record<number, number>;
  };
}

interface SecuritySession {
  csrfToken?: string;
}

declare module "express-session" {
  interface SessionData extends AuthSession, VoteSession, SecuritySession {
  }
}
