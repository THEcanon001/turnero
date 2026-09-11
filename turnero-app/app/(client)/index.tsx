import { useState } from "react";
import { View, Text, SafeAreaView } from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";

export default function ClientHomeScreen() {
  const [searchTerm, setSearchTerm] = useState("");
  const router = useRouter();

  const handleSearch = () => {
    if (searchTerm.trim().length >= 2) {
      router.push(`/(client)/search?q=${encodeURIComponent(searchTerm.trim())}`);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-white">
      <View className="flex-1 justify-center px-6">
        <Text className="text-4xl font-bold text-gray-900 text-center">
          Turnero
        </Text>
        <Text className="text-base text-gray-500 text-center mt-2 mb-8">
          Encontrá y agendá turnos fácilmente
        </Text>

        <Input
          placeholder="Buscar por nombre o negocio..."
          value={searchTerm}
          onChangeText={setSearchTerm}
          onSubmitEditing={handleSearch}
          returnKeyType="search"
        />

        <Button
          title="Buscar"
          onPress={handleSearch}
          disabled={searchTerm.trim().length < 2}
        />

        <View className="mt-8">
          <Button
            title="Mis turnos"
            variant="outline"
            onPress={() => router.push("/(client)/appointments")}
          />
        </View>
      </View>
    </SafeAreaView>
  );
}
