import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from "react";

import { authService } from "../services/auth-service";
import type { Role } from "../types";

interface AuthContextValue {
  role: Role | null;
  token: string | null;
  authenticated: boolean;
  loading: boolean;
  login: (role: Role, token: string) => Promise<void>;
  authLogin: (email: string, password: string, role: Role) => Promise<void>;
  authVerify: (email: string, code: string, role: Role) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [role, setRole] = useState<Role | null>(null);
  const [token, setToken] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const run = async () => {
      try {
        const session = await authService.getSession();
        setRole(session.role);
        setToken(session.token);
      } finally {
        setLoading(false);
      }
    };
    void run();
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      role,
      token,
      loading,
      authenticated: Boolean(role && token),
      login: async (nextRole, nextToken) => {
        await authService.directLogin(nextRole, nextToken);
        setRole(nextRole);
        setToken(nextToken);
      },
      authLogin: async (email, password, nextRole) => {
        await authService.login(email, password, nextRole);
      },
      authVerify: async (email, code, nextRole) => {
        const session = await authService.verify(email, code, nextRole);
        setRole(session.role);
        setToken(session.token);
      },
      logout: async () => {
        await authService.logout();
        setRole(null);
        setToken(null);
      }
    }),
    [loading, role, token]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = (): AuthContextValue => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used in AuthProvider");
  }
  return context;
};

