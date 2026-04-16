import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        "esc-white": "#ffffff",
        "esc-surface": "#ffffff",
        "esc-surface2": "#f7f7f7",
        "esc-black": "#111111",
        "esc-black-soft": "#222222",
        "esc-muted": "#6f6f6f",
        "esc-border": "rgba(0,0,0,0.08)",
        "esc-border-strong": "rgba(0,0,0,0.14)",
        "esc-pink": "#ff0490",
        "esc-pink-dim": "#d9037a",
        "esc-pink-soft": "rgba(255,4,144,0.08)",

        "esc-text": "#111111",
        "esc-yellow": "#ff0490"
      },
      fontFamily: {
        heading: ["Europa", "Inter", "sans-serif"],
        body: ["Europa", "Europa-Bold Bold", "Inter", "sans-serif"]
      },
      keyframes: {
        "stage-glow": {
          "0%, 100%": { boxShadow: "0 0 0px rgba(255, 4, 144, 0.0)" },
          "50%": { boxShadow: "0 0 18px rgba(255, 4, 144, 0.22)" }
        },
        "progress-shine": {
          "0%": { backgroundPosition: "200% 0" },
          "100%": { backgroundPosition: "-200% 0" }
        },
        "budget-pulse": {
          "0%, 100%": { transform: "scale(1)" },
          "50%": { transform: "scale(1.01)" }
        },
        "flash-fade": {
          "0%, 85%": { opacity: "1" },
          "100%": { opacity: "0" }
        },
        "vote-success": {
          "0%": { transform: "translateY(6px)", opacity: "0" },
          "100%": { transform: "translateY(0)", opacity: "1" }
        },
        "hero-word": {
          "0%": {
            opacity: "0",
            transform: "translate3d(0, 115%, 0) scale(0.96) rotate(2deg)",
            filter: "blur(8px)"
          },
          "100%": {
            opacity: "1",
            transform: "translate3d(0, 0, 0) scale(1) rotate(0deg)",
            filter: "blur(0px)"
          }
        },
        "hero-fade-up": {
          "0%": { opacity: "0", transform: "translate3d(0, 26px, 0)", filter: "blur(8px)" },
          "100%": { opacity: "1", transform: "translate3d(0, 0, 0)", filter: "blur(0px)" }
        },
        "hero-aurora": {
          "0%": { transform: "translate3d(0, 0, 0) scale(1)", opacity: "0.55" },
          "50%": { transform: "translate3d(18px, -26px, 0) scale(1.08)", opacity: "0.9" },
          "100%": { transform: "translate3d(0, 0, 0) scale(1)", opacity: "0.55" }
        },
        "hero-aurora-delayed": {
          "0%": { transform: "translate3d(0, 0, 0) scale(1.02)", opacity: "0.42" },
          "50%": { transform: "translate3d(-22px, 22px, 0) scale(1.1)", opacity: "0.82" },
          "100%": { transform: "translate3d(0, 0, 0) scale(1.02)", opacity: "0.42" }
        },
        "hero-sweep": {
          "0%": { transform: "translate3d(-26vw, 0, 0) rotate(12deg)", opacity: "0" },
          "18%": { opacity: "0.18" },
          "50%": { transform: "translate3d(0, 0, 0) rotate(12deg)", opacity: "0.72" },
          "82%": { opacity: "0.16" },
          "100%": { transform: "translate3d(26vw, 0, 0) rotate(12deg)", opacity: "0" }
        },
        "hero-card-float": {
          "0%, 100%": { transform: "translate3d(0, 0, 0)" },
          "50%": { transform: "translate3d(0, -14px, 0)" }
        },
        "hero-card-float-delayed": {
          "0%, 100%": { transform: "translate3d(0, 0, 0)" },
          "50%": { transform: "translate3d(0, 14px, 0)" }
        },
        "hero-orbit": {
          "0%": { transform: "rotate(0deg) scale(1)" },
          "50%": { transform: "rotate(180deg) scale(1.03)" },
          "100%": { transform: "rotate(360deg) scale(1)" }
        },
        "scroll-nudge": {
          "0%, 100%": { transform: "translate3d(0, 0, 0)" },
          "50%": { transform: "translate3d(0, 6px, 0)" }
        }
      },
      animation: {
        "stage-glow": "stage-glow 2s ease-in-out infinite",
        "progress-shine": "progress-shine 2s linear infinite",
        "budget-pulse": "budget-pulse 1.3s ease-in-out infinite",
        "flash-fade": "flash-fade 4s ease-in forwards",
        "vote-success": "vote-success 0.3s ease-out",
        "hero-word": "hero-word 1.05s cubic-bezier(0.16, 1, 0.3, 1) both",
        "hero-fade-up": "hero-fade-up 0.95s cubic-bezier(0.22, 1, 0.36, 1) both",
        "hero-aurora": "hero-aurora 10s ease-in-out infinite",
        "hero-aurora-delayed": "hero-aurora-delayed 12s ease-in-out infinite",
        "hero-sweep": "hero-sweep 10s ease-in-out infinite",
        "hero-card-float": "hero-card-float 6s ease-in-out infinite",
        "hero-card-float-delayed": "hero-card-float-delayed 7s ease-in-out infinite",
        "hero-orbit": "hero-orbit 22s linear infinite",
        "scroll-nudge": "scroll-nudge 1.8s ease-in-out infinite"
      }
    }
  },
  plugins: []
};

export default config;

