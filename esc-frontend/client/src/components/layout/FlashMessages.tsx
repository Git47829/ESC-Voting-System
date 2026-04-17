import { useFlash } from "../../context/FlashContext";

export const FlashMessages = () => {
  const { messages, removeFlash } = useFlash();

  return (
    <div aria-live="polite" className="fixed right-4 top-4 z-50 flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-3">
      {messages.map((message) => (
        <div
          key={message.id}
          role={message.category === "error" ? "alert" : "status"}
          className={`flash-fade rounded-2xl border px-4 py-3 text-left text-sm shadow-[0_18px_44px_rgba(17,17,17,0.16)] backdrop-blur-sm ${
            message.category === "success"
              ? "border-emerald-500/60 bg-emerald-500/20"
              : message.category === "error"
                ? "border-red-500/80 bg-red-600/20"
                : "border-esc-yellow/70 bg-esc-yellow/20"
          }`}
        >
          <div className="flex items-start gap-3">
            <div className="min-w-0 flex-1">
              <p className="text-[11px] font-semibold uppercase tracking-[0.12em] text-esc-black/75">
                {message.category === "error"
                  ? "Backend error"
                  : message.category === "success"
                    ? "Success"
                    : "Notice"}
              </p>
              <p className="mt-1 break-words text-sm text-esc-black">{message.message}</p>
            </div>
            <button
              type="button"
              onClick={() => removeFlash(message.id)}
              className="rounded-md border border-esc-border/70 px-2 py-1 text-xs text-esc-black/70 transition-colors hover:border-esc-pink hover:text-esc-pink"
              aria-label="Dismiss message"
            >
              ✕
            </button>
          </div>
        </div>
      ))}
    </div>
  );
};
