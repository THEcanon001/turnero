import { useState } from "react";
import { View, Text, SafeAreaView, ScrollView } from "react-native";
import { Card } from "../../components/ui/Card";
import { Input } from "../../components/ui/Input";
import { useProviderStats } from "../../hooks/useApi";

function formatDate(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function getMonthRange(): { from: string; to: string } {
  const now = new Date();
  const from = new Date(now.getFullYear(), now.getMonth(), 1);
  const to = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return { from: formatDate(from), to: formatDate(to) };
}

export default function StatsScreen() {
  const defaultRange = getMonthRange();
  const [from, setFrom] = useState(defaultRange.from);
  const [to, setTo] = useState(defaultRange.to);

  const { data: stats, isLoading } = useProviderStats(from, to);

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <ScrollView className="flex-1 px-4 pt-4">
        <View className="flex-row gap-3 mb-4">
          <View className="flex-1">
            <Input
              label="Desde"
              value={from}
              onChangeText={setFrom}
              placeholder="YYYY-MM-DD"
            />
          </View>
          <View className="flex-1">
            <Input
              label="Hasta"
              value={to}
              onChangeText={setTo}
              placeholder="YYYY-MM-DD"
            />
          </View>
        </View>

        {isLoading ? (
          <Text className="text-gray-400 text-center py-8">
            Cargando estadísticas...
          </Text>
        ) : stats ? (
          <>
            <View className="flex-row gap-3 mb-4">
              <StatCard label="Total" value={stats.total} color="text-gray-800" />
              <StatCard label="Completados" value={stats.completed} color="text-green-600" />
            </View>
            <View className="flex-row gap-3 mb-4">
              <StatCard label="Cancelados" value={stats.cancelled} color="text-red-600" />
              <StatCard label="No asistió" value={stats.no_show} color="text-orange-600" />
            </View>
            <StatCard
              label="Confirmados"
              value={stats.confirmed}
              color="text-blue-600"
            />

            {stats.employees && stats.employees.length > 0 ? (
              <View className="mt-6">
                <Text className="text-lg font-semibold text-gray-800 mb-3">
                  Por empleado
                </Text>
                {stats.employees.map((emp) => (
                  <Card key={emp.employee_id} className="mb-3">
                    <Text className="text-base font-semibold text-gray-800 mb-2">
                      {emp.employee_name}
                    </Text>
                    <View className="flex-row justify-between">
                      <Text className="text-sm text-gray-500">
                        Completados: {emp.stats.completed}
                      </Text>
                      <Text className="text-sm text-gray-500">
                        Cancelados: {emp.stats.cancelled}
                      </Text>
                      <Text className="text-sm text-gray-500">
                        No-show: {emp.stats.no_show}
                      </Text>
                    </View>
                  </Card>
                ))}
              </View>
            ) : null}
          </>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}

function StatCard({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: string;
}) {
  return (
    <Card className="flex-1 items-center py-6">
      <Text className={`text-3xl font-bold ${color}`}>{value}</Text>
      <Text className="text-sm text-gray-500 mt-1">{label}</Text>
    </Card>
  );
}
