import { useCookieConsent } from "../context/CookieConsentContext";

const dataPoints = [
  "IP address",
  "Phone number used for voting",
  "Selected entries and assigned points",
  "Session and consent preferences"
];

export const CookieSettingsPage = () => {
  const { consent, saveConsent, clear } = useCookieConsent();
  const statisticsEnabled = consent?.preferences.statistics ?? false;

  return (
    <section className="pb-16 pt-8 sm:pt-10">
      <div className="relative overflow-hidden rounded-[2.3rem] border border-black/8 bg-[linear-gradient(180deg,rgba(255,248,252,0.98),rgba(255,255,255,0.96))] p-6 shadow-[0_22px_54px_rgba(17,17,17,0.06)] sm:p-8 lg:p-10">
        <div className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,4,144,0.09),transparent_28%),radial-gradient(circle_at_bottom_left,rgba(255,4,144,0.06),transparent_26%)]" />
        <div className="pointer-events-none absolute inset-0 opacity-[0.22] [background-image:linear-gradient(rgba(17,17,17,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(17,17,17,0.05)_1px,transparent_1px)] [background-size:8rem_8rem]" />

        <div className="relative grid gap-8 lg:grid-cols-[minmax(0,1.15fr)_22rem] lg:items-start">
          <div>
            <p className="text-[11px] uppercase tracking-[0.22em] text-esc-muted">Privacy</p>
            <h1 className="mt-3 text-4xl font-semibold tracking-[-0.05em] text-esc-black sm:text-5xl">
              Cookie settings
            </h1>
            <p className="mt-5 max-w-3xl text-sm leading-7 text-esc-black-soft/78 sm:text-base">
              Review how cookies are used across the voting experience. Required cookies stay on so
              voting, login and session protection keep working, while optional statistics cookies
              can be changed at any time.
            </p>
          </div>

          <aside className="rounded-[1.7rem] border border-black/8 bg-white/80 p-5 shadow-[0_18px_40px_rgba(17,17,17,0.05)] backdrop-blur-sm sm:p-6">
            <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Current setup</p>
            <div className="mt-5 space-y-4">
              <div className="flex items-center justify-between gap-4 border-b border-black/8 pb-4">
                <div>
                  <p className="text-sm font-semibold text-esc-black">Required cookies</p>
                  <p className="mt-1 text-sm text-esc-black-soft/72">Always enabled for voting and account flows.</p>
                </div>
                <span className="rounded-full border border-black/8 bg-white px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-esc-black-soft">
                  Active
                </span>
              </div>

              <div className="flex items-center justify-between gap-4">
                <div>
                  <p className="text-sm font-semibold text-esc-black">Statistics cookies</p>
                  <p className="mt-1 text-sm text-esc-black-soft/72">Optional insights to improve the experience.</p>
                </div>
                <span
                  className={`rounded-full px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] ${
                    statisticsEnabled
                      ? "bg-esc-pink text-white"
                      : "border border-black/8 bg-white text-esc-black-soft"
                  }`}
                >
                  {statisticsEnabled ? "Enabled" : "Off"}
                </span>
              </div>
            </div>
          </aside>
        </div>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-[minmax(0,0.82fr)_minmax(0,1.18fr)]">
        <article className="rounded-[1.8rem] border border-black/8 bg-white/90 p-6 shadow-[0_18px_40px_rgba(17,17,17,0.05)] sm:p-7">
          <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Data used</p>
          <h2 className="mt-3 text-2xl font-semibold tracking-[-0.04em] text-esc-black">
            What may be processed
          </h2>
          <p className="mt-3 text-sm leading-7 text-esc-black-soft/75">
            These data points support the vote flow, fraud prevention and optional usage analysis.
          </p>

          <ul className="mt-5 space-y-3">
            {dataPoints.map((point) => (
              <li key={point} className="flex items-start gap-3 text-sm text-esc-black-soft">
                <span className="mt-1.5 h-2 w-2 rounded-full bg-esc-pink" />
                <span>{point}</span>
              </li>
            ))}
          </ul>
        </article>

        <article className="rounded-[1.8rem] border border-black/8 bg-white/90 p-6 shadow-[0_18px_40px_rgba(17,17,17,0.05)] sm:p-7">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-[11px] uppercase tracking-[0.18em] text-esc-muted">Cookie categories</p>
              <h2 className="mt-3 text-2xl font-semibold tracking-[-0.04em] text-esc-black">
                Manage your preferences
              </h2>
            </div>
            <p className="max-w-md text-sm leading-7 text-esc-black-soft/72">
              Required cookies cannot be turned off here because they are needed for core site functions.
            </p>
          </div>

          <div className="mt-6 space-y-3">
            <div className="flex items-start justify-between gap-4 rounded-[1.3rem] border border-black/8 bg-[#fcfcfc] px-4 py-4">
              <div>
                <p className="font-semibold text-esc-black">Required voting cookies</p>
                <p className="mt-1 text-sm text-esc-muted">
                  Needed for login, secure voting, session state and duplicate-vote protection.
                </p>
              </div>
              <span className="rounded-full border border-black/8 bg-white px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-esc-black-soft">
                Always on
              </span>
            </div>

            <div
              className={`flex items-start justify-between gap-4 rounded-[1.3rem] border px-4 py-4 transition-colors ${
                statisticsEnabled
                  ? "border-esc-pink/20 bg-[linear-gradient(180deg,rgba(20,20,20,0.97),rgba(39,29,34,0.95))] text-white"
                  : "border-black/8 bg-[#fcfcfc] text-esc-black"
              }`}
            >
              <div>
                <p className={`font-semibold ${statisticsEnabled ? "text-white" : "text-esc-black"}`}>
                  Statistics cookies
                </p>
                <p className={`mt-1 text-sm ${statisticsEnabled ? "text-white/78" : "text-esc-muted"}`}>
                  Optional measurement of site usage so we can improve clarity and performance.
                </p>
              </div>

              <label
                className={`inline-flex items-center gap-2 text-sm ${
                  statisticsEnabled ? "text-white" : "text-esc-black-soft"
                }`}
              >
                <input
                  type="checkbox"
                  checked={statisticsEnabled}
                  onChange={(e) => saveConsent(e.target.checked)}
                  className={statisticsEnabled ? "accent-white" : "accent-esc-pink"}
                />
                <span>{statisticsEnabled ? "Enabled" : "Off"}</span>
              </label>
            </div>
          </div>

          <div className="mt-6 flex flex-wrap gap-3">
            <button
              className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim"
              onClick={() => saveConsent(true)}
            >
              Accept all
            </button>
            <button
              className="rounded-xl border border-black/10 bg-white px-4 py-2.5 text-sm font-medium text-esc-black-soft transition-colors hover:border-black/20 hover:text-esc-black"
              onClick={() => saveConsent(false)}
            >
              Essential only
            </button>
            <button
              className="rounded-xl border border-black/10 bg-white px-4 py-2.5 text-sm font-medium text-esc-black-soft transition-colors hover:border-red-300 hover:text-red-500"
              onClick={clear}
            >
              Clear optional cookies
            </button>
          </div>
        </article>
      </div>
    </section>
  );
};
