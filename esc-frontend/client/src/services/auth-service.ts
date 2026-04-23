import type { Role, SessionState } from "../types";
import * as authApi from "./auth-api";

export interface IAuthService {
  login(email: string, password: string, role: Role): Promise<void>;
  verify(email: string, code: string, role: Role): Promise<SessionState>;
  logout(): Promise<void>;
  getSession(): Promise<SessionState>;
  directLogin(role: Role, token: string): Promise<void>;
}

export const authService: IAuthService = {
  async login(email: string, password: string, role: Role): Promise<void> {
    await authApi.authLogin(email, password, role);
  },

  async verify(email: string, code: string, role: Role): Promise<SessionState> {
    await authApi.authVerify(email, code, role);
    const session = await authApi.session() as SessionState;
    return session;
  },

  async logout(): Promise<void> {
    await authApi.logout();
  },

  async getSession(): Promise<SessionState> {
    return authApi.session() as Promise<SessionState>;
  },

  async directLogin(role: Role, token: string): Promise<void> {
    await authApi.login(role, token);
  }
};
