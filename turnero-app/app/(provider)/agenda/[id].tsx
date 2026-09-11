import { useState } from "react";
import { View, Text, ScrollView, Alert, Linking } from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import {
  useAppointment,
  useUpdateAppointmentStatus,
  useProviderCancel,
} from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

export default function AppointmentDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading } = useAppointment(id);
  const updateStatus = useUpdateAppointmentStatus();
  const providerCancel = useProviderCancel();
  const [actionLoading, setActionLoading] = useState("");

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Cargando...</Text>
      </View>
    );
  }

  const apt = data?.appointment;
  if (!apt) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Turno no encontrado</Text>
      </View>
    );
  }

  const handleAction = async (action: string) => {
    setActionLoading(action);
    try {
      if (action === "cancel") {
        await providerCancel.mutateAsync({ id: apt.id });
      } else {
        await updateStatus.mutateAsync({ id: apt.id, status: action });
      }
      router.back();
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setActionLoading("");
    }
  };

  const whatsappUrl = data?.whatsapp_link;

  return (
    <ScrollView className="flex-1 bg-gray-50 p-4">
      <Card className="mb-4">
        <Text className="text-xl font-bold text-gray-800 mb-1">
          {apt.client_name}
        </Text>
        <Text className="text-base text-gray-500 mb-3">
          {apt.client_phone}
        </Text>
        <View className="flex-row justify-between mb-2">
          <Text className="text-sm text-gray-500">Fecha</Text>
          <Text className="text-sm font-medium text-gray-800">
            {apt.date}
          </Text>
        </View>
        <View className="flex-row justify-between mb-2">
          <Text className="text-sm text-gray-500">Horario</Text>
          <Text className="text-sm font-medium text-gray-800">
            {apt.start_time} - {apt.end_time}
          </Text>
        </View>
        <View className="flex-row justify-between mb-2">
          <Text className="text-sm text-gray-500">Estado</Text>
          <Text className="text-sm font-medium text-gray-800">
            {apt.status}
          </Text>
        </View>
        {apt.notes ? (
          <View className="flex-row justify-between">
            <Text className="text-sm text-gray-500">Notas</Text>
            <Text className="text-sm text-gray-800">{apt.notes}</Text>
          </View>
        ) : null}
      </Card>

      {apt.status === "confirmed" && (
        <View className="gap-3">
          <Button
            title="Completar"
            onPress={() => handleAction("completed")}
            loading={actionLoading === "completed"}
          />
          <Button
            title="No asistió"
            variant="secondary"
            onPress={() => handleAction("no_show")}
            loading={actionLoading === "no_show"}
          />
          <Button
            title="Cancelar turno"
            variant="danger"
            onPress={() =>
              Alert.alert("Cancelar", "¿Estás seguro?", [
                { text: "No" },
                { text: "Sí", onPress: () => handleAction("cancel") },
              ])
            }
            loading={actionLoading === "cancel"}
          />
        </View>
      )}

      {whatsappUrl ? (
        <Button
          title="Enviar WhatsApp"
          variant="outline"
          className="mt-4"
          onPress={() => Linking.openURL(whatsappUrl)}
        />
      ) : null}
    </ScrollView>
  );
}
