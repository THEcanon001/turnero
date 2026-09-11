import { useState } from "react";
import { View, Text, SafeAreaView, Alert } from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";

// Camera/barcode scanner requires expo-camera which may not be installed.
// For now, provide a manual slug entry as fallback.
export default function ScanScreen() {
  const router = useRouter();
  const [slug, setSlug] = useState("");

  const handleNavigate = () => {
    const trimmed = slug.trim().toLowerCase();
    if (!trimmed) {
      Alert.alert("Error", "Ingresá el código del proveedor.");
      return;
    }
    router.push(`/(client)/provider/${trimmed}`);
  };

  return (
    <SafeAreaView className="flex-1 bg-white items-center justify-center px-6">
      <Text className="text-2xl font-bold text-gray-800 mb-2">
        Escanear QR
      </Text>
      <Text className="text-base text-gray-500 mb-8 text-center">
        Ingresá el código del proveedor para acceder a su perfil
      </Text>

      <Input
        label="Código del proveedor"
        placeholder="barberia-juan"
        value={slug}
        onChangeText={(text) =>
          setSlug(text.toLowerCase().replace(/[^a-z0-9-]/g, ""))
        }
        autoCapitalize="none"
        autoCorrect={false}
      />

      <Button title="Ir al perfil" onPress={handleNavigate} />
    </SafeAreaView>
  );
}
