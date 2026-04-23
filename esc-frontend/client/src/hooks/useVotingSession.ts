import { useEffect, useState } from "react";
import { useFlash } from "../context/FlashContext";
import * as songApi from "../services/songs-api";
import * as votingApi from "../services/voting-api";
import type { Song } from "../types";

const TOTAL_POINTS = 20;

export const useVotingSession = () => {
  const [songs, setSongs] = useState<Song[]>([]);
  const [selection, setSelection] = useState<Record<number, number>>({});
  const [serverRemaining, setServerRemaining] = useState(TOTAL_POINTS);
  const [hasSubmitted, setHasSubmitted] = useState(false);
  const [loading, setLoading] = useState(true);
  const { addFlash } = useFlash();

  const votingLocked = hasSubmitted || serverRemaining === 0;

  useEffect(() => {
    const loadVotingData = async () => {
      try {
        const [songsData, voteState] = await Promise.all([
          songApi.getSongs(),
          votingApi.getVoteState()
        ]);
        
        setSongs(songsData);
        setServerRemaining(voteState.votesRemaining);
        
        if (Object.keys(voteState.votesCast).length > 0) {
          setSelection(voteState.votesCast);
        }
        
        if (voteState.votesRemaining === 0 && Object.keys(voteState.votesCast).length > 0) {
          setHasSubmitted(true);
        }
      } catch (error: unknown) {
        const message = error instanceof Error ? error.message : "Failed to load voting data";
        addFlash(message, "error");
      } finally {
        setLoading(false);
      }
    };

    void loadVotingData();
  }, [addFlash]);

  return {
    songs,
    selection,
    setSelection,
    serverRemaining,
    setServerRemaining,
    hasSubmitted,
    setHasSubmitted,
    votingLocked,
    loading
  };
};
