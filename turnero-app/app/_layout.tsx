import "../global.css";
import { useEffect } from "react";
import { Stack, useRouter, useSegments } from "expo-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import { useAuthStore } from "../stores/auth";

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

export default function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthGuard>
        <StatusBar style="dark" />
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
