import { useEffect, useRef, useState } from "react";

import { api } from "../api/client";
import type { ContestState } from "../types";

export const useContestPoll = () => {
  const [contest, setContest] = useState<ContestState | null>(null);
  const previousRef = useRef<{ runId: string; currentIndex: number } | null>(null);

  useEffect(() => {
    const load = async () => {
      const next = await api.getContestCurrent();
      const previous = previousRef.current;
      if (
        previous &&
        (previous.runId !== next.runId || previous.currentIndex !== next.currentIndex)
      ) {
        window.location.reload();
      }
      previousRef.current = { runId: next.runId, currentIndex: next.currentIndex };
      setContest(next);
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, 5000);
    return () => window.clearInterval(timer);
  }, []);

  return contest;
};

