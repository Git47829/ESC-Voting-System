import { ApiError, extractErrorMessage } from "./error-handler";
import { getCsrfToken, resetCsrfToken } from "./csrf-service";

const fetchJson = async <T>(url: string, init?: RequestInit, isRetry = false): Promise<T> => {
  const method = (init?.method ?? "GET").toUpperCase();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string> ?? {})
  };

  if (method !== "GET" && method !== "HEAD") {
    headers["x-csrf-token"] = await getCsrfToken();
  }

  const response = await fetch(url, {
    credentials: "include",
    ...init,
    headers
  });

  if (response.status === 403 && !isRetry) {
    resetCsrfToken();
    return fetchJson<T>(url, init, true);
  }

  const rawBody = await response.text();
  let data = {} as T;
  if (rawBody.length > 0) {
      try {
        data = JSON.parse(rawBody) as T;
      } catch {
        if (!response.ok) {
          throw new ApiError(rawBody.trim() || `Request failed: ${response.status}`, response.status);
        }
        throw new Error("Received invalid JSON response");
      }
  }

  if (!response.ok) {
    const message =
      extractErrorMessage(data) ??
      (response.statusText || `Request failed: ${response.status}`);
    throw new ApiError(message, response.status);
  }

  return data;
};

export { fetchJson };
