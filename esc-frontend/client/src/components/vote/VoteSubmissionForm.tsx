import { useEffect, useState } from "react";
import type { Song } from "../../types";
import { useFlash } from "../../context/FlashContext";
import { Modal } from "../ui/Modal";
import * as songApi from "../../services/songs-api";

type VoteSubmissionFormMode = "total" | "single";

interface VoteSubmissionFormProps {
  open: boolean;
  mode: VoteSubmissionFormMode;
  points: number;
  songId?: number;
  onClose: () => void;
  onSubmit: (phone: string, ownCountry: string, songID?: number) => Promise<void>;
}

export const VoteSubmissionForm = ({
  open,
  mode,
  points,
  songId,
  onClose,
  onSubmit
}: VoteSubmissionFormProps) => {
  const [phone, setPhone] = useState("");
  const [ownCountry, setOwnCountry] = useState("");
  const [songs, setSongs] = useState<Song[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const { addFlash } = useFlash();

  useEffect(() => {
    if (open) {
      void songApi
        .getSongs()
        .then(setSongs)
        .catch((err: unknown) => {
          const message = err instanceof Error ? err.message : "Failed to load songs";
          setError(message);
          addFlash(message, "error");
        });
      setError(null);
    }
  }, [addFlash, open]);

  const handleSubmit = async () => {
    setError(null);
    if (!phone.trim() || !ownCountry) {
      setError("Please fill in your country and phone number.");
      return;
    }
    setSubmitting(true);
    try {
      await onSubmit(phone.trim(), ownCountry, songId);
      setPhone("");
      setOwnCountry("");
      onClose();
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Submission failed");
    } finally {
      setSubmitting(false);
    }
  };

  const title = mode === "total" ? "Submit votes" : "Submit vote";
  const pointText = `point${points !== 1 ? "s" : ""}`;
  const description = mode === "total" 
    ? `You are submitting ${points} ${pointText} in total.`
    : `You are submitting ${points} ${pointText} for this performance.`;

  return (
    <Modal open={open} title={title} onClose={onClose}>
      <div className="space-y-4">
        <p className="text-sm text-esc-muted">
          {mode === "total" ? (
            description
          ) : (
            <>
              You are submitting <span className="font-semibold">{points}</span> {pointText} for this performance.
            </>
          )}
        </p>
        {error && (
          <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
            {error}
          </div>
        )}
        <label className="block text-sm text-esc-black-soft">
          Phone number
          <input
            type="tel"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="Enter your phone number"
            className="mt-1.5 w-full rounded-xl border border-esc-border bg-esc-surface px-3 py-2 text-esc-black transition-colors duration-200 focus:border-esc-pink"
          />
        </label>
        <label className="block text-sm text-esc-black-soft">
          Your country
          <select
            value={ownCountry}
            onChange={(e) => setOwnCountry(e.target.value)}
            className="mt-1.5 w-full rounded-xl border border-esc-border bg-esc-surface px-3 py-2 text-esc-black transition-colors duration-200 focus:border-esc-pink"
          >
            <option value="">Select your country</option>
            {songs.map((song) => (
              <option key={song.countryId} value={song.countryId}>
                {song.countryName} ({song.countryId})
              </option>
            ))}
          </select>
        </label>
        <button
          className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim disabled:opacity-50"
          disabled={submitting}
          onClick={() => void handleSubmit()}
        >
          {submitting ? "Submitting..." : "Confirm Submit"}
        </button>
      </div>
    </Modal>
  );
};
