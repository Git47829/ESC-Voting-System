const CONSENT_COOKIE_NAME = "esc_cookie_consent";
const CONSENT_SCHEMA_VERSION = 1;
const SIX_MONTHS_SECONDS = 180 * 24 * 60 * 60;

function getCookie(name) {
    const cookies = document.cookie ? document.cookie.split(";") : [];
    for (const rawCookie of cookies) {
        const cookie = rawCookie.trim();
        if (cookie.startsWith(name + "=")) {
            return cookie.slice(name.length + 1);
        }
    }
    return null;
}

function buildConsentPayload(analyticsEnabled) {
    return {
        version: CONSENT_SCHEMA_VERSION,
        consentGivenAt: new Date().toISOString(),
        preferences: {
            essential: true,
            analytics: Boolean(analyticsEnabled),
        },
    };
}

function writeConsentCookie(payload) {
    const value = encodeURIComponent(JSON.stringify(payload));
    let cookie =
        `${CONSENT_COOKIE_NAME}=${value}; Max-Age=${SIX_MONTHS_SECONDS}; Path=/; SameSite=Lax`;

    if (window.location.protocol === "https:") {
        cookie += "; Secure";
    }

    document.cookie = cookie;
    document.dispatchEvent(
        new CustomEvent("esc:cookie-consent-updated", { detail: payload }),
    );
}

function readConsentCookie() {
    const raw = getCookie(CONSENT_COOKIE_NAME);
    if (!raw) {
        return null;
    }

    try {
        const parsed = JSON.parse(decodeURIComponent(raw));
        const analyticsEnabled = Boolean(
            parsed &&
                parsed.preferences &&
                parsed.preferences.analytics,
        );
        return buildConsentPayload(analyticsEnabled);
    } catch (error) {
        return null;
    }
}

function initMobileMenu() {
    const button = document.getElementById("mobile-menu-btn");
    const menu = document.getElementById("mobile-menu");
    if (!button || !menu) {
        return;
    }

    button.addEventListener("click", function () {
        menu.classList.toggle("hidden");
    });
}

function initCookieBanner() {
    const banner = document.getElementById("cookie-banner");
    if (!banner) {
        return;
    }

    const essentialButton = document.getElementById("cookie-accept-essential");
    const allButton = document.getElementById("cookie-accept-all");
    const existingConsent = readConsentCookie();

    if (existingConsent) {
        banner.classList.add("hidden");
        return;
    }

    banner.classList.remove("hidden");

    if (essentialButton) {
        essentialButton.addEventListener("click", function () {
            writeConsentCookie(buildConsentPayload(false));
            banner.classList.add("hidden");
        });
    }

    if (allButton) {
        allButton.addEventListener("click", function () {
            writeConsentCookie(buildConsentPayload(true));
            banner.classList.add("hidden");
        });
    }
}

function initCookieSettingsPage() {
    const form = document.getElementById("cookie-settings-form");
    if (!form) {
        return;
    }

    const analyticsCheckbox = document.getElementById("cookie-analytics");
    const status = document.getElementById("cookie-settings-status");
    const existing = readConsentCookie();

    if (analyticsCheckbox && existing) {
        analyticsCheckbox.checked = Boolean(existing.preferences.analytics);
    }

    form.addEventListener("submit", function (event) {
        event.preventDefault();

        const analyticsEnabled = analyticsCheckbox
            ? analyticsCheckbox.checked
            : false;
        writeConsentCookie(buildConsentPayload(analyticsEnabled));

        if (status) {
            status.textContent = "Settings saved.";
        }
    });
}

initMobileMenu();
initCookieBanner();
initCookieSettingsPage();
