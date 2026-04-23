export class ApiError extends Error {
  public readonly status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

export const extractErrorMessage = (value: unknown): string | null => {
  if (typeof value === "string") {
    const trimmed = value.trim();
    return trimmed.length > 0 ? trimmed : null;
  }

  if (!value || typeof value !== "object") {
    return null;
  }

  const record = value as Record<string, unknown>;
  const direct =
    extractErrorMessage(record.error) ??
    extractErrorMessage(record.message) ??
    extractErrorMessage(record.detail) ??
    extractErrorMessage(record.title) ??
    extractErrorMessage(record.payload);

  if (direct) {
    return direct;
  }

  if (Array.isArray(record.errors)) {
    const first = record.errors.map(extractErrorMessage).find(Boolean);
    return first ?? null;
  }

  return null;
};
