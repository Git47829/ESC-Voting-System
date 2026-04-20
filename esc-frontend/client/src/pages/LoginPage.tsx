import { useState } from "react";
import { useNavigate } from "react-router-dom";

import { useAuth } from "../context/AuthContext";
import { useFlash } from "../context/FlashContext";
import type { Role } from "../types";

const accessHighlights = [
  "Admin and jury access",
  "Secure token-based sign-in",
  "Direct route into the right workspace"
];

export const LoginPage = () => {
  const [role, setRole] = useState<Role>("admin");
  const [token, setToken] = useState("");
  const { login } = useAuth();
  const navigate = useNavigate();
  const { addFlash } = useFlash();

  return (
    <section className="-mx-4 sm:-mx-6 lg:-mx-8">
      <div className="relative overflow-hidden bg-[linear-gradient(180deg,#f8f1f5_0%,#fff7fa_36%,#ffffff_100%)]">
        <div className="pointer-events-none absolute inset-0 opacity-[0.32] [background-image:linear-gradient(rgba(17,17,17,0.045)_1px,transparent_1px),linear-gradient(90deg,rgba(17,17,17,0.045)_1px,transparent_1px)] [background-size:8.5rem_8.5rem]" />
        <div className="pointer-events-none absolute left-[-8rem] top-16 h-72 w-72 rounded-full bg-esc-pink/12 blur-3xl" />
        <div className="pointer-events-none absolute right-[-10rem] top-8 h-80 w-80 rounded-full bg-esc-pink/10 blur-3xl" />

        <div className="relative mx-auto max-w-7xl px-6 pb-16 pt-8 sm:px-10 lg:px-14 lg:pb-24">
          <div className="flex flex-wrap items-center justify-between gap-4 border-b border-black/10 pb-4 text-[11px] uppercase tracking-[0.24em] text-esc-muted">
            <div>Secure access</div>
            <div>Admin and jury workspace</div>
          </div>

          <div className="mt-10 grid gap-8 xl:grid-cols-[minmax(0,1.02fr)_25rem] xl:items-start">
            <div>
              <div className="inline-flex items-center gap-2 rounded-full border border-black/10 bg-white/78 px-4 py-2 text-[11px] font-semibold uppercase tracking-[0.22em] text-esc-black-soft shadow-[0_14px_34px_rgba(17,17,17,0.05)] backdrop-blur-sm">
                <span className="h-2 w-2 rounded-full bg-esc-pink" />
                Account access
              </div>

              <h1 className="mt-6 text-[clamp(4.2rem,11vw,8.6rem)] font-bold leading-[0.9] tracking-[-0.09em] text-esc-black">
                <span className="block">Login</span>
                <span className="block text-esc-pink">Panel</span>
              </h1>

              <p className="mt-7 max-w-3xl text-base leading-8 text-esc-black-soft/80 sm:text-lg">
                Choose your role, enter your token and continue straight into the correct control
                area. This page is intentionally simple, focused and built for quick access.
              </p>

              <div className="mt-8 grid gap-3 sm:grid-cols-3 xl:max-w-4xl">
                {accessHighlights.map((item) => (
                  <div
                    key={item}
                    className="rounded-[1.2rem] border border-black/10 bg-white/80 px-4 py-3 text-[11px] font-semibold uppercase tracking-[0.18em] text-esc-black-soft shadow-[0_14px_34px_rgba(17,17,17,0.05)]"
                  >
                    {item}
                  </div>
                ))}
              </div>
            </div>

            <aside className="overflow-hidden rounded-[2.1rem] border border-esc-pink/22 bg-[linear-gradient(180deg,rgba(18,18,18,0.98),rgba(34,24,29,0.95))] p-6 text-white shadow-[0_32px_70px_rgba(17,17,17,0.24)]">
              <p className="text-[11px] uppercase tracking-[0.2em] text-white/55">Sign in</p>
              <h2 className="mt-3 text-3xl font-semibold tracking-[-0.05em]">Account access</h2>
              <p className="mt-3 text-sm leading-7 text-white/72">
                Use your assigned role and token to continue to the right workspace.
              </p>

              <form
                className="mt-7 space-y-5"
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
                <div>
                  <p className="text-[11px] uppercase tracking-[0.18em] text-white/50">Role</p>
                  <div className="mt-3 grid grid-cols-2 gap-2">
                    {(["admin", "jury"] as Role[]).map((option) => {
                      const active = role === option;
                      return (
                        <button
                          key={option}
                          type="button"
                          onClick={() => setRole(option)}
                          className={`rounded-[1.15rem] border px-4 py-3 text-sm font-semibold capitalize transition-all ${
                            active
                              ? "border-esc-pink/40 bg-esc-pink text-white shadow-[0_18px_38px_rgba(255,4,144,0.22)]"
                              : "border-white/10 bg-white/[0.04] text-white/82 hover:border-white/18 hover:bg-white/[0.08]"
                          }`}
                        >
                          {option}
                        </button>
                      );
                    })}
                  </div>
                </div>

                <label className="block space-y-2">
                  <span className="text-[11px] uppercase tracking-[0.18em] text-white/50">Token</span>
                  <input
                    className="w-full rounded-[1.2rem] border border-white/10 bg-white/[0.05] px-4 py-3 text-white placeholder:text-white/36 focus:border-esc-pink focus:outline-none"
                    placeholder="Enter your token"
                    value={token}
                    onChange={(e) => setToken(e.target.value)}
                  />
                </label>

                <button className="w-full rounded-full bg-esc-pink px-5 py-3 text-sm font-semibold uppercase tracking-[0.18em] text-white transition-transform duration-300 hover:-translate-y-0.5 hover:bg-esc-pink-dim">
                  Continue
                </button>
              </form>

              <div className="mt-6 rounded-[1.2rem] border border-white/10 bg-white/[0.05] px-4 py-4 text-sm leading-7 text-white/68">
                Access permissions depend on the selected role and a valid token.
              </div>
            </aside>
          </div>
        </div>
      </div>
    </section>
  );
};
