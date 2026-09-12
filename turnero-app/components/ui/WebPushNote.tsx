import { useState } from "react";
import { Platform, View, Text, TouchableOpacity } from "react-native";

export function WebPushNote() {
  const [dismissed, setDismissed] = useState(false);

  if (Platform.OS !== "web" || dismissed) return null;

  // Don't show if already running as installed PWA
  if (typeof window !== "undefined" && window.matchMedia("(display-mode: standalone)").matches) {
    return null;
  }

  return (
    <View className="bg-yellow-50 border border-yellow-200 rounded-xl p-3 mx-4 mt-2 flex-row items-center justify-between">
      <Text className="text-sm text-yellow-800 flex-1 mr-3">
        Descargá la app para recibir notificaciones push.
      </Text>
      <TouchableOpacity onPress={() => setDismissed(true)}>
        <Text className="text-sm text-gray-400">Cerrar</Text>
      </TouchableOpacity>
    </View>
  );
}
