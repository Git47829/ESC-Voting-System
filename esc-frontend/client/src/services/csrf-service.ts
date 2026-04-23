let csrfToken: string | null = null;

export const getCsrfToken = async (): Promise<string> => {
  if (csrfToken) return csrfToken;
  const res = await fetch("/api/csrf-token", { credentials: "include" });
  const data = (await res.json()) as { csrfToken: string };
  csrfToken = data.csrfToken || "";
  return csrfToken;
};

export const resetCsrfToken = (): void => {
  csrfToken = null;
};
