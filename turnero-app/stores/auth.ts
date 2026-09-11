import { create } from "zustand";
import { api, setTokens, clearTokens, getAccessToken } from "../lib/api";
import type {
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
} from "../types/api";

export type AuthStatus = "loading" | "authenticated" | "unauthenticated";

interface AuthUser {
  id: string;
  name: string;
  role: string;
  type: string;
  provider_id: string;
}

interface AuthState {
  status: AuthStatus;
  user: AuthUser | null;
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  checkAuth: () => Promise<void>;
}

export const useAuthStore = create<AuthState>((set) => ({
  status: "loading",
  user: null,

  login: async (data: LoginRequest) => {
    const { data: res } = await api.post<LoginResponse>(
      "/v1/auth/login",
      data
    );
    await setTokens(res.access_token, res.refresh_token);
    set({
      status: "authenticated",
      user: res.user,
    });
  },

  register: async (data: RegisterRequest) => {
    const { data: res } = await api.post<RegisterResponse>(
      "/v1/auth/register",
      data
    );
    await setTokens(res.access_token, res.refresh_token);
    set({
      status: "authenticated",
      user: {
        id: res.id,
        name: res.name,
        role: "admin",
        type: res.type,
        provider_id: res.id,
      },
    });
  },

  logout: async () => {
    await clearTokens();
    set({ status: "unauthenticated", user: null });
  },

  checkAuth: async () => {
    const token = await getAccessToken();
    if (!token) {
      set({ status: "unauthenticated" });
      return;
    }

    // Restore user data from the API
    try {
      const { data: provider } = await api.get("/v1/provider/me");
      set({
        status: "authenticated",
        user: {
          id: provider.id,
          name: provider.name,
          role: "admin",
          type: provider.type,
          provider_id: provider.id,
        },
      });
    } catch {
      // Token might be expired and refresh failed — treat as unauthenticated
      await clearTokens();
      set({ status: "unauthenticated", user: null });
    }
  },
}));
