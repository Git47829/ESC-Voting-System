import axios, { type AxiosResponseHeaders, type RawAxiosResponseHeaders } from "axios";

import { config } from "./config.js";

export const upstream = axios.create({
  baseURL: config.apiBaseUrl,
  timeout: config.apiTimeout,
  validateStatus: () => true
});

export const passSetCookie = (
  headers: AxiosResponseHeaders | RawAxiosResponseHeaders,
  setCookie: ((name: string, val: string, options?: Record<string, unknown>) => unknown) | ((field: string, value?: string | string[]) => unknown)
): void => {
  const raw = headers["set-cookie"];
  if (raw && Array.isArray(raw) && raw.length > 0) {
    setCookie("set-cookie", raw as unknown as string);
  }
};

export const parseConsentCookie = (cookieHeader?: string): boolean => {
  if (!cookieHeader) {
    return false;
  }
  const match = cookieHeader.match(/esc_cookie_consent=([^;]+)/);
  if (!match) {
    return false;
  }
  try {
    const value = decodeURIComponent(match[1]);
    const parsed = JSON.parse(value) as { preferences?: { essential?: boolean } };
    return parsed.preferences?.essential === true;
  } catch {
    return false;
  }
};

