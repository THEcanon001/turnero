import { View, Text, SafeAreaView, FlatList, TouchableOpacity } from "react-native";
import { useRouter } from "expo-router";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { useServices } from "../../../hooks/useApi";

export default function ServicesListScreen() {
  const router = useRouter();
  const { data: services, isLoading } = useServices();

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Cargando...</Text>
      </View>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <View className="px-4 pt-4 pb-2">
        <Button
          title="+ Nuevo servicio"
          onPress={() => router.push("/(provider)/services/create" as never)}
        />
      </View>
      {!services || services.length === 0 ? (
        <EmptyState
          title="Sin servicios"
          message="Creá tu primer servicio para empezar"
        />
      ) : (
        <FlatList
          data={services}
          keyExtractor={(item) => item.id}
          contentContainerClassName="px-4 pt-2 pb-4"
          renderItem={({ item }) => (
            <TouchableOpacity
              onPress={() =>
                router.push(`/(provider)/services/${item.id}` as never)
              }
            >
              <Card className="mb-3">
                <View className="flex-row justify-between items-center">
                  <View className="flex-1">
                    <Text className="text-base font-semibold text-gray-800">
                      {item.name}
                    </Text>
                    {item.description ? (
                      <Text className="text-sm text-gray-500 mt-1">
                        {item.description}
                      </Text>
                    ) : null}
                  </View>
                  <Text className="text-sm text-gray-400">
                    {item.duration_minutes} min
                  </Text>
                </View>
              </Card>
            </TouchableOpacity>
          )}
        />
      )}
    </SafeAreaView>
  );
}
