import { fetchJson } from "./http-client";
import type { Role } from "../types";

export const authLogin = (email: string, password: string, role: Role): Promise<{ message: string }> =>
  fetchJson<{ message: string }>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password, role })
  });

export const authVerify = (email: string, code: string, role: Role): Promise<{ ok: boolean; role: Role }> =>
  fetchJson<{ ok: boolean; role: Role }>("/api/auth/verify", {
    method: "POST",
    body: JSON.stringify({ email, code, role })
  });

export const login = (role: Role, token: string): Promise<{ ok: boolean; role: Role }> =>
  fetchJson<{ ok: boolean; role: Role }>("/api/login", {
    method: "POST",
    body: JSON.stringify({ role, token })
  });

export const logout = (): Promise<{ ok: boolean }> =>
  fetchJson<{ ok: boolean }>("/api/logout", { method: "POST" });

export const session = () =>
  fetchJson("/api/session");
