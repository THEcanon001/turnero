import { Platform } from "react-native";
import { api } from "./api";

// Register push token with the backend.
// Call this on app launch after authentication.
export async function registerPushToken(token: string): Promise<void> {
  const platform = Platform.OS === "ios" ? "ios" : "android";
  await api.post("/v1/push-tokens", { token, platform });
}

// Deregister push token from the backend.
export async function deregisterPushToken(): Promise<void> {
  await api.delete("/v1/push-tokens");
}
