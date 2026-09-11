import { useState } from "react";
import { View, Text, ActivityIndicator, Alert, ScrollView } from "react-native";
import { useLocalSearchParams, useRouter, Stack } from "expo-router";
import {
  useAppointment,
  useCancelAppointment,
  useRescheduleAppointment,
} from "../../../hooks/useApi";
import { Card } from "../../../components/ui/Card";
import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { DayPicker } from "../../../components/calendar/DayPicker";
import { EmptyState } from "../../../components/ui/EmptyState";
import { openWhatsApp } from "../../../lib/whatsapp";
import { getApiErrorMessage } from "../../../lib/api";

export default function AppointmentDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data, isLoading, error } = useAppointment(id ?? "");
  const cancelMutation = useCancelAppointment();
  const rescheduleMutation = useRescheduleAppointment();

  const [showReschedule, setShowReschedule] = useState(false);
  const [newDate, setNewDate] = useState("");
  const [newTime, setNewTime] = useState("");

  if (isLoading) {
    return (
      <View className="flex-1 items-center justify-center">
        <Stack.Screen options={{ headerShown: true, title: "Turno" }} />
        <ActivityIndicator size="large" color="#6366F1" />
      </View>
    );
  }

  if (error || !data) {
    return (
      <View className="flex-1">
        <Stack.Screen options={{ headerShown: true, title: "Error" }} />
        <EmptyState
          title="Turno no encontrado"
          message="No se pudo cargar la información del turno."
        />
      </View>
    );
  }

  const { appointment, whatsapp_link } = data;

  const statusLabels: Record<string, string> = {
    confirmed: "Confirmado",
    cancelled: "Cancelado",
    completed: "Completado",
    no_show: "No asistió",
  };

  const statusColors: Record<string, string> = {
    confirmed: "text-green-600",
    cancelled: "text-red-500",
    completed: "text-blue-600",
    no_show: "text-yellow-600",
  };

  const handleCancel = () => {
    Alert.alert(
      "Cancelar turno",
      "¿Estás seguro que querés cancelar este turno?",
      [
        { text: "No", style: "cancel" },
        {
          text: "Sí, cancelar",
          style: "destructive",
          onPress: async () => {
            try {
              await cancelMutation.mutateAsync({ id: id! });
              Alert.alert("Listo", "Tu turno fue cancelado.");
            } catch (err) {
              Alert.alert("Error", getApiErrorMessage(err));
            }
          },
        },
      ]
    );
  };

  const handleReschedule = async () => {
    if (!newDate || !newTime.trim()) {
      Alert.alert("Error", "Seleccioná una fecha y un horario.");
      return;
    }
    try {
      await rescheduleMutation.mutateAsync({
        id: id!,
        date: newDate,
        start_time: newTime.trim(),
      });
      Alert.alert("Listo", "Tu turno fue reprogramado.");
      setShowReschedule(false);
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    }
  };

  return (
    <View className="flex-1">
      <Stack.Screen options={{ headerShown: true, title: "Detalle del turno" }} />
      <ScrollView contentContainerStyle={{ padding: 16, paddingBottom: 40 }}>
        <Card className="mb-4">
          <Text className="text-xl font-bold text-gray-900">
            {appointment.client_name}
          </Text>
          <View className="mt-3 gap-2">
            <View className="flex-row justify-between">
              <Text className="text-gray-500">Estado</Text>
              <Text
                className={`font-semibold ${statusColors[appointment.status] ?? "text-gray-800"}`}
              >
                {statusLabels[appointment.status] ?? appointment.status}
              </Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-gray-500">Fecha</Text>
              <Text className="text-gray-800">{appointment.date}</Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-gray-500">Horario</Text>
              <Text className="text-gray-800">
                {appointment.start_time} - {appointment.end_time}
              </Text>
            </View>
            <View className="flex-row justify-between">
              <Text className="text-gray-500">Teléfono</Text>
              <Text className="text-gray-800">{appointment.client_phone}</Text>
            </View>
            {appointment.notes && (
              <View>
                <Text className="text-gray-500">Notas</Text>
                <Text className="text-gray-800">{appointment.notes}</Text>
              </View>
            )}
          </View>
        </Card>

        {appointment.status === "confirmed" && (
          <View className="gap-3">
            {whatsapp_link && (
              <Button
                title="WhatsApp"
                variant="outline"
                onPress={() =>
                  openWhatsApp(appointment.client_phone, "Consulta sobre turno")
                }
              />
            )}
            <Button
              title="Reprogramar"
              variant="secondary"
              onPress={() => setShowReschedule(!showReschedule)}
            />
            <Button
              title="Cancelar turno"
              variant="danger"
              onPress={handleCancel}
              loading={cancelMutation.isPending}
            />
          </View>
        )}

        {showReschedule && (
          <Card className="mt-4">
            <Text className="text-lg font-semibold text-gray-800 mb-3">
              Reprogramar
            </Text>
            <DayPicker
              selectedDate={newDate}
              onSelectDate={setNewDate}
            />
            <Input
              label="Nuevo horario (HH:MM)"
              placeholder="09:30"
              value={newTime}
              onChangeText={setNewTime}
              className="mt-4"
            />
            <Button
              title="Confirmar nueva fecha"
              onPress={handleReschedule}
              loading={rescheduleMutation.isPending}
              className="mt-2"
            />
          </Card>
        )}
      </ScrollView>
    </View>
  );
}
