import { useState } from "react";
import { Link } from "react-router-dom";

import { useCookieConsent } from "../../context/CookieConsentContext";

export const CookieBanner = () => {
  const { consent, shouldShowBanner, saveConsent } = useCookieConsent();
  const [showDetails, setShowDetails] = useState(false);
  const statisticsEnabled = consent?.preferences.statistics ?? false;

  if (!shouldShowBanner) return null;

  const statisticsCardClass = statisticsEnabled
    ? "border-white/16 bg-esc-black text-white"
    : "border-black/8 bg-white/72 text-esc-black";

  const statisticsTitleClass = statisticsEnabled ? "text-white" : "text-esc-black";
  const statisticsBodyClass = statisticsEnabled ? "text-white/78" : "text-esc-muted";

  const statisticsOffButtonClass = statisticsEnabled
    ? "border-white/24 text-white/88 hover:border-white/50 hover:text-white"
    : "border-black/10 text-esc-black-soft hover:border-esc-pink hover:text-esc-pink";

  const detailsDividerClass = statisticsEnabled ? "border-white/12" : "border-black/8";

  return (
    <aside
      className="fixed bottom-24 right-4 z-50 w-[min(28rem,calc(100vw-2rem))] sm:right-5"
      role="dialog"
      aria-label="Cookie preferences"
    >
      <div className="relative overflow-hidden rounded-[1.65rem] border border-black/8 bg-[linear-gradient(180deg,rgba(255,255,255,0.98),rgba(255,249,252,0.96))] p-5 shadow-[0_24px_54px_rgba(0,0,0,0.16)] backdrop-blur-xl sm:p-6">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,4,144,0.1),transparent_28%)]" />

        <div className="relative space-y-2">
          <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Cookie settings</p>
          <h2 className="text-2xl font-semibold tracking-[-0.04em] text-esc-black">
            Your privacy preferences
          </h2>
          <p className="text-sm leading-6 text-esc-black-soft/80">
            Required cookies keep voting, login and session protection working. Optional statistics
            cookies help us improve the experience.
          </p>
        </div>

        <div className="relative mt-5 flex flex-wrap gap-2.5">
          <button
            className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim"
            onClick={() => saveConsent(true)}
          >
            Accept all
          </button>

          <button
            className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-medium text-esc-black-soft transition-colors duration-200 hover:border-black/20 hover:text-esc-black"
            onClick={() => saveConsent(false)}
          >
            Essential only
          </button>

          <button
            className="rounded-xl border border-black/10 bg-white px-4 py-2 text-sm font-medium text-esc-muted transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
            onClick={() => setShowDetails((current) => !current)}
            aria-expanded={showDetails}
          >
            {showDetails ? "Hide details" : "Show details"}
          </button>
        </div>

        <div className="relative mt-4">
          <Link to="/cookies" className="text-sm font-medium text-esc-black-soft transition-colors hover:text-esc-pink">
            Review full cookie settings
          </Link>
        </div>

        {showDetails ? (
          <div className={`relative mt-5 grid gap-3 border-t pt-4 text-sm ${detailsDividerClass}`}>
            <div className="flex items-start justify-between gap-4 rounded-[1.2rem] border border-black/8 bg-white/72 px-4 py-3">
              <div>
                <p className="font-semibold text-esc-black">Required voting cookies</p>
                <p className="mt-1 text-esc-muted">
                  Needed for login, voting, session state and duplicate-vote protection.
                </p>
              </div>

              <span className="rounded-full border border-black/8 bg-white px-2.5 py-1 text-[11px] font-semibold uppercase tracking-[0.12em] text-esc-muted">
                Always on
              </span>
            </div>

            <div
              className={`flex items-start justify-between gap-4 rounded-[1.2rem] border px-4 py-3 transition-colors ${statisticsCardClass}`}
            >
              <div>
                <p className={`font-semibold ${statisticsTitleClass}`}>Statistics cookies</p>
                <p className={`mt-1 ${statisticsBodyClass}`}>
                  Optional usage measurement to improve the application.
                </p>
              </div>

              <div className="flex gap-2">
                <button
                  className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${statisticsOffButtonClass}`}
                  onClick={() => saveConsent(false)}
                >
                  Off
                </button>

                <button
                  className="rounded-lg border border-esc-pink bg-esc-pink px-3 py-1.5 text-xs font-semibold text-white transition-colors hover:bg-esc-pink-dim"
                  onClick={() => saveConsent(true)}
                >
                  On
                </button>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </aside>
  );
};
