import { useCallback, useEffect, useRef, useState } from "react";

import { ApiError, api } from "../api/client";
import type { ContestState } from "../types";

export const useContestPoll = () => {
  const [contest, setContest] = useState<ContestState | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [contestEnded, setContestEnded] = useState(false);
  const previousRef = useRef<{ runId: string; currentIndex: number } | null>(null);
  const hasActiveContestRef = useRef(false);

  const load = useCallback(async () => {
    try {
      const next = await api.getContestCurrent();
      if (!next?.contestActive || !next.currentSong) {
        setContest(next);
        setError(null);
        previousRef.current = null;
        setContestEnded(hasActiveContestRef.current);
        hasActiveContestRef.current = false;
        return;
      }

      previousRef.current = { runId: next.runId, currentIndex: next.currentIndex };
      setContest(next);
      setContestEnded(false);
      setError(null);
      hasActiveContestRef.current = true;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Could not reach backend";
      if (err instanceof ApiError && err.status === 404) {
        setContest(null);
        setError(null);
        previousRef.current = null;
        setContestEnded(hasActiveContestRef.current);
        hasActiveContestRef.current = false;
        return;
      }

      if (!hasActiveContestRef.current) {
        setContest(null);
        setError(message);
      }
    }
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => {
      void load();
    }, 5000);
    return () => window.clearInterval(timer);
  }, [load]);

  return { contest, error, contestEnded };
};
