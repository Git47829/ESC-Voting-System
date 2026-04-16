import dotenv from "dotenv";

dotenv.config();

const toInt = (value: string | undefined, fallback: number): number => {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isFinite(parsed) ? parsed : fallback;
};

export const config = {
  nodeEnv: process.env.NODE_ENV ?? "development",
  useMock: (process.env.USE_MOCK ?? "").toLowerCase() === "true",
  apiBaseUrl: process.env.API_BASE_URL ?? "http://db-crud-api:8000",
  apiTimeout: toInt(process.env.API_TIMEOUT, 10_000),
  escConverterUrl:
    process.env.ESC_CONVERTER_URL ?? "http://public-vote-converter:8090",
  eurostatsUrl: process.env.EUROSTATS_URL ?? "http://eurostats:8880",
  port: toInt(process.env.PORT, 3001),
  sessionSecret: process.env.SESSION_SECRET ?? "change-me-in-production",
  totalVotePoints: toInt(process.env.TOTAL_VOTE_POINTS, 20)
};

export const isMockMode = (): boolean =>
  config.useMock || config.nodeEnv === "development";

