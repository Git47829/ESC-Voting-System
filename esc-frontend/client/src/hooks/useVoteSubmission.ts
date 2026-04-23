import { useState } from "react";
import { useFlash } from "../context/FlashContext";
import * as votingApi from "../services/voting-api";

export const useVoteSubmission = () => {
  const [phone, setPhone] = useState("");
  const [ownCountry, setOwnCountry] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const { addFlash } = useFlash();

  const resetForm = () => {
    setPhone("");
    setOwnCountry("");
    setError(null);
  };

  const validateForm = (): boolean => {
    if (!phone.trim() || !ownCountry) {
      setError("Please fill in your country and phone number.");
      return false;
    }
    return true;
  };

  const submitVote = async (
    songID: number,
    points: number,
    onSuccess?: () => void
  ): Promise<void> => {
    setError(null);
    if (!validateForm()) {
      return;
    }

    setSubmitting(true);
    try {
      await votingApi.submitVote({
        songID,
        phoneNum: phone.trim(),
        ownCountry,
        points
      });
      
      const pointText = `point${points !== 1 ? "s" : ""}`;
      addFlash(`Vote submitted: ${points} ${pointText}`, "success");
      resetForm();
      onSuccess?.();
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : "Submission failed";
      setError(message);
    } finally {
      setSubmitting(false);
    }
  };

  return {
    phone,
    setPhone,
    ownCountry,
    setOwnCountry,
    error,
    setError,
    submitting,
    validateForm,
    submitVote,
    resetForm
  };
};
