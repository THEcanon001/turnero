import { useState, useEffect } from "react";
import { View, Text } from "react-native";
import { useRouter, Stack } from "expo-router";
import { storage } from "../../../lib/storage";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";

const PHONE_KEY = "turnero_client_phone";

export default function AppointmentsListScreen() {
  const router = useRouter();
  const [phone, setPhone] = useState("");
  const [searchPhone, setSearchPhone] = useState("");

  useEffect(() => {
    storage.getItem(PHONE_KEY).then((saved) => {
      if (saved) {
        setPhone(saved);
        setSearchPhone(saved);
      }
    });
  }, []);

  const handleSearch = async () => {
    if (phone.trim()) {
      setSearchPhone(phone.trim());
      await storage.setItem(PHONE_KEY, phone.trim());
    }
  };

  return (
    <View className="flex-1">
      <Stack.Screen options={{ headerShown: true, title: "Mis turnos" }} />
      <View className="p-4">
        <Text className="text-base text-gray-600 mb-4">
          Para ver tus turnos, ingresá el teléfono con el que agendaste.
        </Text>
        <Input
          placeholder="+54 11 1234 5678"
          value={phone}
          onChangeText={setPhone}
          keyboardType="phone-pad"
        />
        <Button title="Buscar mis turnos" onPress={handleSearch} />

        <View className="mt-4">
          <Text className="text-sm font-medium text-gray-700 mb-2">
            ¿Tenés el ID de tu turno?
          </Text>
          <Input
            placeholder="ID del turno"
            onSubmitEditing={(e) => {
              const id = e.nativeEvent.text.trim();
              if (id) router.push(`/(client)/appointments/${id}`);
            }}
            returnKeyType="go"
          />
        </View>

        {searchPhone ? (
          <View className="mt-6">
            <Text className="text-sm text-gray-400 text-center">
              La búsqueda de turnos por teléfono estará disponible próximamente.
            </Text>
          </View>
        ) : null}
      </View>
    </View>
  );
}
