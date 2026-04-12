import { useCookieConsent } from "../context/CookieConsentContext";

export const CookieSettingsPage = () => {
  const { consent, saveConsent, clear } = useCookieConsent();

  return (
    <section className="space-y-6">
      <div className="rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.05)] sm:p-8">
        <h1 className="text-4xl font-bold text-esc-black">Cookie Settings</h1>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 text-sm shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Verarbeitete Daten</p>
        <ul className="mt-3 list-disc space-y-1.5 pl-6 text-esc-black-soft">
          <li>IP-Adresse</li>
          <li>Telefonnummer</li>
          <li>Für wen gestimmt wurde</li>
        </ul>
      </div>

      <div className="rounded-[1.75rem] border border-esc-border bg-white/92 p-5 text-sm shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-6">
        <label className="mb-3 flex items-center gap-2 text-esc-black-soft">
          <input type="checkbox" checked readOnly />
          Required vote cookies (always active)
        </label>
        <label className="mb-4 flex items-center gap-2 text-esc-black-soft">
          <input
            type="checkbox"
            checked={consent?.preferences.statistics ?? false}
            onChange={(e) => saveConsent(e.target.checked)}
          />
          Statistics cookies
        </label>
        <div className="flex gap-2">
          <button className="rounded-xl border border-esc-pink bg-esc-pink px-3 py-2 font-semibold text-white hover:bg-esc-pink-dim" onClick={() => saveConsent(true)}>
            Save
          </button>
          <button className="rounded-xl border border-red-500 px-3 py-2 text-red-500 hover:bg-red-50" onClick={clear}>
            Delete optional cookies
          </button>
        </div>
      </div>
    </section>
  );
};
