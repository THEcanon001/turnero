import "../global.css";
import { useEffect } from "react";
import { Platform } from "react-native";
import { Stack, useRouter, useSegments } from "expo-router";
import * as Linking from "expo-linking";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import { useAuthStore } from "../stores/auth";
import { WebInstallBanner } from "../components/ui/WebInstallBanner";
import { WebPushNote } from "../components/ui/WebPushNote";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 30_000,
    },
  },
});

function AuthGuard({ children }: { children: React.ReactNode }) {
  const { status, user, checkAuth } = useAuthStore();
  const segments = useSegments();
  const router = useRouter();

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (status === "loading") return;

    const inAuthGroup = segments[0] === "(auth)";

    if (status === "unauthenticated" && !inAuthGroup) {
      // Client routes are public — only redirect provider/employee areas
      const inProviderGroup = segments[0] === "(provider)";
      const inEmployeeGroup = segments[0] === "(employee)";
      if (inProviderGroup || inEmployeeGroup) {
        router.replace("/(auth)/login");
      }
    } else if (status === "authenticated" && inAuthGroup) {
      // Redirect based on role
      if (user?.role === "admin") {
        router.replace("/(provider)" as never);
      } else {
        router.replace("/(client)");
      }
    }
  }, [status, segments, router, user]);

  return <>{children}</>;
}

function DeepLinkHandler() {
  const router = useRouter();

  useEffect(() => {
    // Handle deep links: turnero://{slug} or https://turnero.app/{slug}
    const handleDeepLink = (event: { url: string }) => {
      const url = event.url;
      const parsed = Linking.parse(url);

      // If the path is a slug (single segment, no special route prefix)
      if (parsed.path && !parsed.path.includes("/")) {
        const slug = parsed.path;
        router.push(`/(client)/provider/${slug}`);
      }
    };

    // On web, expo-router handles URL routing natively — only use deep links on native
    if (Platform.OS === "web") return;

    // Handle URL that launched the app
    Linking.getInitialURL().then((url) => {
      if (url) handleDeepLink({ url });
    });

    // Handle incoming URLs while app is open
    const subscription = Linking.addEventListener("url", handleDeepLink);
    return () => subscription.remove();
  }, [router]);

  return null;
}

export default function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthGuard>
        <StatusBar style="dark" />
        <DeepLinkHandler />
        {Platform.OS === "web" && <WebInstallBanner />}
        {Platform.OS === "web" && <WebPushNote />}
        <Stack
          screenOptions={{
            headerShown: false,
            contentStyle: { backgroundColor: "#FFFFFF" },
          }}
        />
      </AuthGuard>
    </QueryClientProvider>
  );
}
