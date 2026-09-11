import { View, Text, SafeAreaView, FlatList, TouchableOpacity } from "react-native";
import { useRouter } from "expo-router";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { useEmployees } from "../../../hooks/useApi";

export default function EmployeesListScreen() {
  const router = useRouter();
  const { data: employees, isLoading } = useEmployees();

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Cargando...</Text>
      </View>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <View className="px-4 pt-4 pb-2 gap-2">
        <Button
          title="+ Invitar empleado"
          onPress={() => router.push("/(provider)/employees/invite" as never)}
        />
      </View>
      {!employees || employees.length === 0 ? (
        <EmptyState
          title="Sin empleados"
          message="Invitá a tu primer empleado"
        />
      ) : (
        <FlatList
          data={employees}
          keyExtractor={(item) => item.id}
          contentContainerClassName="px-4 pt-2 pb-4"
          renderItem={({ item }) => (
            <TouchableOpacity
              onPress={() =>
                router.push(`/(provider)/employees/${item.id}` as never)
              }
            >
              <Card className="mb-3">
                <View className="flex-row justify-between items-center">
                  <View>
                    <Text className="text-base font-semibold text-gray-800">
                      {item.name}
                    </Text>
                    <Text className="text-sm text-gray-500">{item.phone}</Text>
                  </View>
                  <View className="flex-row items-center gap-2">
                    <Text className="text-xs text-gray-400">{item.role}</Text>
                    <View
                      className={`w-2 h-2 rounded-full ${
                        item.is_active ? "bg-green-500" : "bg-gray-300"
                      }`}
                    />
                  </View>
                </View>
              </Card>
            </TouchableOpacity>
          )}
        />
      )}
    </SafeAreaView>
  );
}
