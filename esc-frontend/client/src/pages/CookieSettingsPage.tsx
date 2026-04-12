import { useCookieConsent } from "../context/CookieConsentContext";

export const CookieSettingsPage = () => {
  const { consent, saveConsent, clear } = useCookieConsent();

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Privacy</p>
        <h1 className="mt-2 text-4xl font-bold text-esc-black">Cookie Settings</h1>
        <p className="mt-3 max-w-3xl text-sm leading-7 text-esc-black-soft/78 sm:text-base">
          Verwalten Sie Ihre Cookie-Einstellungen für diese Anwendung. Notwendige Vote-Cookies bleiben aktiv, optionale Statistics-Cookies können Sie jederzeit ändern.
        </p>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Verarbeitete Daten</p>
        <ul className="mt-3 list-disc space-y-1.5 pl-6 text-sm text-esc-black-soft">
          <li>IP-Adresse</li>
          <li>Telefonnummer</li>
          <li>Für wen gestimmt wurde</li>
        </ul>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Cookie-Kategorien</p>

        <div className="mt-4 space-y-3">
          <div className="flex items-start justify-between gap-4 rounded-xl border border-esc-border bg-white px-4 py-3">
            <div>
              <p className="font-semibold text-esc-black">Notwendige Vote-Cookies</p>
              <p className="mt-1 text-sm text-esc-muted">Erforderlich für Login, Abstimmung, Session und Schutz vor Mehrfachabgabe.</p>
            </div>
            <span className="rounded-full border border-esc-border px-2.5 py-1 text-xs font-semibold uppercase tracking-[0.12em] text-esc-muted">
              Immer aktiv
            </span>
          </div>

          <div className="flex items-start justify-between gap-4 rounded-xl border border-esc-border bg-white px-4 py-3">
            <div>
              <p className="font-semibold text-esc-black">Statistics-Cookies</p>
              <p className="mt-1 text-sm text-esc-muted">Optionale Analysefunktionen für die Verbesserung der Anwendung.</p>
            </div>
            <label className="inline-flex items-center gap-2 text-sm text-esc-black-soft">
              <input
                type="checkbox"
                checked={consent?.preferences.statistics ?? false}
                onChange={(e) => saveConsent(e.target.checked)}
              />
              Aktiv
            </label>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-2.5">
          <button
            className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors hover:bg-esc-pink-dim"
            onClick={() => saveConsent(true)}
          >
            Alle akzeptieren
          </button>
          <button
            className="rounded-xl border border-esc-border px-4 py-2 text-sm font-medium text-esc-black-soft transition-colors hover:border-esc-border-strong hover:text-esc-black"
            onClick={() => saveConsent(false)}
          >
            Nur notwendige
          </button>
          <button
            className="rounded-xl border border-red-500 px-4 py-2 text-sm font-medium text-red-500 transition-colors hover:bg-red-50"
            onClick={clear}
          >
            Optionale Cookies löschen
          </button>
        </div>
      </div>
    </section>
  );
};
