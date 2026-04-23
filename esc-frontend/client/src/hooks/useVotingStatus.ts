import { useEffect, useState } from "react";

import { getSongs } from "../services/songs-api";

export const useVotingStatus = () => {
  const [isOpen, setIsOpen] = useState(false);

  useEffect(() => {
    const load = async () => {
      const songs = await getSongs();
      setIsOpen(songs[0]?.votingIsOpen ?? false);
    };

    void load();
    const timer = window.setInterval(() => {
      void load();
    }, 5000);
    return () => window.clearInterval(timer);
  }, []);

  return isOpen;
};

