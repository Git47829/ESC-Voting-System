import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";

import { useAuth } from "../../context/AuthContext";

export const ProtectedRoute = ({
  role,
  children
}: {
  role: "admin" | "jury";
  children: ReactNode;
}) => {
  const { authenticated, loading, role: currentRole } = useAuth();

  if (loading) return null;
  if (!authenticated) return <Navigate to="/login" replace />;
  if (currentRole !== role) return <Navigate to="/login" replace />;
  return <>{children}</>;
};


