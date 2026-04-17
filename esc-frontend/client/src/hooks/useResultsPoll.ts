import { useCallback, useEffect, useRef, useState } from "react";

import { api } from "../api/client";
import type { VoteResult } from "../types";

export const useResultsPoll = () => {
  const [results, setResults] = useState<VoteResult[]>([]);
  const [countdown, setCountdown] = useState(10);
  const [paused, setPaused] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const hasLoadedDataRef = useRef(false);
  const pausedRef = useRef(false);
  const mountedRef = useRef(false);

  const fetchResults = useCallback(async () => {
    try {
      const next = await api.getResults();
      setResults(next);
      setError(null);
      hasLoadedDataRef.current = true;
    } catch (err) {
      if (!hasLoadedDataRef.current) {
        setError(err instanceof Error ? err.message : "Failed to load results");
      }
    }
    setCountdown(10);
  }, []);

  useEffect(() => {
    void fetchResults();

    const secondTick = window.setInterval(() => {
      if (pausedRef.current) {
        return;
      }
      setCountdown((current) => (current > 0 ? current - 1 : 0));
    }, 1000);

    const dataTick = window.setInterval(() => {
      if (!pausedRef.current) {
        void fetchResults();
      }
    }, 10000);

    return () => {
      window.clearInterval(secondTick);
      window.clearInterval(dataTick);
    };
  }, [fetchResults]);

  useEffect(() => {
    if (!mountedRef.current) {
      mountedRef.current = true;
      return;
    }
    pausedRef.current = paused;
    if (!paused) {
      setCountdown(10);
      void fetchResults();
    }
  }, [paused, fetchResults]);

  return {
    results,
    countdown,
    paused,
    setPaused,
    error
  };
};
