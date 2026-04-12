import { useCookieConsent } from "../../context/CookieConsentContext";

export const CookieBanner = () => {
  const { shouldShowBanner, saveConsent } = useCookieConsent();
  if (!shouldShowBanner) return null;

  return (
    <aside className="fixed bottom-4 left-4 right-4 z-50 rounded-2xl border border-esc-border bg-white/95 p-4 text-sm text-esc-black shadow-[0_14px_40px_rgba(0,0,0,0.12)] backdrop-blur-sm">
      <p className="mb-3">
        Wir verwenden notwendige Vote-Cookies (nicht deaktivierbar) und optionale Statistics-Cookies.
      </p>
      <div className="flex gap-2">
        <button
          className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-1.5 font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim"
          onClick={() => saveConsent(true)}
        >
          Accept all
        </button>
        <button
          className="rounded-xl border border-esc-border px-3 py-1.5 text-esc-black-soft transition-colors duration-200 hover:border-esc-border-strong hover:text-esc-black"
          onClick={() => saveConsent(false)}
        >
          Only required
        </button>
      </div>
    </aside>
  );
};