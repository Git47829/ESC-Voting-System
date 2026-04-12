import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../context/AuthContext";
import { useFlash } from "../context/FlashContext";
import type { Role } from "../types";

export const LoginPage = () => {
  const [role, setRole] = useState<Role>("admin");
  const [token, setToken] = useState("");
  const { login } = useAuth();
  const navigate = useNavigate();
  const { addFlash } = useFlash();

  return (
    <section className="mx-auto max-w-md border border-esc-muted p-4">
      <h1 className="mb-4 text-2xl font-bold">Login</h1>
      <form
        className="space-y-3"
        onSubmit={(e) => {
          e.preventDefault();
          void login(role, token)
            .then(() => {
              addFlash("Login successful", "success");
              navigate(role === "admin" ? "/admin" : "/jury");
            })
            .catch((error: unknown) => {
              addFlash(error instanceof Error ? error.message : "Login failed", "error");
            });
        }}
      >
        <select className="w-full border border-esc-muted bg-transparent px-2 py-1" value={role} onChange={(e) => setRole(e.target.value as Role)}>
          <option value="admin">Admin</option>
          <option value="jury">Jury</option>
        </select>
        <input className="w-full border border-esc-muted bg-transparent px-2 py-1" placeholder="Token" value={token} onChange={(e) => setToken(e.target.value)} />
        <button className="w-full border border-esc-yellow px-3 py-1 text-esc-yellow">Login</button>
      </form>
    </section>
  );
};

