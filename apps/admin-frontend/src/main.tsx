import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Route, Routes } from "react-router";

import { AppShell } from "./pages/AppShell";
import { CampaignDetailsPage, CampaignsPage } from "./pages/CampaignsPage";
import { CostsPage } from "./pages/CostsPage";
import { DemoSubscribePage } from "./pages/DemoSubscribePage";
import { LoginPage } from "./pages/LoginPage";
import { PostbackEventsPage, PostbacksPage } from "./pages/PostbacksPage";
import { PublishersPage } from "./pages/PublishersPage";
import { ReportsPage } from "./pages/ReportsPage";
import { SourcesPage } from "./pages/SourcesPage";
import { UsersPage } from "./pages/UsersPage";
import { VAPIDKeysPage } from "./pages/VAPIDKeysPage";
import { ProtectedRoute } from "./routes/ProtectedRoute";
import "./styles.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      staleTime: 30_000
    }
  }
});

const rootElement = document.getElementById("root");

if (!rootElement) {
  throw new Error("Root element not found");
}

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/demo/:sourceId" element={<DemoSubscribePage />} />
          <Route element={<ProtectedRoute />}>
            <Route element={<AppShell />}>
              <Route index element={<Navigate to="/publishers" replace />} />
              <Route path="/users" element={<UsersPage />} />
              <Route path="/publishers" element={<PublishersPage />} />
              <Route path="/publishers/:publisherId/sources" element={<SourcesPage />} />
              <Route path="/sources" element={<Navigate to="/publishers" replace />} />
              <Route path="/campaigns" element={<CampaignsPage />} />
              <Route path="/campaigns/:campaignId" element={<CampaignDetailsPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/costs" element={<CostsPage />} />
              <Route path="/postbacks" element={<PostbacksPage />} />
              <Route path="/postbacks/:postbackConfigId" element={<PostbackEventsPage />} />
              <Route path="/vapid-keys" element={<VAPIDKeysPage />} />
            </Route>
          </Route>
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>
);
