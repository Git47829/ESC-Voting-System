import { useFlash } from "../../context/FlashContext";

export const FlashMessages = () => {
  const { messages, removeFlash } = useFlash();

  return (
    <div aria-live="polite" className="fixed right-4 top-20 z-50 flex w-80 flex-col gap-2">
      {messages.map((message) => (
        <button
          key={message.id}
          onClick={() => removeFlash(message.id)}
          className={`flash-fade border px-3 py-2 text-left text-sm ${
            message.category === "success"
              ? "border-emerald-400 bg-emerald-500/20"
              : message.category === "error"
                ? "border-red-500 bg-red-600/20"
                : "border-esc-yellow bg-esc-yellow/20"
          }`}
        >
          {message.message}
        </button>
      ))}
    </div>
  );
};

