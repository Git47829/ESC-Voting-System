import axios from "axios";
import { config } from "./config.js";
export const upstream = axios.create({
    baseURL: config.apiBaseUrl,
    timeout: config.apiTimeout,
    validateStatus: () => true
});
export const passSetCookie = (headers, setCookie) => {
    const raw = headers["set-cookie"];
    if (raw && Array.isArray(raw) && raw.length > 0) {
        setCookie("set-cookie", raw);
    }
};
export const parseConsentCookie = (cookieHeader) => {
    if (!cookieHeader) {
        return false;
    }
    const match = cookieHeader.match(/esc_cookie_consent=([^;]+)/);
    if (!match) {
        return false;
    }
    try {
        const value = decodeURIComponent(match[1]);
        const parsed = JSON.parse(value);
        return parsed.preferences?.essential === true;
    }
    catch {
        return false;
    }
};
