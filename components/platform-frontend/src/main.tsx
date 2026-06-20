import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "react-router-dom";
import { AppStoreProvider } from "./app/store";
import { UIProvider } from "./app/ui";
import { router } from "./app/router";
import "./styles/app.css";

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
      <AppStoreProvider>
        <UIProvider>
          <RouterProvider router={router} />
        </UIProvider>
      </AppStoreProvider>
    </QueryClientProvider>
  </StrictMode>,
);
