import { Link } from "react-router-dom";

export const Footer = () => {
  return (
    <footer className="mt-12 border-t border-esc-border bg-esc-surface px-4 py-7 text-sm text-esc-muted">
      <div className="mx-auto flex max-w-7xl items-center justify-between">
        <span>ESC Voting System</span>
        <Link to="/cookies" className="font-medium text-esc-pink transition-colors duration-200 hover:text-esc-pink-dim">
          Cookie Settings
        </Link>
      </div>
    </footer>
  );
};