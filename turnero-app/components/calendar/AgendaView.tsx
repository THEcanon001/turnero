import { View, Text, TouchableOpacity, ScrollView } from "react-native";
import type { Appointment } from "../../types/api";

interface AgendaViewProps {
  appointments: Appointment[];
  onPressAppointment: (id: string) => void;
  loading?: boolean;
}

const STATUS_COLORS: Record<string, string> = {
  confirmed: "bg-blue-100 text-blue-700",
  completed: "bg-green-100 text-green-700",
  cancelled: "bg-gray-100 text-gray-500",
  no_show: "bg-red-100 text-red-700",
};

const STATUS_LABELS: Record<string, string> = {
  confirmed: "Confirmado",
  completed: "Completado",
  cancelled: "Cancelado",
  no_show: "No asistió",
};

export function AgendaView({
  appointments,
  onPressAppointment,
  loading,
}: AgendaViewProps) {
  if (loading) {
    return (
      <View className="py-8 items-center">
        <Text className="text-gray-400">Cargando turnos...</Text>
      </View>
    );
  }

  if (appointments.length === 0) {
    return (
      <View className="py-8 items-center">
        <Text className="text-gray-400">No hay turnos para este día.</Text>
      </View>
    );
  }

  const sorted = [...appointments].sort((a, b) =>
    a.start_time.localeCompare(b.start_time)
  );

  return (
    <ScrollView className="flex-1" showsVerticalScrollIndicator={false}>
      {sorted.map((apt) => {
        const statusStyle = STATUS_COLORS[apt.status] ?? "bg-gray-100 text-gray-500";
        const [bgColor, textColor] = statusStyle.split(" ");
        return (
          <TouchableOpacity
            key={apt.id}
            onPress={() => onPressAppointment(apt.id)}
            className="flex-row items-center bg-white rounded-xl p-4 mb-2 border border-gray-100"
          >
            <View className="mr-4 items-center">
              <Text className="text-lg font-bold text-gray-800">
                {apt.start_time}
              </Text>
              <Text className="text-xs text-gray-400">{apt.end_time}</Text>
            </View>
            <View className="flex-1">
              <Text className="text-base font-semibold text-gray-800">
                {apt.client_name}
              </Text>
              <Text className="text-sm text-gray-500">{apt.client_phone}</Text>
            </View>
            <View className={`px-2 py-1 rounded-lg ${bgColor}`}>
              <Text className={`text-xs font-medium ${textColor}`}>
                {STATUS_LABELS[apt.status] ?? apt.status}
              </Text>
            </View>
          </TouchableOpacity>
        );
      })}
    </ScrollView>
  );
}
