import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../context/AuthContext";
import { useFlash } from "../context/FlashContext";
import type { Role } from "../types";

export const LoginPage = () => {
  const [role, setRole] = useState<Role>("admin");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const { authLogin } = useAuth();
  const navigate = useNavigate();
  const { addFlash } = useFlash();

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      await authLogin(email, password, role);
      addFlash("Login successful", "success");
      navigate(role === "admin" ? "/admin" : "/jury");
    } catch (error: unknown) {
      addFlash(error instanceof Error ? error.message : "Login failed", "error");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="mx-auto w-full max-w-4xl px-2 py-8 sm:px-0 sm:py-12">
      <div className="space-y-8">
        <div className="space-y-3 rounded-[1.75rem] border border-esc-border bg-white/92 p-6 shadow-[0_16px_36px_rgba(0,0,0,0.05)] sm:p-8">
          <p className="text-xs uppercase tracking-[0.16em] text-esc-muted">Secure access</p>
          <h1 className="text-4xl font-bold text-esc-black">Login</h1>
          <p className="max-w-2xl text-sm leading-7 text-esc-black-soft/75 sm:text-base">
            Sign in to continue to the appropriate workspace.
          </p>
        </div>

        <div className="rounded-[1.75rem] border border-esc-border bg-white/94 p-6 shadow-[0_18px_44px_rgba(0,0,0,0.06)] sm:p-8">
          <div className="mb-6 space-y-2 border-b border-esc-border pb-5">
            <h2 className="text-2xl font-bold text-esc-black">Account access</h2>
            <p className="text-sm text-esc-muted">Enter your credentials to sign in.</p>
          </div>

          <form className="space-y-5" onSubmit={(e) => void handleLogin(e)}>
            <label className="block space-y-2">
              <span className="text-xs font-semibold uppercase tracking-[0.14em] text-esc-muted">Role</span>
              <select
                className="w-full rounded-xl border border-esc-border bg-white px-3 py-2.5 text-esc-black focus:border-esc-pink"
                value={role}
                onChange={(e) => setRole(e.target.value as Role)}
              >
                <option value="admin">Admin</option>
                <option value="jury">Jury</option>
              </select>
            </label>

            <label className="block space-y-2">
              <span className="text-xs font-semibold uppercase tracking-[0.14em] text-esc-muted">Email</span>
              <input
                type="email"
                className="w-full rounded-xl border border-esc-border bg-white px-3 py-2.5 text-esc-black placeholder:text-esc-muted/70 focus:border-esc-pink"
                placeholder="Enter your email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                required
              />
            </label>

            <label className="block space-y-2">
              <span className="text-xs font-semibold uppercase tracking-[0.14em] text-esc-muted">Password</span>
              <input
                type="password"
                className="w-full rounded-xl border border-esc-border bg-white px-3 py-2.5 text-esc-black placeholder:text-esc-muted/70 focus:border-esc-pink"
                placeholder="Enter your password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </label>

            <button
              type="submit"
              disabled={submitting}
              className="w-full rounded-xl border border-esc-pink bg-esc-pink px-4 py-2.5 text-sm font-semibold text-white transition-colors hover:border-esc-pink-dim hover:bg-esc-pink-dim disabled:opacity-50"
            >
              {submitting ? "Signing in..." : "Sign in"}
            </button>
          </form>
        </div>
      </div>
    </section>
  );
};
