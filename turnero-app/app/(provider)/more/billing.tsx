import { View, Text, SafeAreaView, ScrollView, FlatList } from "react-native";
import { Card } from "../../../components/ui/Card";
import {
  useBillingUsage,
  useBillingHistory,
  useBillingTransactions,
} from "../../../hooks/useApi";

export default function BillingScreen() {
  const { data: usage, isLoading: loadingUsage } = useBillingUsage();
  const { data: history } = useBillingHistory();
  const { data: transactions } = useBillingTransactions();

  if (loadingUsage) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Cargando...</Text>
      </View>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <ScrollView className="flex-1 px-4 pt-4">
        {usage ? (
          <Card className="mb-4">
            <Text className="text-lg font-semibold text-gray-800 mb-3">
              Uso del mes actual
            </Text>
            <View className="flex-row justify-between mb-2">
              <Text className="text-sm text-gray-500">Completados</Text>
              <Text className="text-sm font-medium text-gray-800">
                {usage.completed_appointments}
              </Text>
            </View>
            <View className="flex-row justify-between mb-2">
              <Text className="text-sm text-gray-500">Gratis incluidos</Text>
              <Text className="text-sm font-medium text-gray-800">
                {usage.free_tier_limit}
              </Text>
            </View>
            <View className="flex-row justify-between mb-2">
              <Text className="text-sm text-gray-500">Facturables</Text>
              <Text className="text-sm font-medium text-gray-800">
                {usage.billable_appointments}
              </Text>
            </View>

            {/* Usage bar */}
            <View className="h-3 bg-gray-200 rounded-full mt-2 mb-3">
              <View
                className="h-3 bg-primary-600 rounded-full"
                style={{
                  width: `${Math.min(
                    100,
                    (usage.completed_appointments / Math.max(usage.free_tier_limit, 1)) * 100
                  )}%`,
                }}
              />
            </View>

            <View className="flex-row justify-between items-center">
              <Text className="text-sm text-gray-500">Monto a pagar</Text>
              <Text className="text-xl font-bold text-primary-600">
                ${usage.amount_due.toFixed(2)} MXN
              </Text>
            </View>
            <View className="flex-row justify-between mt-1">
              <Text className="text-sm text-gray-500">Estado</Text>
              <Text
                className={`text-sm font-medium ${
                  usage.is_paid ? "text-green-600" : "text-orange-600"
                }`}
              >
                {usage.is_paid ? "Pagado" : "Pendiente"}
              </Text>
            </View>
          </Card>
        ) : null}

        {history && history.length > 0 ? (
          <View className="mb-4">
            <Text className="text-lg font-semibold text-gray-800 mb-3">
              Historial de uso
            </Text>
            {history.map((item) => (
              <Card key={item.month} className="mb-2">
                <View className="flex-row justify-between">
                  <Text className="text-sm text-gray-600">{item.month}</Text>
                  <Text className="text-sm text-gray-500">
                    {item.completed_appointments} turnos
                  </Text>
                  <Text className="text-sm font-medium text-gray-800">
                    ${item.amount_due.toFixed(2)}
                  </Text>
                </View>
              </Card>
            ))}
          </View>
        ) : null}

        {transactions && transactions.length > 0 ? (
          <View className="mb-8">
            <Text className="text-lg font-semibold text-gray-800 mb-3">
              Transacciones
            </Text>
            {transactions.map((tx) => (
              <Card key={tx.id} className="mb-2">
                <View className="flex-row justify-between">
                  <View>
                    <Text className="text-sm text-gray-800">
                      {tx.description ?? "Pago"}
                    </Text>
                    <Text className="text-xs text-gray-400">
                      {tx.created_at.slice(0, 10)}
                    </Text>
                  </View>
                  <Text className="text-sm font-medium text-gray-800">
                    ${tx.amount.toFixed(2)} {tx.currency}
                  </Text>
                </View>
              </Card>
            ))}
          </View>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}
