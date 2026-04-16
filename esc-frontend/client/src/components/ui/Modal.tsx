import type { ReactNode } from "react";

export const Modal = ({
  open,
  title,
  children,
  onClose
}: {
  open: boolean;
  title: string;
  children: ReactNode;
  onClose: () => void;
}) => {
  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/20 p-4 backdrop-blur-[2px]"
      role="dialog"
      aria-modal="true"
    >
      <div className="w-full max-w-lg rounded-2xl border border-esc-border bg-esc-surface p-5 shadow-[0_28px_80px_rgba(0,0,0,0.18)]">
        <div className="mb-4 flex items-center justify-between border-b border-esc-border pb-3">
          <h3 className="text-lg font-bold text-esc-black">{title}</h3>
          <button
            onClick={onClose}
            className="rounded-lg border border-esc-border px-2.5 py-1 text-sm text-esc-muted transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
          >
            x
          </button>
        </div>
        {children}
      </div>
    </div>
  );
};