import { useState } from "react";
import { View, Text, ScrollView, Alert, SafeAreaView } from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { useCreateWalkIn, useEmployees } from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

function formatToday(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}

function formatNow(): string {
  const d = new Date();
  return `${String(d.getHours()).padStart(2, "0")}:${String(d.getMinutes()).padStart(2, "0")}`;
}

export default function WalkInScreen() {
  const router = useRouter();
  const createWalkIn = useCreateWalkIn();
  const { data: employees } = useEmployees();

  const [clientName, setClientName] = useState("");
  const [clientPhone, setClientPhone] = useState("");
  const [startTime, setStartTime] = useState(formatNow());
  const [selectedEmployeeId, setSelectedEmployeeId] = useState("");
  const [loading, setLoading] = useState(false);

  const handleCreate = async () => {
    if (!clientName.trim() || !clientPhone.trim() || !selectedEmployeeId) {
      Alert.alert("Error", "Completá todos los campos obligatorios.");
      return;
    }

    setLoading(true);
    try {
      await createWalkIn.mutateAsync({
        employee_id: selectedEmployeeId,
        client_name: clientName.trim(),
        client_phone: clientPhone.trim(),
        date: formatToday(),
        start_time: startTime,
      });
      router.back();
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-6">
          Turno sin cita
        </Text>

        <Input
          label="Nombre del cliente"
          placeholder="Juan Pérez"
          value={clientName}
          onChangeText={setClientName}
        />
        <Input
          label="Teléfono"
          placeholder="+54 11 1234 5678"
          value={clientPhone}
          onChangeText={setClientPhone}
          keyboardType="phone-pad"
        />
        <Input
          label="Hora de inicio"
          placeholder="HH:MM"
          value={startTime}
          onChangeText={setStartTime}
        />

        <Text className="text-sm font-medium text-gray-700 mb-2">
          Empleado
        </Text>
        <View className="flex-row flex-wrap gap-2 mb-6">
          {(employees ?? [])
            .filter((e) => e.is_active)
            .map((emp) => (
              <Button
                key={emp.id}
                title={emp.name}
                variant={
                  selectedEmployeeId === emp.id ? "primary" : "outline"
                }
                onPress={() => setSelectedEmployeeId(emp.id)}
              />
            ))}
        </View>

        <Button title="Crear turno" onPress={handleCreate} loading={loading} />
      </ScrollView>
    </SafeAreaView>
  );
}
