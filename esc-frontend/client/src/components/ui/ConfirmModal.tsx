import { Modal } from "./Modal";

export const ConfirmModal = ({
  open,
  title,
  text,
  onConfirm,
  onClose
}: {
  open: boolean;
  title: string;
  text: string;
  onConfirm: () => void;
  onClose: () => void;
}) => {
  return (
    <Modal open={open} title={title} onClose={onClose}>
      <p className="mb-4 text-sm text-esc-muted">{text}</p>
      <div className="flex gap-2">
        <button className="border border-red-500 px-3 py-1 text-red-400" onClick={onConfirm}>Confirm</button>
        <button className="border border-esc-muted px-3 py-1" onClick={onClose}>Cancel</button>
      </div>
    </Modal>
  );
};

