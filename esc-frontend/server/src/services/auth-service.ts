import type { AuthService } from "./interfaces.js";
import { upstream } from "../upstream.js";

export class ProductionAuthService implements AuthService {
  async loginWithToken(role: "admin" | "jury", token: string) {
    const endpoint = role === "admin" ? "/admin/authenticate" : "/jury/authenticate";
    const response = await upstream.get(endpoint, { params: { Token: token } });
    if (response.status !== 202) {
      return { ok: false, role };
    }
    return { ok: true, role };
  }

  async loginWithEmail(email: string, password: string, role: "admin" | "jury") {
    const response = await upstream.post("/auth/login", { email, password, role });
    if (response.status === 202) {
      return { ok: true };
    }
    return { error: response.data?.error ?? "Login failed" };
  }

  async verifyEmail(email: string, code: string, role: "admin" | "jury") {
    const response = await upstream.post("/auth/verify", { email, code });
    if (response.status === 202) {
      return { ok: true, role };
    }
    return { error: response.data?.error ?? "Verification failed" };
  }

  async authenticateToken(token: string, role: "admin" | "jury") {
    const endpoint = role === "admin" ? "/admin/authenticate" : "/jury/authenticate";
    const response = await upstream.get(endpoint, { params: { Token: token } });
    if (response.status === 202) {
      return { ok: true };
    }
    return { error: response.data?.error ?? "Authentication failed" };
  }
}
