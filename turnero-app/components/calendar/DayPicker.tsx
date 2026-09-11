import { View, Text, TouchableOpacity, ScrollView } from "react-native";
import { useMemo } from "react";

interface DayPickerProps {
  selectedDate: string; // YYYY-MM-DD
  onSelectDate: (date: string) => void;
  daysToShow?: number;
}

const DAY_NAMES = ["Dom", "Lun", "Mar", "Mié", "Jue", "Vie", "Sáb"];
const MONTH_NAMES = [
  "Enero",
  "Febrero",
  "Marzo",
  "Abril",
  "Mayo",
  "Junio",
  "Julio",
  "Agosto",
  "Septiembre",
  "Octubre",
  "Noviembre",
  "Diciembre",
];

function formatDate(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

export function DayPicker({
  selectedDate,
  onSelectDate,
  daysToShow = 14,
}: DayPickerProps) {
  const days = useMemo(() => {
    const result: Date[] = [];
    const today = new Date();
    for (let i = 0; i < daysToShow; i++) {
      const d = new Date(today);
      d.setDate(today.getDate() + i);
      result.push(d);
    }
    return result;
  }, [daysToShow]);

  const selectedObj = selectedDate ? new Date(selectedDate + "T00:00:00") : null;
  const headerMonth = selectedObj
    ? `${MONTH_NAMES[selectedObj.getMonth()]} ${selectedObj.getFullYear()}`
    : `${MONTH_NAMES[days[0].getMonth()]} ${days[0].getFullYear()}`;

  return (
    <View>
      <Text className="text-lg font-semibold text-gray-800 mb-3">
        {headerMonth}
      </Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false}>
        <View className="flex-row gap-2">
          {days.map((day) => {
            const dateStr = formatDate(day);
            const isSelected = dateStr === selectedDate;
            return (
              <TouchableOpacity
                key={dateStr}
                onPress={() => onSelectDate(dateStr)}
                className={`w-14 h-20 rounded-xl items-center justify-center ${
                  isSelected ? "bg-primary-600" : "bg-gray-100"
                }`}
              >
                <Text
                  className={`text-xs ${
                    isSelected ? "text-primary-200" : "text-gray-500"
                  }`}
                >
                  {DAY_NAMES[day.getDay()]}
                </Text>
                <Text
                  className={`text-lg font-bold mt-1 ${
                    isSelected ? "text-white" : "text-gray-800"
                  }`}
                >
                  {day.getDate()}
                </Text>
              </TouchableOpacity>
            );
          })}
        </View>
      </ScrollView>
    </View>
  );
}
