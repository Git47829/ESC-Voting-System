import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from "react";

import { clearConsent, readConsent, type ConsentState, writeConsent } from "../utils/cookieConsent";

interface CookieConsentContextValue {
  consent: ConsentState | null;
  shouldShowBanner: boolean;
  saveConsent: (statistics: boolean) => void;
  clear: () => void;
}

const CookieConsentContext = createContext<CookieConsentContextValue | null>(null);

export const CookieConsentProvider = ({ children }: { children: ReactNode }) => {
  const [consent, setConsent] = useState<ConsentState | null>(null);

  useEffect(() => {
    setConsent(readConsent());
  }, []);

  const value = useMemo<CookieConsentContextValue>(
    () => ({
      consent,
      shouldShowBanner: !consent,
      saveConsent: (statistics) => {
        const next: ConsentState = {
          acceptedAt: new Date().toISOString(),
          version: 1,
          preferences: {
            essential: true,
            statistics
          }
        };
        writeConsent(next);
        setConsent(next);
      },
      clear: () => {
        clearConsent();
        setConsent(null);
      }
    }),
    [consent]
  );

  return (
    <CookieConsentContext.Provider value={value}>
      {children}
    </CookieConsentContext.Provider>
  );
};

export const useCookieConsent = (): CookieConsentContextValue => {
  const context = useContext(CookieConsentContext);
  if (!context) {
    throw new Error("useCookieConsent must be used in CookieConsentProvider");
  }
  return context;
};
