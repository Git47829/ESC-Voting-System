import { useCookieConsent } from "../context/CookieConsentContext";

export const CookieSettingsPage = () => {
  const { consent, saveConsent, clear } = useCookieConsent();

  return (
    <section className="space-y-4">
      <h1 className="text-3xl font-bold">Cookie Settings</h1>
      <div className="border border-esc-muted p-4 text-sm">
        <p>Verarbeitete Daten:</p>
        <ul className="mt-2 list-disc space-y-1 pl-6">
          <li>IP-Adresse</li>
          <li>Telefonnummer</li>
          <li>Für wen gestimmt wurde</li>
        </ul>
      </div>
      <div className="border border-esc-muted p-4 text-sm">
        <label className="mb-2 flex items-center gap-2">
          <input type="checkbox" checked readOnly />
          Required vote cookies (always active)
        </label>
        <label className="mb-2 flex items-center gap-2">
          <input
            type="checkbox"
            checked={consent?.preferences.statistics ?? false}
            onChange={(e) => saveConsent(e.target.checked)}
          />
          Statistics cookies
        </label>
        <div className="flex gap-2">
          <button className="border border-esc-yellow px-3 py-1 text-esc-yellow" onClick={() => saveConsent(true)}>
            Save
          </button>
          <button className="border border-red-500 px-3 py-1 text-red-400" onClick={clear}>
            Delete optional cookies
          </button>
        </div>
      </div>
    </section>
  );
};

