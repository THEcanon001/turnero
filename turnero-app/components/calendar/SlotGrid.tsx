import { View, Text, TouchableOpacity } from "react-native";
import type { Slot } from "../../types/api";

interface SlotGridProps {
  slots: Slot[];
  selectedSlot: string | null; // start_time
  onSelectSlot: (startTime: string) => void;
  loading?: boolean;
}

export function SlotGrid({
  slots,
  selectedSlot,
  onSelectSlot,
  loading,
}: SlotGridProps) {
  if (loading) {
    return (
      <View className="py-8 items-center">
        <Text className="text-gray-400">Cargando horarios...</Text>
      </View>
    );
  }

  if (slots.length === 0) {
    return (
      <View className="py-8 items-center">
        <Text className="text-gray-400">
          No hay horarios disponibles para esta fecha.
        </Text>
      </View>
    );
  }

  const availableSlots = slots.filter((s) => s.available);

  if (availableSlots.length === 0) {
    return (
      <View className="py-8 items-center">
        <Text className="text-gray-400">
          Todos los horarios están ocupados.
        </Text>
      </View>
    );
  }

  return (
    <View className="flex-row flex-wrap gap-2">
      {slots.map((slot) => {
        const isSelected = slot.start_time === selectedSlot;
        const isAvailable = slot.available;
        return (
          <TouchableOpacity
            key={slot.start_time}
            onPress={() => isAvailable && onSelectSlot(slot.start_time)}
            disabled={!isAvailable}
            className={`px-4 py-3 rounded-xl min-w-[80px] items-center ${
              isSelected
                ? "bg-primary-600"
                : isAvailable
                  ? "bg-gray-100"
                  : "bg-gray-50 opacity-40"
            }`}
          >
            <Text
              className={`text-sm font-medium ${
                isSelected
                  ? "text-white"
                  : isAvailable
                    ? "text-gray-800"
                    : "text-gray-400 line-through"
              }`}
            >
              {slot.start_time}
            </Text>
          </TouchableOpacity>
        );
      })}
    </View>
  );
}
