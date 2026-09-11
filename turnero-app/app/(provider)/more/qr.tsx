import { useState, useEffect } from "react";
import { View, Text, SafeAreaView, Image, Alert } from "react-native";
import { Button } from "../../../components/ui/Button";
import { useProviderMe } from "../../../hooks/useApi";
import { api, getAccessToken } from "../../../lib/api";

export default function QRScreen() {
  const { data: provider } = useProviderMe();
  const [token, setToken] = useState<string | null>(null);

  useEffect(() => {
    getAccessToken().then(setToken);
  }, []);

  const qrUrl =
    provider && token ? `${api.defaults.baseURL}/v1/provider/me/qr` : null;

  const handleShare = async () => {
    Alert.alert("Compartir", "Función de compartir disponible próximamente.");
  };

  return (
    <SafeAreaView className="flex-1 bg-white items-center justify-center px-8">
      <Text className="text-2xl font-bold text-gray-800 mb-2">
        Tu código QR
      </Text>
      <Text className="text-base text-gray-500 mb-8 text-center">
        Los clientes pueden escanear este código para agendar turnos
      </Text>

      {qrUrl ? (
        <View className="bg-white p-4 rounded-2xl shadow-sm border border-gray-100 mb-8">
          <Image
            source={{
              uri: qrUrl,
              headers: {
                Authorization: `Bearer ${token}`,
              },
            }}
            className="w-64 h-64"
            resizeMode="contain"
          />
        </View>
      ) : (
        <Text className="text-gray-400 py-8">Cargando QR...</Text>
      )}

      {provider ? (
        <Text className="text-sm text-gray-400 mb-6">
          turnero.app/{provider.slug}
        </Text>
      ) : null}

      <Button title="Compartir QR" variant="outline" onPress={handleShare} />
    </SafeAreaView>
  );
}
