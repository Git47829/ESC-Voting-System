import { Link } from "react-router-dom";

const primaryLinks = [
  { to: "/", label: "Vote" },
  { to: "/now", label: "Running Now" },
  { to: "/results", label: "Results" },
  { to: "/stats", label: "Stats" }
];

const adminLinks = [
  { to: "/admin", label: "Admin" },
  { to: "/jury", label: "Jury" },
  { to: "/login", label: "Login" },
  { to: "/cookies", label: "Cookie Settings" }
];

export const Footer = () => {
  return (
    <footer className="border-t border-esc-border bg-white/95 px-4 py-8 text-sm text-esc-muted">
      <div className="mx-auto max-w-7xl">
        <div className="grid gap-8 border-b border-esc-border pb-7 md:grid-cols-[minmax(0,1.2fr)_minmax(0,0.8fr)_minmax(0,0.8fr)]">
          <div className="space-y-2">
            <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">ESC Voting System</p>
            <p className="max-w-md text-sm leading-6 text-esc-black-soft/75">
              Live voting experience for Eurovision-style contests. Cast points, follow rankings and review results in real time.
            </p>
          </div>

          <nav aria-label="Footer primary" className="space-y-2">
            <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">Navigate</p>
            <div className="grid gap-1">
              {primaryLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className="transition-colors duration-200 hover:text-esc-pink focus-visible:text-esc-pink"
                >
                  {link.label}
                </Link>
              ))}
            </div>
          </nav>

          <nav aria-label="Footer secondary" className="space-y-2">
            <p className="text-xs uppercase tracking-[0.14em] text-esc-muted">More</p>
            <div className="grid gap-1">
              {adminLinks.map((link) => (
                <Link
                  key={link.to}
                  to={link.to}
                  className="transition-colors duration-200 hover:text-esc-pink focus-visible:text-esc-pink"
                >
                  {link.label}
                </Link>
              ))}
            </div>
          </nav>
        </div>

        <div className="pt-4 text-xs text-esc-muted/90">
          <p>© {new Date().getFullYear()} ESC Voting System · Live voting experience.</p>
        </div>
      </div>
    </footer>
  );
};