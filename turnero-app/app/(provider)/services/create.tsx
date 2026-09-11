import { useState } from "react";
import { View, Text, SafeAreaView, ScrollView, Alert } from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { useCreateService } from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

export default function CreateServiceScreen() {
  const router = useRouter();
  const createService = useCreateService();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [duration, setDuration] = useState("30");
  const [loading, setLoading] = useState(false);

  const handleCreate = async () => {
    if (!name.trim()) {
      Alert.alert("Error", "El nombre es obligatorio.");
      return;
    }
    setLoading(true);
    try {
      await createService.mutateAsync({
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

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-6">
          Nuevo servicio
        </Text>
        <Input label="Nombre" placeholder="Corte de pelo" value={name} onChangeText={setName} />
        <Input
          label="Descripción (opcional)"
          placeholder="Descripción del servicio"
          value={description}
          onChangeText={setDescription}
          multiline
        />
        <Input
          label="Duración (minutos)"
          placeholder="30"
          value={duration}
          onChangeText={setDuration}
          keyboardType="number-pad"
        />
        <Button title="Crear servicio" onPress={handleCreate} loading={loading} />
      </ScrollView>
    </SafeAreaView>
  );
}
