import { useState, useEffect } from "react";
import { View, Text, SafeAreaView, ScrollView, Alert } from "react-native";
import { useLocalSearchParams, useRouter } from "expo-router";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { useServices, useUpdateService, useDeleteService } from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

export default function EditServiceScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const router = useRouter();
  const { data: services } = useServices();
  const updateService = useUpdateService();
  const deleteService = useDeleteService();

  const service = services?.find((s) => s.id === id);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [duration, setDuration] = useState("30");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (service) {
      setName(service.name);
      setDescription(service.description ?? "");
      setDuration(String(service.duration_minutes));
    }
  }, [service]);

  const handleUpdate = async () => {
    if (!name.trim()) {
      Alert.alert("Error", "El nombre es obligatorio.");
      return;
    }
    setLoading(true);
    try {
      await updateService.mutateAsync({
        id,
        name: name.trim(),
        description: description.trim() || null,
        duration_minutes: parseInt(duration, 10) || 30,
      });
      router.back();
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = () => {
    Alert.alert("Eliminar servicio", "¿Estás seguro?", [
      { text: "No" },
      {
        text: "Sí, eliminar",
        style: "destructive",
        onPress: async () => {
          try {
            await deleteService.mutateAsync(id);
            router.back();
          } catch (err) {
            Alert.alert("Error", getApiErrorMessage(err));
          }
        },
      },
    ]);
  };

  if (!service) {
    return (
      <View className="flex-1 items-center justify-center">
        <Text className="text-gray-400">Servicio no encontrado</Text>
      </View>
    );
  }

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-6">
          Editar servicio
        </Text>
        <Input label="Nombre" value={name} onChangeText={setName} />
        <Input
          label="Descripción"
          value={description}
          onChangeText={setDescription}
          multiline
        />
        <Input
          label="Duración (minutos)"
          value={duration}
          onChangeText={setDuration}
          keyboardType="number-pad"
        />
        <Button title="Guardar cambios" onPress={handleUpdate} loading={loading} />
        <Button
          title="Eliminar servicio"
          variant="danger"
          onPress={handleDelete}
          className="mt-3"
        />
      </ScrollView>
    </SafeAreaView>
  );
}
