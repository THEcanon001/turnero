import { View, Text, FlatList, TouchableOpacity, ActivityIndicator } from "react-native";
import { useLocalSearchParams, useRouter, Stack } from "expo-router";
import { useSearch } from "../../hooks/useApi";
import { Card } from "../../components/ui/Card";
import { EmptyState } from "../../components/ui/EmptyState";
import type { SearchResult } from "../../types/api";

export default function SearchScreen() {
  const { q } = useLocalSearchParams<{ q: string }>();
  const router = useRouter();
  const { data: results, isLoading, error } = useSearch(q ?? "");

  const renderResult = ({ item }: { item: SearchResult }) => (
    <TouchableOpacity
      onPress={() => router.push(`/(client)/provider/${item.slug}`)}
    >
      <Card className="mb-3">
        <Text className="text-lg font-semibold text-gray-900">
          {item.name}
        </Text>
        <Text className="text-sm text-gray-500 mt-1">
          {item.type === "business" ? "Negocio" : "Profesional"}
        </Text>
        {item.address && (
          <Text className="text-sm text-gray-400 mt-1">
            {item.address}
          </Text>
        )}
      </Card>
    </TouchableOpacity>
  );

  return (
    <View className="flex-1">
      <Stack.Screen
        options={{ headerShown: true, title: `Resultados: "${q}"` }}
      />
      {isLoading ? (
        <View className="flex-1 items-center justify-center">
          <ActivityIndicator size="large" color="#6366F1" />
        </View>
      ) : error ? (
        <EmptyState
          title="Error al buscar"
          message="No se pudieron cargar los resultados. Intentá de nuevo."
        />
      ) : (
        <FlatList
          data={results}
          renderItem={renderResult}
          keyExtractor={(item) => item.id}
          contentContainerStyle={{ padding: 16 }}
          ListEmptyComponent={
            <EmptyState
              title="Sin resultados"
              message={`No encontramos nada para "${q}".`}
            />
          }
        />
      )}
    </View>
  );
}
