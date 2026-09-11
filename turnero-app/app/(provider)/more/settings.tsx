import { useState, useEffect } from "react";
import { View, Text, SafeAreaView, ScrollView, Alert } from "react-native";
import { useRouter } from "expo-router";
import { Input } from "../../../components/ui/Input";
import { Button } from "../../../components/ui/Button";
import { useProviderMe, useUpdateProvider } from "../../../hooks/useApi";
import { useAuth } from "../../../hooks/useAuth";
import { getApiErrorMessage } from "../../../lib/api";

export default function SettingsScreen() {
  const router = useRouter();
  const { data: provider } = useProviderMe();
  const updateProvider = useUpdateProvider();
  const { logout } = useAuth();

  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [address, setAddress] = useState("");
  const [timezone, setTimezone] = useState("");
  const [businessName, setBusinessName] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (provider) {
      setName(provider.name);
      setPhone(provider.phone);
      setAddress(provider.address ?? "");
      setTimezone(provider.timezone);
      setBusinessName(provider.business_name ?? "");
    }
  }, [provider]);

  const handleSave = async () => {
    setLoading(true);
    try {
      await updateProvider.mutateAsync({
        name: name.trim(),
        phone: phone.trim(),
        address: address.trim() || null,
        timezone: timezone.trim(),
        business_name: businessName.trim() || null,
      });
      Alert.alert("Guardado", "Perfil actualizado correctamente.");
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    Alert.alert("Cerrar sesión", "¿Estás seguro?", [
      { text: "No" },
      {
        text: "Sí",
        onPress: async () => {
          await logout();
          router.replace("/(auth)/login");
        },
      },
    ]);
  };

  return (
    <SafeAreaView className="flex-1 bg-white">
      <ScrollView className="flex-1 px-4 pt-4" keyboardShouldPersistTaps="handled">
        <Text className="text-2xl font-bold text-gray-800 mb-6">
          Configuración
        </Text>

        <Input label="Nombre" value={name} onChangeText={setName} />
        <Input
          label="Teléfono"
          value={phone}
          onChangeText={setPhone}
          keyboardType="phone-pad"
        />
        <Input label="Dirección" value={address} onChangeText={setAddress} />
        <Input
          label="Zona horaria"
          value={timezone}
          onChangeText={setTimezone}
        />
        <Input
          label="Nombre del negocio"
          value={businessName}
          onChangeText={setBusinessName}
        />

        <Button
          title="Guardar cambios"
          onPress={handleSave}
          loading={loading}
        />

        <Button
          title="Cerrar sesión"
          variant="danger"
          onPress={handleLogout}
          className="mt-8 mb-8"
        />
      </ScrollView>
    </SafeAreaView>
  );
}
