import { create } from "zustand";

const TOKEN_KEY = "push_booster_admin_token";

type AuthState = {
  token: string | null;
  setToken: (token: string) => void;
  clearToken: () => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem(TOKEN_KEY),
  setToken: (token: string) => {
    localStorage.setItem(TOKEN_KEY, token);
    set({ token });
  },
  clearToken: () => {
    localStorage.removeItem(TOKEN_KEY);
    set({ token: null });
  }
}));
