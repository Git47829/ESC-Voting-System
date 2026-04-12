import { createContext, type ReactNode, useContext, useMemo, useState } from "react";

import type { FlashMessage } from "../types";

interface FlashContextValue {
  messages: FlashMessage[];
  addFlash: (message: string, category: FlashMessage["category"]) => void;
  removeFlash: (id: string) => void;
}

const FlashContext = createContext<FlashContextValue | null>(null);

export const FlashProvider = ({ children }: { children: ReactNode }) => {
  const [messages, setMessages] = useState<FlashMessage[]>([]);

  const value = useMemo<FlashContextValue>(
    () => ({
      messages,
      addFlash: (message, category) => {
        const id = `${Date.now()}-${Math.random()}`;
        setMessages((current) => [...current, { id, category, message }]);
        window.setTimeout(() => {
          setMessages((current) => current.filter((entry) => entry.id !== id));
        }, 4000);
      },
      removeFlash: (id) => {
        setMessages((current) => current.filter((entry) => entry.id !== id));
      }
    }),
    [messages]
  );

  return <FlashContext.Provider value={value}>{children}</FlashContext.Provider>;
};

export const useFlash = (): FlashContextValue => {
  const context = useContext(FlashContext);
  if (!context) {
    throw new Error("useFlash must be used inside FlashProvider");
  }
  return context;
};


