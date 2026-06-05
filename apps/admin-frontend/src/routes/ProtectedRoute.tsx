import { useQuery } from "@tanstack/react-query";
import { Navigate, Outlet } from "react-router";

import { api } from "../api/client";
import { useAuthStore } from "../store/auth";

export function ProtectedRoute() {
  const token = useAuthStore((state) => state.token);
  const clearToken = useAuthStore((state) => state.clearToken);

  const me = useQuery({
    queryKey: ["me", token],
    queryFn: () => {
      if (!token) {
        throw new Error("Missing auth token");
      }
      return api.me(token);
    },
    enabled: Boolean(token)
  });

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (me.isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-zinc-100 text-sm text-zinc-600">
        Loading session
      </div>
    );
  }

  if (me.isError) {
    clearToken();
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
