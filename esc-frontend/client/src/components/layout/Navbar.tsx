import { Link, NavLink } from "react-router-dom";

import { useAuth } from "../../context/AuthContext";
import { useVotingStatus } from "../../hooks/useVotingStatus";
import { StatusBadge } from "../ui/StatusBadge";

export const Navbar = () => {
  const { role, logout, authenticated } = useAuth();
  const votingOpen = useVotingStatus();

  const navItemClass = ({ isActive }: { isActive: boolean }) =>
    `relative px-1 py-1 text-sm md:text-base transition-colors duration-200 after:absolute after:-bottom-1 after:left-0 after:h-px after:w-full after:origin-left after:transition-transform after:duration-200 ${
      isActive
        ? "text-esc-pink after:scale-x-100 after:bg-esc-pink"
        : "text-esc-muted hover:text-esc-black after:scale-x-0 after:bg-esc-pink"
    }`;

  return (
    <header className="sticky top-0 z-40 border-b border-esc-border bg-white/92 backdrop-blur-sm">
      <nav className="mx-auto flex max-w-7xl items-center justify-between px-4 py-4">
        <Link to="/" className="font-heading text-xl font-bold tracking-[0.03em] text-esc-black">
          ESC-Voting
        </Link>
        <div className="flex items-center gap-7">
          <NavLink to="/" className={navItemClass}>Vote</NavLink>
          <NavLink to="/now" className={navItemClass}>Running Now</NavLink>
          <NavLink to="/results" className={navItemClass}>Results</NavLink>
          <NavLink to="/stats" className={navItemClass}>Stats</NavLink>
          <NavLink to="/admin" className={navItemClass}>Admin</NavLink>
          <NavLink to="/jury" className={navItemClass}>Jury</NavLink>
          {authenticated ? (
            <button
              onClick={() => void logout()}
              className="rounded-xl border border-esc-border px-3 py-1.5 text-sm text-esc-muted transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
            >
              Logout ({role})
            </button>
          ) : (
            <NavLink to="/login" className={navItemClass}>Login</NavLink>
          )}

          <StatusBadge isOpen={votingOpen} />
        </div>
      </nav>
    </header>
  );
};