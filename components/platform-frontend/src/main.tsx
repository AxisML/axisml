import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import "./i18n";
import "./api/setup";
import { SessionProvider } from "./app/session";
import { AppStoreProvider } from "./app/store";
import { AntdProvider } from "./app/theme";
import { router } from "./app/router";
import "./styles/tokens.css";
import "./styles/tailwind.css";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      refetchOnWindowFocus: false,
      staleTime: 15_000,
    },
  },
});

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <AppStoreProvider>
          <AntdProvider>
            <RouterProvider router={router} />
          </AntdProvider>
        </AppStoreProvider>
      </SessionProvider>
    </QueryClientProvider>
  </StrictMode>,
);
