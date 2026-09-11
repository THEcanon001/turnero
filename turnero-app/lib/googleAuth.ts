import * as Google from "expo-auth-session/providers/google";
import * as WebBrowser from "expo-web-browser";
import type { AuthSessionResult } from "expo-auth-session";

// Complete any pending auth sessions on app load
WebBrowser.maybeCompleteAuthSession();

// Replace these with your actual Google Cloud OAuth client IDs
const GOOGLE_CLIENT_ID = "YOUR_GOOGLE_WEB_CLIENT_ID";
const GOOGLE_IOS_CLIENT_ID = "YOUR_GOOGLE_IOS_CLIENT_ID";
const GOOGLE_ANDROID_CLIENT_ID = "YOUR_GOOGLE_ANDROID_CLIENT_ID";

export function useGoogleAuth() {
  const [request, response, promptAsync] = Google.useIdTokenAuthRequest({
    clientId: GOOGLE_CLIENT_ID,
    iosClientId: GOOGLE_IOS_CLIENT_ID,
    androidClientId: GOOGLE_ANDROID_CLIENT_ID,
  });

  return {
    request,
    response,
    promptAsync,
  };
}

// Extract id_token from a successful Google auth response
export function extractIdToken(
  response: AuthSessionResult | null
): string | null {
  if (response?.type === "success") {
    return response.params.id_token ?? null;
  }
  return null;
}
