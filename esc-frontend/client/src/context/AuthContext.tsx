import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from "react";

import { api } from "../api/client";
import type { Role } from "../types";

interface AuthContextValue {
  role: Role | null;
  token: string | null;
  authenticated: boolean;
  loading: boolean;
  login: (role: Role, token: string) => Promise<void>;
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
        const session = await api.session();
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
        await api.login(nextRole, nextToken);
        setRole(nextRole);
        setToken(nextToken);
      },
      logout: async () => {
        await api.logout();
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

