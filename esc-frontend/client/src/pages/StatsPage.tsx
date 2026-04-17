import { useCallback, useEffect, useRef, useState } from "react";

type ConnectionStatus = "connecting" | "live" | "reconnecting";

interface StatsMessage {
  type: "stats";
  vote_count: number;
  charts: {
    voters_by_country: string;
    votes_received_by_country: string;
  } | null;
}

const SAFE_DATA_URL = /^data:image\/(png|jpeg|gif|webp|svg\+xml);base64,[A-Za-z0-9+/=]+$/;
const safeImageSrc = (value: unknown): string =>
  typeof value === "string" && SAFE_DATA_URL.test(value) ? value : "";

export const StatsPage = () => {
  const [status, setStatus] = useState<ConnectionStatus>("connecting");
  const [voteCount, setVoteCount] = useState(0);
  const [charts, setCharts] = useState<StatsMessage["charts"]>(null);
  const [hasData, setHasData] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  const delayRef = useRef(1000);
  const unmountedRef = useRef(false);

  const connect = useCallback(() => {
    if (unmountedRef.current) return;

    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${protocol}//${window.location.host}/eurostats/ws/stats`);
    wsRef.current = ws;

    ws.onopen = () => {
      delayRef.current = 1000;
      setStatus("live");
    };

    ws.onmessage = (event) => {
      let msg: { type: string; vote_count?: number; charts?: StatsMessage["charts"] };
      try {
        msg = JSON.parse(event.data as string) as typeof msg;
      } catch {
        return;
      }
      if (msg.type === "ping") return;
      if (msg.type === "stats") {
        setVoteCount(Number.isFinite(msg.vote_count) ? msg.vote_count! : 0);
        setCharts(msg.charts ?? null);
        setHasData(true);
      }
    };

    ws.onclose = () => {
      if (unmountedRef.current) return;
      setStatus("reconnecting");
      setTimeout(connect, delayRef.current);
      delayRef.current = Math.min(delayRef.current * 2, 30_000);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, []);

  useEffect(() => {
    unmountedRef.current = false;
    connect();
    return () => {
      unmountedRef.current = true;
      wsRef.current?.close();
    };
  }, [connect]);

  const showCharts = hasData && charts && voteCount > 0;

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative isolate overflow-hidden px-4 pb-10 pt-8 sm:px-6 sm:pb-12 lg:px-8">
        <div className="pointer-events-none absolute inset-0 vote-section-wash" />
        <div className="pointer-events-none absolute inset-0 vote-grid-lines" />
        <div className="pointer-events-none absolute inset-0 vote-noise opacity-60 mix-blend-soft-light" />
        <div className="pointer-events-none absolute left-[-22%] top-[4%] h-[54rem] w-[92rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.4)_0%,_rgba(255,4,144,0.22)_30%,_rgba(255,4,144,0.08)_52%,_rgba(255,4,144,0)_80%)] blur-[190px] opacity-94" />
        <div className="pointer-events-none absolute right-[-20%] top-[16%] h-[48rem] w-[80rem] rounded-full bg-[radial-gradient(ellipse_at_center,_rgba(255,4,144,0.28)_0%,_rgba(255,4,144,0.15)_34%,_rgba(255,4,144,0.06)_54%,_rgba(255,4,144,0)_82%)] blur-[182px] opacity-86" />

        <div className="relative z-10 mx-auto max-w-7xl space-y-6">
          {/* Header */}
          <div className="vote-panel rounded-[2rem] p-6 shadow-[0_18px_44px_rgba(0,0,0,0.2)] sm:p-8">
            <p className="text-xs uppercase tracking-[0.16em] text-esc-pink/72">Analytics</p>
            <h1 className="mt-2 text-4xl font-bold text-esc-black">Vote Statistics</h1>
            <p className="mt-2 text-sm text-esc-muted">
              Live charts — updated in real-time as votes arrive.
            </p>
          </div>

          {/* Status bar */}
          <div className="vote-panel-soft rounded-[1.5rem] px-6 py-4 shadow-[0_14px_30px_rgba(0,0,0,0.18)]">
            <div className="flex items-center justify-between">
              <span className="inline-flex items-center gap-2 text-xs text-esc-muted">
                <span className="relative flex h-2 w-2">
                  {status === "live" ? (
                    <>
                      <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-green-400 opacity-75" />
                      <span className="relative inline-flex h-2 w-2 rounded-full bg-green-400" />
                    </>
                  ) : (
                    <span className="relative inline-flex h-2 w-2 rounded-full bg-esc-muted" />
                  )}
                </span>
                {status === "connecting" && "Connecting\u2026"}
                {status === "live" && "Live"}
                {status === "reconnecting" && "Reconnecting\u2026"}
              </span>
              <span className="text-xs text-esc-muted">
                Total votes processed:{" "}
                <span className="font-bold text-esc-pink">{voteCount}</span>
              </span>
            </div>
          </div>

          {/* Loading state */}
          {!hasData && status !== "reconnecting" && (
            <div className="vote-panel rounded-[1.75rem] p-8 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
              <div className="flex flex-col items-center justify-center py-12">
                <svg
                  className="mb-4 h-8 w-8 animate-spin text-esc-pink"
                  xmlns="http://www.w3.org/2000/svg"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z" />
                </svg>
                <p className="text-sm text-esc-muted">Connecting to live stats&hellip;</p>
              </div>
            </div>
          )}

          {/* Disconnected state */}
          {hasData && status === "reconnecting" && (
            <div className="vote-panel rounded-[1.75rem] p-8 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <p className="mb-2 text-xl font-semibold text-red-500">Disconnected</p>
                <p className="max-w-sm text-sm text-esc-muted">
                  Lost connection to the stats service. Reconnecting automatically&hellip;
                </p>
              </div>
            </div>
          )}

          {/* Empty state */}
          {hasData && voteCount === 0 && (
            <div className="vote-panel rounded-[1.75rem] p-8 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
              <div className="flex flex-col items-center justify-center py-12 text-center">
                <p className="mb-2 text-xl font-semibold text-esc-black">No Votes Yet</p>
                <p className="max-w-sm text-sm text-esc-muted">
                  Charts will appear here once votes are cast.
                </p>
              </div>
            </div>
          )}

          {/* Charts */}
          {showCharts && (
            <div className="grid gap-6 lg:grid-cols-2">
              <div className="vote-panel rounded-[1.75rem] p-5 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
                <img
                  src={safeImageSrc(charts.voters_by_country)}
                  alt="Voters by Country"
                  className="h-auto w-full rounded-xl"
                />
              </div>
              <div className="vote-panel rounded-[1.75rem] p-5 shadow-[0_16px_36px_rgba(0,0,0,0.2)]">
                <img
                  src={safeImageSrc(charts.votes_received_by_country)}
                  alt="Votes Received by Country"
                  className="h-auto w-full rounded-xl"
                />
              </div>
            </div>
          )}
        </div>
      </div>
    </section>
  );
};
