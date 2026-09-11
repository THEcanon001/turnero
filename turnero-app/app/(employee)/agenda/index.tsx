import { useState } from "react";
import { View, SafeAreaView } from "react-native";
import { useRouter } from "expo-router";
import { DayPicker } from "../../../components/calendar/DayPicker";
import { AgendaView } from "../../../components/calendar/AgendaView";
import { useProviderAppointments } from "../../../hooks/useApi";
import { useAuthStore } from "../../../stores/auth";

function formatToday(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

export default function EmployeeAgendaScreen() {
  const [selectedDate, setSelectedDate] = useState(formatToday());
  const router = useRouter();
  const user = useAuthStore((s) => s.user);

  const { data: appointments, isLoading } = useProviderAppointments({
    date: selectedDate,
    employee_id: user?.id,
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
            router.push(`/(employee)/agenda/${id}`)
          }
          loading={isLoading}
        />
      </View>
    </SafeAreaView>
  );
}
