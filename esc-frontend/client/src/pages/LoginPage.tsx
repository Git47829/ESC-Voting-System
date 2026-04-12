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
    <section className="mx-auto max-w-xl rounded-[2rem] border border-esc-border bg-white/92 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.06)] sm:p-8">
      <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Access</p>
      <h1 className="mt-2 text-3xl font-bold text-esc-black">Login</h1>
      <form
        className="mt-6 space-y-4"
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
        <select className="w-full rounded-xl border border-esc-border bg-white px-3 py-2 text-esc-black focus:border-esc-pink" value={role} onChange={(e) => setRole(e.target.value as Role)}>
          <option value="admin">Admin</option>
          <option value="jury">Jury</option>
        </select>
        <input className="w-full rounded-xl border border-esc-border bg-white px-3 py-2 text-esc-black placeholder:text-esc-muted/70 focus:border-esc-pink" placeholder="Token" value={token} onChange={(e) => setToken(e.target.value)} />
        <button className="w-full rounded-xl border border-esc-pink bg-esc-pink px-4 py-2.5 font-semibold text-white transition-colors hover:bg-esc-pink-dim">
          Login
        </button>
      </form>
    </section>
  );
};
