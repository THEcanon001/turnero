import { useState, useEffect } from "react";
import { View, Text, TouchableOpacity, Platform } from "react-native";

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: "accepted" | "dismissed" }>;
}

export function WebInstallBanner() {
  const [deferredPrompt, setDeferredPrompt] =
    useState<BeforeInstallPromptEvent | null>(null);
  const [dismissed, setDismissed] = useState(false);

  useEffect(() => {
    if (Platform.OS !== "web") return;

    const handler = (e: Event) => {
      e.preventDefault();
      setDeferredPrompt(e as BeforeInstallPromptEvent);
    };

    window.addEventListener("beforeinstallprompt", handler);
    return () => window.removeEventListener("beforeinstallprompt", handler);
  }, []);

  if (Platform.OS !== "web" || !deferredPrompt || dismissed) return null;

  const handleInstall = async () => {
    await deferredPrompt.prompt();
    const { outcome } = await deferredPrompt.userChoice;
    if (outcome === "accepted") {
      setDeferredPrompt(null);
    }
    setDismissed(true);
  };

  return (
    <View className="bg-primary-50 border border-primary-200 rounded-xl p-4 mx-4 mt-4 flex-row items-center justify-between">
      <Text className="text-sm text-primary-800 flex-1 mr-3">
        Instalá Turnero en tu dispositivo para acceso rápido.
      </Text>
      <View className="flex-row gap-2">
        <TouchableOpacity onPress={handleInstall}>
          <Text className="text-sm font-semibold text-primary-600">
            Instalar
          </Text>
        </TouchableOpacity>
        <TouchableOpacity onPress={() => setDismissed(true)}>
          <Text className="text-sm text-gray-400">Cerrar</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}
