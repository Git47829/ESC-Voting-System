const points = [1, 2, 3, 4, 5, 6, 7, 8, 10, 12];

export const PointsLegend = () => (
  <div className="rounded-[2rem] border border-esc-border bg-white/95 px-5 py-5 shadow-[0_18px_42px_rgba(0,0,0,0.05)] sm:px-6">
    <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
      <div>
        <p className="text-[11px] uppercase tracking-[0.2em] text-esc-muted">Jury values</p>
        <h2 className="mt-2 text-2xl font-bold tracking-[-0.04em] text-esc-black">
          Allowed point set
        </h2>
      </div>

      <div className="flex flex-wrap gap-2">
        {points.map((point) => (
          <span
            key={point}
            className="inline-flex min-w-[3rem] items-center justify-center rounded-full border border-esc-border bg-esc-surface2 px-3 py-2 text-sm font-semibold text-esc-black"
          >
            {point}
          </span>
        ))}
      </div>
    </div>
  </div>
);
