import { useState } from "react";

import { Modal } from "../ui/Modal";

export const SubmitModal = ({
  open,
  totalPoints,
  onClose,
  onSubmit
}: {
  open: boolean;
  totalPoints: number;
  onClose: () => void;
  onSubmit: (phone: string, ownCountry: string) => Promise<void>;
}) => {
  const [phone, setPhone] = useState("");
  const [ownCountry, setOwnCountry] = useState("");

  return (
    <Modal open={open} title="Submit votes" onClose={onClose}>
      <div className="space-y-4">
        <p className="text-sm text-esc-muted">You are submitting {totalPoints} points in total.</p>
        <label className="block text-sm text-esc-black-soft">
          Phone number
          <input
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            className="mt-1.5 w-full rounded-xl border border-esc-border bg-esc-surface px-3 py-2 text-esc-black transition-colors duration-200 focus:border-esc-pink"
          />
        </label>
        <label className="block text-sm text-esc-black-soft">
          Your country ID
          <input
            value={ownCountry}
            onChange={(e) => setOwnCountry(e.target.value.toUpperCase())}
            className="mt-1.5 w-full rounded-xl border border-esc-border bg-esc-surface px-3 py-2 text-esc-black transition-colors duration-200 focus:border-esc-pink"
          />
        </label>
        <button
          className="rounded-xl border border-esc-pink bg-esc-pink px-4 py-2 text-sm font-semibold text-white transition-colors duration-200 hover:border-esc-pink-dim hover:bg-esc-pink-dim"
          onClick={() => void onSubmit(phone, ownCountry)}
        >
          Confirm Submit
        </button>
      </div>
    </Modal>
  );
};