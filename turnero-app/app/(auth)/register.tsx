import { useState } from "react";
import {
  View,
  Text,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from "react-native";
import { Link } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";
import { useAuth } from "../../hooks/useAuth";
import { getApiErrorMessage } from "../../lib/api";

type AccountType = "individual" | "business";

export default function RegisterScreen() {
  const { register } = useAuth();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [phone, setPhone] = useState("");
  const [slug, setSlug] = useState("");
  const [accountType, setAccountType] = useState<AccountType>("individual");
  const [businessName, setBusinessName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleRegister = async () => {
    if (!name.trim() || !email.trim() || !password || !phone.trim() || !slug.trim()) {
      setError("Completá todos los campos obligatorios.");
      return;
    }
    if (password.length < 8) {
      setError("La contraseña debe tener al menos 8 caracteres.");
      return;
    }
    if (accountType === "business" && !businessName.trim()) {
      setError("Ingresá el nombre de tu negocio.");
      return;
    }

    setError("");
    setLoading(true);
    try {
      await register({
        name: name.trim(),
        email: email.trim(),
        password,
        phone: phone.trim(),
        type: accountType,
        slug: slug.trim().toLowerCase(),
        business_name:
          accountType === "business" ? businessName.trim() : null,
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      });
    } catch (err) {
      setError(getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView
      className="flex-1"
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <ScrollView
        contentContainerClassName="py-12 px-6"
        keyboardShouldPersistTaps="handled"
      >
        <View className="mb-8">
          <Text className="text-3xl font-bold text-gray-900">
            Crear cuenta
          </Text>
          <Text className="text-base text-gray-500 mt-2">
            Registrate para gestionar tus turnos
          </Text>
        </View>

        {error ? (
          <View className="bg-red-50 border border-red-200 rounded-xl p-3 mb-4">
            <Text className="text-red-600 text-sm">{error}</Text>
          </View>
        ) : null}

        <Input
          label="Nombre"
          placeholder="Tu nombre completo"
          value={name}
          onChangeText={setName}
        />

        <Input
          label="Email"
          placeholder="tu@email.com"
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          autoCapitalize="none"
          autoCorrect={false}
        />

        <Input
          label="Contraseña"
          placeholder="Mínimo 8 caracteres"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
        />

        <Input
          label="Teléfono"
          placeholder="+54 11 1234 5678"
          value={phone}
          onChangeText={setPhone}
          keyboardType="phone-pad"
        />

        <Input
          label="Slug (URL de tu perfil)"
          placeholder="mi-negocio"
          value={slug}
          onChangeText={(text) =>
            setSlug(text.toLowerCase().replace(/[^a-z0-9-]/g, ""))
          }
          autoCapitalize="none"
          autoCorrect={false}
        />

        {/* Account type selector */}
        <Text className="text-sm font-medium text-gray-700 mb-2">
          Tipo de cuenta
        </Text>
        <View className="flex-row gap-3 mb-4">
          <Button
            title="Personal"
            variant={accountType === "individual" ? "primary" : "outline"}
            onPress={() => setAccountType("individual")}
            className="flex-1"
          />
          <Button
            title="Negocio"
            variant={accountType === "business" ? "primary" : "outline"}
            onPress={() => setAccountType("business")}
            className="flex-1"
          />
        </View>

        {accountType === "business" && (
          <Input
            label="Nombre del negocio"
            placeholder="Mi Peluquería"
            value={businessName}
            onChangeText={setBusinessName}
          />
        )}

        <Button
          title="Registrarme"
          onPress={handleRegister}
          loading={loading}
          className="mt-2"
        />

        <View className="flex-row justify-center mt-6">
          <Text className="text-gray-500">¿Ya tenés cuenta? </Text>
          <Link href="/(auth)/login" asChild>
            <Text className="text-primary-600 font-semibold">
              Iniciá sesión
            </Text>
          </Link>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
