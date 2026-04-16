import { useState } from "react";

import { useCookieConsent } from "../../context/CookieConsentContext";

export const CookieBanner = () => {
  const { consent, shouldShowBanner, saveConsent } = useCookieConsent();
  const [showDetails, setShowDetails] = useState(false);
  const statisticsEnabled = consent?.preferences.statistics ?? false;

  if (!shouldShowBanner) return null;

  const statisticsCardClass = statisticsEnabled
    ? "border-white/20 bg-esc-black text-white"
    : "border-esc-border bg-white/62 text-esc-black";

  const statisticsTitleClass = statisticsEnabled ? "text-white" : "text-esc-black";
  const statisticsBodyClass = statisticsEnabled ? "text-white/78" : "text-esc-muted";

  const statisticsOffButtonClass = statisticsEnabled
    ? "border-white/30 text-white/88 hover:border-white/60 hover:text-white"
    : "border-esc-border text-esc-black-soft hover:border-esc-pink hover:text-esc-pink";

  const detailsDividerClass = statisticsEnabled ? "border-white/12" : "border-esc-border";

  return (
    <aside
      className="fixed bottom-24 right-4 z-50 w-[min(26rem,calc(100vw-2rem))] sm:bottom-24 sm:right-5"
      role="dialog"
      aria-label="Cookie preferences"
    >
      <div className="relative overflow-hidden rounded-[1.5rem] border border-esc-border-strong bg-[#ffffff] p-5 shadow-[0_22px_48px_rgba(0,0,0,0.16)] sm:p-6">
        <div className="relative">
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">
              Cookie settings
            </p>
            <h2 className="text-2xl font-bold text-esc-black">
              Ihre Privatsphäre-Einstellungen
            </h2>
            <p className="text-sm leading-6 text-esc-black-soft/80">
              Wir verwenden notwendige Vote-Cookies, damit die Abstimmung zuverlässig
              funktioniert. Optionale Statistics-Cookies helfen uns, die Nutzererfahrung zu
              verbessern.
            </p>
          </div>

          <div className="mt-5 flex flex-wrap gap-2.5">
            <button
              className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim"
              onClick={() => saveConsent(true)}
            >
              Alle akzeptieren
            </button>

            <button
              className="rounded-xl border border-esc-border px-4 py-2 text-sm font-medium text-esc-black-soft transition-colors duration-200 hover:border-esc-border-strong hover:text-esc-black"
              onClick={() => saveConsent(false)}
            >
              Nur notwendige
            </button>

            <button
              className="rounded-xl border border-esc-border px-4 py-2 text-sm font-medium text-esc-muted transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
              onClick={() => setShowDetails((current) => !current)}
              aria-expanded={showDetails}
            >
              {showDetails ? "Einstellungen ausblenden" : "Einstellungen anzeigen"}
            </button>
          </div>

          {showDetails ? (
            <div className={`mt-5 grid gap-3 border-t pt-4 text-sm ${detailsDividerClass}`}>
              <div className="flex items-start justify-between gap-4 rounded-xl border border-esc-border bg-white/62 px-4 py-3">
                <div>
                  <p className="font-semibold text-esc-black">Notwendige Vote-Cookies</p>
                  <p className="mt-1 text-esc-muted">
                    Erforderlich für Login, Abstimmung und Sitzungsstatus.
                  </p>
                </div>

                <span className="rounded-full border border-esc-border px-2.5 py-1 text-xs font-semibold uppercase tracking-[0.12em] text-esc-muted">
                  Immer aktiv
                </span>
              </div>

              <div
                className={`flex items-start justify-between gap-4 rounded-xl border px-4 py-3 transition-colors ${statisticsCardClass}`}
              >
                <div>
                  <p className={`font-semibold ${statisticsTitleClass}`}>
                    Statistics-Cookies
                  </p>
                  <p className={`mt-1 ${statisticsBodyClass}`}>
                    Optionale Analyse der Nutzung zur Verbesserung der Anwendung.
                  </p>
                </div>

                <div className="flex gap-2">
                  <button
                    className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${statisticsOffButtonClass}`}
                    onClick={() => saveConsent(false)}
                  >
                    Aus
                  </button>

                  <button
                    className="rounded-lg border border-esc-pink bg-esc-pink px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-esc-pink-dim"
                    onClick={() => saveConsent(true)}
                  >
                    An
                  </button>
                </div>
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </aside>
  );
};