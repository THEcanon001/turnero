import { useState, useEffect } from "react";
import { View, Text, SafeAreaView, ScrollView, Alert, Switch } from "react-native";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { Card } from "../../../components/ui/Card";
import {
  useProviderMe,
  useEmployees,
  useSchedules,
  useSetSchedule,
  useScheduleExceptions,
  useAddException,
  useDeleteException,
} from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";
import type { DaySchedule } from "../../../types/api";

const DAY_NAMES = [
  "Domingo",
  "Lunes",
  "Martes",
  "Miércoles",
  "Jueves",
  "Viernes",
  "Sábado",
];

const DEFAULT_DAYS: DaySchedule[] = Array.from({ length: 7 }, (_, i) => ({
  day_of_week: i,
  start_time: i >= 1 && i <= 5 ? "09:00" : "",
  end_time: i >= 1 && i <= 5 ? "18:00" : "",
  is_active: i >= 1 && i <= 5,
}));

export default function ScheduleScreen() {
  const { data: provider } = useProviderMe();
  const { data: employees } = useEmployees();

  // For individual providers, use first employee (self)
  const selfEmployee = employees?.[0];
  const employeeId = selfEmployee?.id ?? "";

  const { data: schedule } = useSchedules(employeeId);
  const { data: exceptions } = useScheduleExceptions(employeeId);
  const setSchedule = useSetSchedule();
  const addException = useAddException();
  const deleteException = useDeleteException();

  const [days, setDays] = useState<DaySchedule[]>(DEFAULT_DAYS);
  const [slotDuration, setSlotDuration] = useState("30");
  const [breakMinutes, setBreakMinutes] = useState("0");
  const [excDate, setExcDate] = useState("");
  const [excReason, setExcReason] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (schedule) {
      setDays(schedule.days);
      setSlotDuration(String(schedule.slot_duration_minutes));
      setBreakMinutes(String(schedule.break_after_slot_minutes));
    }
  }, [schedule]);

  const updateDay = (index: number, field: keyof DaySchedule, value: string | boolean) => {
    setDays((prev) => {
      const updated = [...prev];
      updated[index] = { ...updated[index], [field]: value };
      return updated;
    });
  };

  const handleSave = async () => {
    if (!employeeId) return;
    setLoading(true);
    try {
      await setSchedule.mutateAsync({
        employeeId,
        slot_duration_minutes: parseInt(slotDuration, 10) || 30,
        break_after_slot_minutes: parseInt(breakMinutes, 10) || 0,
        days,
      });
      Alert.alert("Guardado", "Horario actualizado.");
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleAddException = async () => {
    if (!employeeId || !excDate.trim()) {
      Alert.alert("Error", "Ingresá una fecha.");
      return;
    }
    try {
      await addException.mutateAsync({
        employeeId,
        date: excDate.trim(),
        is_day_off: true,
        reason: excReason.trim() || null,
      });
      setExcDate("");
      setExcReason("");
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-4">Horarios</Text>

        <View className="flex-row gap-3 mb-4">
          <View className="flex-1">
            <Input
              label="Duración turno (min)"
              value={slotDuration}
              onChangeText={setSlotDuration}
              keyboardType="number-pad"
            />
          </View>
          <View className="flex-1">
            <Input
              label="Descanso entre turnos (min)"
              value={breakMinutes}
              onChangeText={setBreakMinutes}
              keyboardType="number-pad"
            />
          </View>
        </View>

        {days.map((day, index) => (
          <Card key={day.day_of_week} className="mb-2">
            <View className="flex-row items-center justify-between">
              <View className="flex-row items-center flex-1">
                <Switch
                  value={day.is_active}
                  onValueChange={(v) => updateDay(index, "is_active", v)}
                />
                <Text className="text-base text-gray-800 ml-2">
                  {DAY_NAMES[day.day_of_week]}
                </Text>
              </View>
              {day.is_active ? (
                <View className="flex-row items-center gap-1">
                  <Input
                    value={day.start_time}
                    onChangeText={(v) => updateDay(index, "start_time", v)}
                    placeholder="09:00"
                    className="w-20 mb-0"
                  />
                  <Text className="text-gray-400">-</Text>
                  <Input
                    value={day.end_time}
                    onChangeText={(v) => updateDay(index, "end_time", v)}
                    placeholder="18:00"
                    className="w-20 mb-0"
                  />
                </View>
              ) : null}
            </View>
          </Card>
        ))}

        <Button
          title="Guardar horario"
          onPress={handleSave}
          loading={loading}
          className="mt-4 mb-6"
        />

        <Text className="text-lg font-semibold text-gray-800 mb-3">
          Excepciones (días libres)
        </Text>

        <View className="flex-row gap-2 mb-3">
          <View className="flex-1">
            <Input
              placeholder="YYYY-MM-DD"
              value={excDate}
              onChangeText={setExcDate}
            />
          </View>
          <View className="flex-1">
            <Input
              placeholder="Motivo (opcional)"
              value={excReason}
              onChangeText={setExcReason}
            />
          </View>
        </View>
        <Button
          title="Agregar excepción"
          variant="outline"
          onPress={handleAddException}
          className="mb-4"
        />

        {exceptions?.map((exc) => (
          <Card key={exc.id} className="mb-2">
            <View className="flex-row justify-between items-center">
              <View>
                <Text className="text-sm font-medium text-gray-800">
                  {exc.date}
                </Text>
                {exc.reason ? (
                  <Text className="text-xs text-gray-500">{exc.reason}</Text>
                ) : null}
              </View>
              <Button
                title="Eliminar"
                variant="danger"
                onPress={() => deleteException.mutate(exc.id)}
              />
            </View>
          </Card>
        ))}

        <View className="h-8" />
      </ScrollView>
    </SafeAreaView>
  );
}
