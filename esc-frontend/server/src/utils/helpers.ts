export const authHeaders = (session: { token?: string; email?: string }) => ({
  authorization: `Bearer ${session.token ?? ""}`,
  "X-Email": session.email ?? ""
});

export const decodeVoteStateCookie = (
  cookieValue: string
): { votes_remaining: number; votes_cast: Record<string, number> } | null => {
  try {
    const raw = Buffer.from(cookieValue, "hex");
    const sep = raw.lastIndexOf(0x2e); // '.'
    if (sep === -1) return null;
    const payload = raw.subarray(0, sep).toString("utf-8");
    return JSON.parse(payload) as { votes_remaining: number; votes_cast: Record<string, number> };
  } catch {
    return null;
  }
};

export const toInt = (value: unknown, fallback = 0): number => {
  const parsed = Number.parseInt(String(value ?? ""), 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

export const normalizeYoutubeUrl = (url: string): string => {
  if (!url) return url;
  const embedMatch = /youtube\.com\/embed\/([A-Za-z0-9_-]{11})/.exec(url);
  if (embedMatch) return `https://www.youtube.com/embed/${embedMatch[1]}`;
  const shortMatch = /youtu\.be\/([A-Za-z0-9_-]{11})/.exec(url);
  if (shortMatch) return `https://www.youtube.com/embed/${shortMatch[1]}`;
  const watchMatch = /youtube\.com\/(?:watch\?(?:.*&)?v=|shorts\/)([A-Za-z0-9_-]{11})/.exec(url);
  if (watchMatch) return `https://www.youtube.com/embed/${watchMatch[1]}`;
  return url;
};

export const juryPointValues = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12] as const;

export const getJuryVoteState = (session: {
  token?: string;
  juryVoteState?: { token: string; votesCast: Record<number, number> };
}) => {
  const token = session.token ?? "";
  if (!session.juryVoteState || session.juryVoteState.token !== token) {
    session.juryVoteState = {
      token,
      votesCast: {}
    };
  }
  return session.juryVoteState;
};
