import { useState } from "react";
import { View, Text, SafeAreaView, Alert } from "react-native";
import { Button } from "../../../components/ui/Button";
import { Card } from "../../../components/ui/Card";
import { useCreateInvitation } from "../../../hooks/useApi";
import { getApiErrorMessage } from "../../../lib/api";

export default function InviteEmployeeScreen() {
  const createInvitation = useCreateInvitation();
  const [code, setCode] = useState("");
  const [loading, setLoading] = useState(false);

  const handleCreate = async () => {
    setLoading(true);
    try {
      const result = await createInvitation.mutateAsync({ role: "employee" });
      setCode(result.code);
    } catch (err) {
      Alert.alert("Error", getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <SafeAreaView className="flex-1 bg-white px-4 pt-8">
      <Text className="text-2xl font-bold text-gray-800 mb-4">
        Invitar empleado
      </Text>
      <Text className="text-base text-gray-500 mb-6">
        Generá un código de invitación para que tu empleado se una.
      </Text>

      {code ? (
        <Card className="mb-6 items-center">
          <Text className="text-sm text-gray-500 mb-2">
            Código de invitación
          </Text>
          <Text className="text-3xl font-bold text-primary-600 tracking-widest">
            {code}
          </Text>
          <Text className="text-xs text-gray-400 mt-2">
            Compartí este código con tu empleado
          </Text>
        </Card>
      ) : null}

      <Button
        title={code ? "Generar otro código" : "Generar código"}
        onPress={handleCreate}
        loading={loading}
      />
    </SafeAreaView>
  );
}
