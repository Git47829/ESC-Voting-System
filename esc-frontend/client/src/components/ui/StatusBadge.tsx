export const StatusBadge = ({ isOpen }: { isOpen: boolean }) => {
  return (
    <span
      className={`inline-flex items-center gap-2 rounded-full border px-2.5 py-1 text-xs tracking-wide ${
        isOpen ? "border-esc-pink text-esc-pink" : "border-esc-border-strong text-esc-muted"
      }`}
    >
      <span
        className={`h-2 w-2 rounded-full ${isOpen ? "animate-pulse bg-esc-pink" : "bg-esc-muted"}`}
      />
      {isOpen ? "OPEN" : "CLOSED"}
    </span>
  );
};
