import { useState } from "react";
import { View, SafeAreaView } from "react-native";
import { useRouter } from "expo-router";
import { DayPicker } from "../../../components/calendar/DayPicker";
import { AgendaView } from "../../../components/calendar/AgendaView";
import { Button } from "../../../components/ui/Button";
import { useProviderAppointments } from "../../../hooks/useApi";

function formatToday(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export default function ProviderAgendaScreen() {
  const [selectedDate, setSelectedDate] = useState(formatToday());
  const router = useRouter();

  const { data: appointments, isLoading } = useProviderAppointments({
    date: selectedDate,
  });

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <View className="px-4 pt-4 bg-white pb-3">
        <DayPicker
          selectedDate={selectedDate}
          onSelectDate={setSelectedDate}
        />
      </View>
      <View className="flex-1 px-4 pt-3">
        <AgendaView
          appointments={appointments ?? []}
          onPressAppointment={(id) =>
            router.push(`/(provider)/agenda/${id}`)
          }
          loading={isLoading}
        />
      </View>
      <View className="px-4 pb-4">
        <Button
          title="+ Turno sin cita"
          variant="outline"
          onPress={() => router.push("/(provider)/agenda/walk-in" as never)}
        />
      </View>
    </SafeAreaView>
  );
}
