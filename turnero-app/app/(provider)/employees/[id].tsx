import { useState, useEffect } from "react";
import { View, Text, SafeAreaView, ScrollView, Alert } from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { Card } from "../../../components/ui/Card";
import {
  useEmployees,
  useUpdateEmployee,
  useDeleteEmployee,
  useServices,
  useAssignServices,
} from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

export default function EmployeeDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data: employees } = useEmployees();
  const { data: services } = useServices();
  const updateEmployee = useUpdateEmployee();
  const deleteEmployee = useDeleteEmployee();
  const assignServices = useAssignServices();

  const employee = employees?.find((e) => e.id === id);

  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [selectedServiceIds, setSelectedServiceIds] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (employee) {
      setName(employee.name);
      setPhone(employee.phone);
    }
  }, [employee]);

  const handleUpdate = async () => {
    setLoading(true);
    try {
      await updateEmployee.mutateAsync({
        id,
        name: name.trim(),
        phone: phone.trim(),
      });
      if (selectedServiceIds.length > 0) {
        await assignServices.mutateAsync({
          id,
          service_ids: selectedServiceIds,
        });
      }
      Alert.alert("Guardado", "Empleado actualizado correctamente.");
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleToggleActive = async () => {
    if (!employee) return;
    try {
      await updateEmployee.mutateAsync({
        id,
        is_active: !employee.is_active,
      });
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    }
  };

  const handleDelete = () => {
    Alert.alert("Eliminar empleado", "¿Estás seguro?", [
      { text: "No" },
      {
        text: "Sí",
        style: "destructive",
        onPress: async () => {
          try {
            await deleteEmployee.mutateAsync(id);
            router.back();
          } catch (err) {
            Alert.alert("Error", getApiErrorMessage(err));
          }
        },
      },
    ]);
  };

  const toggleService = (serviceId: string) => {
    setSelectedServiceIds((prev) =>
      prev.includes(serviceId)
        ? prev.filter((s) => s !== serviceId)
        : [...prev, serviceId]
    );
  };

  if (!employee) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Empleado no encontrado</Text>
      </View>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-6">
          {employee.name}
        </Text>

        <Input label="Nombre" value={name} onChangeText={setName} />
        <Input label="Teléfono" value={phone} onChangeText={setPhone} keyboardType="phone-pad" />

        {services && services.length > 0 ? (
          <View className="mb-4">
            <Text className="text-sm font-medium text-gray-700 mb-2">
              Servicios asignados
            </Text>
            <View className="flex-row flex-wrap gap-2">
              {services.map((svc) => (
                <Button
                  key={svc.id}
                  title={svc.name}
                  variant={
                    selectedServiceIds.includes(svc.id) ? "primary" : "outline"
                  }
                  onPress={() => toggleService(svc.id)}
                />
              ))}
            </View>
          </View>
        ) : null}

        <Button title="Guardar cambios" onPress={handleUpdate} loading={loading} />

        <Card className="mt-4 mb-2">
          <View className="flex-row justify-between items-center">
            <Text className="text-sm text-gray-600">
              Estado: {employee.is_active ? "Activo" : "Inactivo"}
            </Text>
            <Button
              title={employee.is_active ? "Desactivar" : "Activar"}
              variant={employee.is_active ? "secondary" : "primary"}
              onPress={handleToggleActive}
            />
          </View>
        </Card>

        <Button
          title="Eliminar empleado"
          variant="danger"
          onPress={handleDelete}
          className="mt-2 mb-8"
        />
      </ScrollView>
    </SafeAreaView>
  );
}
