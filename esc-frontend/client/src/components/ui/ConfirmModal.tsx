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
      <p className="mb-5 text-sm leading-6 text-esc-muted">{text}</p>
      <div className="flex flex-col-reverse gap-3 sm:flex-row sm:justify-end">
        <button
          className="inline-flex items-center justify-center rounded-xl border border-esc-border px-4 py-2.5 text-sm font-medium text-esc-black transition-colors duration-200 hover:border-esc-pink hover:text-esc-pink"
          onClick={onClose}
        >
          Cancel
        </button>
        <button
          className="inline-flex items-center justify-center rounded-xl border border-red-500 bg-red-500 px-4 py-2.5 text-sm font-semibold text-white transition-colors duration-200 hover:bg-red-600"
          onClick={onConfirm}
        >
          Confirm reset
        </button>
      </div>
    </Modal>
  );
};
