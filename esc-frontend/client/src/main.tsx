import React from "react";
import ReactDOM from "react-dom/client";
import { BrowserRouter } from "react-router-dom";

import { App } from "./App";
import { AuthProvider } from "./context/AuthContext";
import { CookieConsentProvider } from "./context/CookieConsentContext";
import { FlashProvider } from "./context/FlashContext";
import "./globals.css";

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <BrowserRouter>
      <FlashProvider>
        <AuthProvider>
          <CookieConsentProvider>
            <App />
          </CookieConsentProvider>
        </AuthProvider>
      </FlashProvider>
    </BrowserRouter>
  </React.StrictMode>
);

