import { useEffect, useState } from "react";

import { api } from "../api/client";
import type { VoteResult } from "../types";

export const useResultsPoll = () => {
  const [results, setResults] = useState<VoteResult[]>([]);
  const [countdown, setCountdown] = useState(10);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    const fetchResults = async () => {
      const next = await api.getResults();
      setResults(next);
      setCountdown(10);
    };

    void fetchResults();

    const secondTick = window.setInterval(() => {
      setCountdown((current) => (current <= 1 ? 10 : current - 1));
    }, 1000);

    const dataTick = window.setInterval(() => {
      if (!paused) {
        void fetchResults();
      }
    }, 10000);

    return () => {
      window.clearInterval(secondTick);
      window.clearInterval(dataTick);
    };
  }, [paused]);

  return {
    results,
    countdown,
    paused,
    setPaused
  };
};

