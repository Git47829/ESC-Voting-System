import { createContext, type ReactNode, useContext, useEffect, useMemo, useState } from "react";

import { api } from "../api/client";
import type { Role } from "../types";

interface AuthContextValue {
  role: Role | null;
  authenticated: boolean;
  loading: boolean;
  authLogin: (email: string, password: string, role: Role) => Promise<void>;
  authVerify: () => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [role, setRole] = useState<Role | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const run = async () => {
      try {
        const session = await api.authMe();
        setRole(session.authenticated ? session.role : null);
      } catch {
        setRole(null);
      } finally {
        setLoading(false);
      }
    };
    void run();
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      role,
      loading,
      authenticated: Boolean(role),
      authLogin: async (email, password, requestedRole) => {
        await api.authLogin(email, password, requestedRole);
        const verified = await api.authVerify();
        if (!verified.ok || !verified.role) {
          throw new Error("Authentication verification failed");
        }
        setRole(verified.role);
      },
      authVerify: async () => {
        const verified = await api.authVerify();
        setRole(verified.ok ? verified.role : null);
      },
      logout: async () => {
        await api.logout();
        setRole(null);
      }
    }),
    [loading, role]
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
