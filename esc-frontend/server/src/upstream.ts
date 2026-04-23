import axios, { type AxiosResponseHeaders, type RawAxiosResponseHeaders, type InternalAxiosRequestConfig } from "axios";
import { trace, type Span } from "@opentelemetry/api";

import { config } from "./config.js";

export const upstream = axios.create({
  baseURL: config.apiBaseUrl,
  timeout: config.apiTimeout,
  validateStatus: () => true
});

// Add request/response interceptors for observability
upstream.interceptors.request.use((axiosConfig: InternalAxiosRequestConfig) => {
  const tracer = trace.getTracer("esc-frontend-upstream");
  const span = tracer.startSpan(`${axiosConfig.method?.toUpperCase()} ${axiosConfig.url}`);

  span.setAttributes({
    "http.method": axiosConfig.method?.toUpperCase() ?? "unknown",
    "http.url": axiosConfig.url ?? "unknown",
    "service.name": "esc-api-upstream"
  });

  // Store span in config for use in response interceptor
  const configWithOtel = axiosConfig as unknown as Record<string, unknown>;
  configWithOtel._otelSpan = span;
  configWithOtel._otelStartTime = Date.now();

  return axiosConfig;
});

upstream.interceptors.response.use(
  (response) => {
    const configWithOtel = response.config as unknown as Record<string, unknown>;
    const span = configWithOtel._otelSpan as Span | undefined;
    const startTime = configWithOtel._otelStartTime as number | undefined;

    if (span && startTime) {
      const duration = Date.now() - startTime;
      span.setAttributes({
        "http.status_code": response.status,
        "http.response_content_length": response.headers["content-length"] ?? 0,
        "http.duration_ms": duration
      });
      span.end();
    }

    return response;
  },
  (error) => {
    const configWithOtel = error.config as unknown as Record<string, unknown> | undefined;
    const span = configWithOtel?._otelSpan as Span | undefined;
    const startTime = configWithOtel?._otelStartTime as number | undefined;

    if (span && startTime) {
      const duration = Date.now() - startTime;
      span.setAttributes({
        "http.error": true,
        "error.message": error instanceof Error ? error.message : "unknown",
        "http.duration_ms": duration
      });
      span.end();
    }

    return Promise.reject(error);
  }
);

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


