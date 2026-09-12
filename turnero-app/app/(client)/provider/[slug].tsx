import { useEffect } from "react";
import { View, Text, FlatList, ActivityIndicator, TouchableOpacity } from "react-native";
import { useLocalSearchParams, useRouter, Stack } from "expo-router";
import { useProviderProfile } from "../../../hooks/useApi";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { openWhatsApp } from "../../../lib/whatsapp";
import { useAppStore } from "../../../stores/appStore";
import type { Service } from "../../../types/api";

export default function ProviderProfileScreen() {
  const { slug } = useLocalSearchParams<{ slug: string }>();
  const router = useRouter();
  const { data: provider, isLoading, error } = useProviderProfile(slug ?? "");
  const addRecent = useAppStore((s) => s.addRecent);

  useEffect(() => {
    if (provider) {
      addRecent({ slug: provider.slug, name: provider.name, type: provider.type });
    }
  }, [provider, addRecent]);

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Stack.Screen options={{ headerShown: true, title: "Cargando..." }} />
        <ActivityIndicator size="large" color="#6366F1" />
      </View>
    );
  }

  if (error || !provider) {
    return (
      <View className="flex-1">
        <Stack.Screen options={{ headerShown: true, title: "Error" }} />
        <EmptyState
          title="No encontrado"
          message="No se pudo cargar el perfil del profesional."
        />
      </View>
    );
  }

  const renderService = ({ item }: { item: Service }) => (
    <Card className="mb-3">
      <View className="flex-row justify-between items-center">
        <View className="flex-1">
          <Text className="text-base font-semibold text-gray-900">
            {item.name}
          </Text>
          {item.description && (
            <Text className="text-sm text-gray-500 mt-1">
              {item.description}
            </Text>
          )}
          <Text className="text-sm text-primary-600 mt-1">
            {item.duration_minutes} min
          </Text>
        </View>
      </View>
    </Card>
  );

  return (
    <View className="flex-1">
      <Stack.Screen
        options={{ headerShown: true, title: provider.name }}
      />
      <FlatList
        data={provider.services.filter((s) => s.is_active)}
        renderItem={renderService}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ padding: 16 }}
        ListHeaderComponent={
          <View className="mb-4">
            <Text className="text-2xl font-bold text-gray-900">
              {provider.name}
            </Text>
            <Text className="text-sm text-gray-500 mt-1">
              {provider.type === "business" ? "Negocio" : "Profesional"}
            </Text>
            {provider.address && (
              <Text className="text-sm text-gray-400 mt-1">
                {provider.address}
              </Text>
            )}

            <View className="flex-row gap-3 mt-4">
              <Button
                title="Agendar turno"
                onPress={() =>
                  router.push(`/(client)/book/${slug}`)
                }
                className="flex-1"
              />
              <Button
                title="WhatsApp"
                variant="outline"
                onPress={() =>
                  openWhatsApp(
                    provider.phone,
                    `Hola! Quisiera agendar un turno.`
                  )
                }
                className="flex-1"
              />
            </View>

            <Text className="text-lg font-semibold text-gray-800 mt-6 mb-2">
              Servicios
            </Text>
          </View>
        }
        ListEmptyComponent={
          <EmptyState
            title="Sin servicios"
            message="Este profesional aún no tiene servicios configurados."
          />
        }
      />
    </View>
  );
}
