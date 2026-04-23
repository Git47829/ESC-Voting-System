import { useEffect, useState } from "react";

const MEDIA_QUERY = "(prefers-reduced-motion: reduce)";
const MAX_SCROLL = 700;

export const useHeroAnimation = () => {
  const [heroAccentActive, setHeroAccentActive] = useState(false);
  const [heroSweepX, setHeroSweepX] = useState(-20);

  useEffect(() => {
    const media = window.matchMedia(MEDIA_QUERY);

    const onScroll = () => {
      if (media.matches) {
        setHeroSweepX(-20);
        return;
      }

      const progress = Math.max(0, Math.min(window.scrollY / MAX_SCROLL, 1));
      const nextX = -20 + progress * 48;
      setHeroSweepX(nextX);
    };

    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });

    return () => {
      window.removeEventListener("scroll", onScroll);
    };
  }, []);

  useEffect(() => {
    const onScroll = () => {
      if (window.scrollY > 24) {
        setHeroAccentActive(true);
        window.removeEventListener("scroll", onScroll);
      }
    };

    if (window.scrollY > 24) {
      setHeroAccentActive(true);
    } else {
      window.addEventListener("scroll", onScroll, { passive: true });
    }

    return () => {
      window.removeEventListener("scroll", onScroll);
    };
  }, []);

  return { heroAccentActive, heroSweepX };
};
