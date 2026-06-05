import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  BarChart3,
  Building2,
  CircleDollarSign,
  Inbox,
  KeyRound,
  LogOut,
  Megaphone,
  Users
} from "lucide-react";
import { NavLink, Outlet, useNavigate } from "react-router";

import { api } from "../api/client";
import { cn } from "../lib/utils";
import { useAuthStore } from "../store/auth";

export function AppShell() {
  const token = useAuthStore((state) => state.token);
  const clearToken = useAuthStore((state) => state.clearToken);
  const navigate = useNavigate();
  const queryClient = useQueryClient();

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

  function logout() {
    clearToken();
    queryClient.clear();
    void navigate("/login", { replace: true });
  }

  return (
    <div className="min-h-screen bg-zinc-100 text-zinc-950">
      <header className="border-b border-zinc-200 bg-white">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-4">
          <div className="flex items-center gap-6">
            <div className="font-semibold">Push Booster</div>
            <nav className="flex items-center gap-1">
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/users"
              >
                <Users className="h-4 w-4" />
                Users
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/publishers"
              >
                <Building2 className="h-4 w-4" />
                Publishers
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/campaigns"
              >
                <Megaphone className="h-4 w-4" />
                Campaigns
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/reports"
              >
                <BarChart3 className="h-4 w-4" />
                Reports
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/costs"
              >
                <CircleDollarSign className="h-4 w-4" />
                Costs
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/postbacks"
              >
                <Inbox className="h-4 w-4" />
                Postbacks
              </NavLink>
              <NavLink
                className={({ isActive }) =>
                  cn(
                    "inline-flex h-9 items-center gap-2 rounded-md px-3 text-sm text-zinc-700 transition hover:bg-zinc-100",
                    isActive && "bg-zinc-950 text-white hover:bg-zinc-950"
                  )
                }
                to="/vapid-keys"
              >
                <KeyRound className="h-4 w-4" />
                VAPID
              </NavLink>
            </nav>
          </div>
          <div className="flex items-center gap-3">
            <div className="hidden text-right sm:block">
              <div className="text-sm font-medium">{me.data?.email}</div>
              <div className="text-xs text-zinc-500">{me.data?.role}</div>
            </div>
            <button
              className="inline-flex h-9 items-center gap-2 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-700 transition hover:bg-zinc-50"
              type="button"
              onClick={logout}
            >
              <LogOut className="h-4 w-4" />
              Logout
            </button>
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-4 py-6">
        <Outlet />
      </main>
    </div>
  );
}
