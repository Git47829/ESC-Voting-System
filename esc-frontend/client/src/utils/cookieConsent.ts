export interface ConsentState {
  acceptedAt: string;
  version: number;
  preferences: {
    essential: boolean;
    statistics: boolean;
  };
}

const KEY = "esc_cookie_consent";
const MAX_AGE_DAYS = 180;

export const readConsent = (): ConsentState | null => {
  const entry = document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith(`${KEY}=`));
  if (!entry) return null;
  try {
    return JSON.parse(decodeURIComponent(entry.split("=")[1])) as ConsentState;
  } catch {
    return null;
  }
};

export const writeConsent = (value: ConsentState): void => {
  const expires = new Date(Date.now() + MAX_AGE_DAYS * 24 * 60 * 60 * 1000).toUTCString();
  document.cookie = `${KEY}=${encodeURIComponent(JSON.stringify(value))}; path=/; expires=${expires}; SameSite=Lax`;
};

export const clearConsent = (): void => {
  document.cookie = `${KEY}=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax`;
};

