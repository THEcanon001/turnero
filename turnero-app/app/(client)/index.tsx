import { useState } from "react";
import {
  View,
  Text,
  SafeAreaView,
  ScrollView,
  TouchableOpacity,
} from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";
import { useAppStore } from "../../stores/appStore";

export default function ClientHomeScreen() {
  const [searchTerm, setSearchTerm] = useState("");
  const router = useRouter();
  const recentProviders = useAppStore((s) => s.recentProviders);

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

        {recentProviders.length > 0 && (
          <View className="mt-6">
            <Text className="text-lg font-semibold text-gray-800 mb-3">
              Recientes
            </Text>
            <ScrollView horizontal showsHorizontalScrollIndicator={false}>
              {recentProviders.map((rp) => (
                <TouchableOpacity
                  key={rp.slug}
                  className="bg-gray-100 rounded-xl px-4 py-3 mr-3 items-center"
                  style={{ minWidth: 100 }}
                  onPress={() => router.push(`/(client)/provider/${rp.slug}`)}
                >
                  <View className="w-10 h-10 rounded-full bg-primary-100 items-center justify-center mb-2">
                    <Text className="text-primary-700 font-bold text-base">
                      {rp.name.charAt(0).toUpperCase()}
                    </Text>
                  </View>
                  <Text
                    className="text-sm font-medium text-gray-900 text-center"
                    numberOfLines={1}
                  >
                    {rp.name}
                  </Text>
                  <Text className="text-xs text-gray-500" numberOfLines={1}>
                    {rp.type === "business" ? "Negocio" : "Profesional"}
                  </Text>
                </TouchableOpacity>
              ))}
            </ScrollView>
          </View>
        )}

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
