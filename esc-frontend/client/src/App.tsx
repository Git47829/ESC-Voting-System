import { Route, Routes, useLocation } from "react-router-dom";

import { CookieBanner } from "./components/layout/CookieBanner";
import { FlashMessages } from "./components/layout/FlashMessages";
import { Footer } from "./components/layout/Footer";
import { Navbar } from "./components/layout/Navbar";
import { ProtectedRoute } from "./components/layout/ProtectedRoute";
import { AdminPage } from "./pages/AdminPage";
import { CookieSettingsPage } from "./pages/CookieSettingsPage";
import { JuryPage } from "./pages/JuryPage";
import { LoginPage } from "./pages/LoginPage";
import { NowPlayingPage } from "./pages/NowPlayingPage";
import { ResultsPage } from "./pages/ResultsPage";
import { StatsPage } from "./pages/StatsPage";
import { VotePage } from "./pages/VotePage";

export const App = () => {
  const { pathname } = useLocation();
  const isImmersiveRoute = pathname === "/" || pathname === "/now" || pathname === "/stats";

  return (
    <div className="min-h-screen bg-esc-white text-esc-black">
      <Navbar />
      <FlashMessages />
      <main className={isImmersiveRoute ? "w-full py-0" : "mx-auto max-w-7xl px-4 py-0"}>
        <Routes>
          <Route path="/" element={<VotePage />} />
          <Route path="/now" element={<NowPlayingPage />} />
          <Route path="/results" element={<ResultsPage />} />
          <Route path="/stats" element={<StatsPage />} />
          <Route path="/cookies" element={<CookieSettingsPage />} />
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/admin"
            element={
              <ProtectedRoute role="admin">
                <AdminPage />
              </ProtectedRoute>
            }
          />
          <Route
            path="/jury"
            element={
              <ProtectedRoute role="jury">
                <JuryPage />
              </ProtectedRoute>
            }
          />
        </Routes>
      </main>
      <Footer />
      <CookieBanner />
    </div>
  );
};
